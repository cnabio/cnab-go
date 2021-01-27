package kubernetes

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	batchclientv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// load credential helpers
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/driver"
)

const (
	k8sContainerName    = "invocation"
	k8sFileSecretVolume = "files"
	numBackoffLoops     = 6
	cnabPrefix          = "cnab.io/"
)

var (
	dns1123Reg = regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*`)
)

// Driver runs an invocation image in a Kubernetes cluster.
type Driver struct {
	Namespace             string
	ServiceAccountName    string
	Annotations           map[string]string
	LimitCPU              resource.Quantity
	LimitMemory           resource.Quantity
	Tolerations           []v1.Toleration
	ActiveDeadlineSeconds int64
	BackoffLimit          int32
	SkipCleanup           bool
	skipJobStatusCheck    bool
	jobs                  batchclientv1.JobInterface
	secrets               coreclientv1.SecretInterface
	pods                  coreclientv1.PodInterface
	deletionPolicy        metav1.DeletionPropagation
	requiredCompletions   int32
}

// New initializes a Kubernetes driver.
func New(namespace, serviceAccount string, conf *rest.Config) (*Driver, error) {
	driver := &Driver{
		Namespace:          namespace,
		ServiceAccountName: serviceAccount,
	}
	driver.setDefaults()
	err := driver.setClient(conf)
	return driver, err
}

// Handles receives an ImageType* and answers whether this driver supports that type.
func (k *Driver) Handles(imagetype string) bool {
	return imagetype == driver.ImageTypeDocker || imagetype == driver.ImageTypeOCI
}

// Config returns the Kubernetes driver configuration options.
func (k *Driver) Config() map[string]string {
	return map[string]string{
		"IN_CLUSTER":      "Connect to the cluster using in-cluster environment variables",
		"CLEANUP_JOBS":    "If true, the job and associated secrets will be destroyed when it finishes running. If false, it will not be destroyed. The supported values are true and false. Defaults to true.",
		"KUBE_NAMESPACE":  "Kubernetes namespace in which to run the invocation image",
		"SERVICE_ACCOUNT": "Kubernetes service account to be mounted by the invocation image (if empty, no service account token will be mounted)",
		"KUBECONFIG":      "Absolute path to the kubeconfig file",
		"MASTER_URL":      "Kubernetes master endpoint",
	}
}

// SetConfig sets Kubernetes driver configuration.
func (k *Driver) SetConfig(settings map[string]string) error {
	k.setDefaults()
	k.Namespace = settings["KUBE_NAMESPACE"]
	k.ServiceAccountName = settings["SERVICE_ACCOUNT"]

	cleanup, err := strconv.ParseBool(settings["CLEANUP_JOBS"])
	if err != nil {
		k.SkipCleanup = !cleanup
	}

	var conf *rest.Config
	if incluster, _ := strconv.ParseBool(settings["IN_CLUSTER"]); incluster {
		conf, err = rest.InClusterConfig()
		if err != nil {
			return errors.Wrap(err, "error retrieving in-cluster kubernetes configuration")
		}
	} else {
		var kubeconfig string
		if kpath := settings["KUBECONFIG"]; kpath != "" {
			kubeconfig = kpath
		} else if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		conf, err = clientcmd.BuildConfigFromFlags(settings["MASTER_URL"], kubeconfig)
		if err != nil {
			return errors.Wrapf(err, "error retrieving external kubernetes configuration using configuration:\n%v", settings)
		}
	}

	return k.setClient(conf)
}

func (k *Driver) setDefaults() {
	k.SkipCleanup = false
	k.BackoffLimit = 0
	k.ActiveDeadlineSeconds = 300
	k.requiredCompletions = 1
	k.deletionPolicy = metav1.DeletePropagationBackground
}

func (k *Driver) setClient(conf *rest.Config) error {
	coreClient, err := coreclientv1.NewForConfig(conf)
	if err != nil {
		return errors.Wrap(err, "error creating CoreClient for Kubernetes Driver")
	}
	batchClient, err := batchclientv1.NewForConfig(conf)
	if err != nil {
		return errors.Wrap(err, "error creating BatchClient for Kubernetes Driver")
	}
	k.jobs = batchClient.Jobs(k.Namespace)
	k.secrets = coreClient.Secrets(k.Namespace)
	k.pods = coreClient.Pods(k.Namespace)

	return nil
}

// Run executes the operation inside of the invocation image.
func (k *Driver) Run(op *driver.Operation) (driver.OperationResult, error) {
	if k.Namespace == "" {
		return driver.OperationResult{}, fmt.Errorf("KUBE_NAMESPACE is required")
	}

	meta := metav1.ObjectMeta{
		Namespace:    k.Namespace,
		GenerateName: generateNameTemplate(op),
		Labels: map[string]string{
			"cnab.io/driver": "kubernetes",
		},
		Annotations: generateMergedAnnotations(op, k.Annotations),
	}
	// Mount SA token if a non-zero value for ServiceAccountName has been specified
	mountServiceAccountToken := k.ServiceAccountName != ""
	job := &batchv1.Job{
		ObjectMeta: meta,
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &k.ActiveDeadlineSeconds,
			Completions:           &k.requiredCompletions,
			BackoffLimit:          &k.BackoffLimit,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      meta.Labels,
					Annotations: meta.Annotations,
				},
				Spec: v1.PodSpec{
					ServiceAccountName:           k.ServiceAccountName,
					AutomountServiceAccountToken: &mountServiceAccountToken,
					RestartPolicy:                v1.RestartPolicyNever,
					Tolerations:                  k.Tolerations,
				},
			},
		},
	}
	img, err := imageWithDigest(op.Image)
	if err != nil {
		return driver.OperationResult{}, err
	}

	container := v1.Container{
		Name:    k8sContainerName,
		Image:   img,
		Command: []string{"/cnab/app/run"},
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU:    k.LimitCPU,
				v1.ResourceMemory: k.LimitMemory,
			},
		},
		ImagePullPolicy: v1.PullIfNotPresent,
	}

	if len(op.Environment) > 0 {
		secret := &v1.Secret{
			ObjectMeta: meta,
			StringData: op.Environment,
		}
		secret.ObjectMeta.GenerateName += "env-"
		secret, err := k.secrets.Create(secret)
		if err != nil {
			return driver.OperationResult{}, err
		}
		if !k.SkipCleanup {
			defer k.deleteSecret(secret.ObjectMeta.Name)
		}

		container.EnvFrom = []v1.EnvFromSource{
			{
				SecretRef: &v1.SecretEnvSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: secret.ObjectMeta.Name,
					},
				},
			},
		}
	}

	if len(op.Files) > 0 {
		secret, mounts := generateFileSecret(op.Files)
		secret.ObjectMeta = meta
		secret.ObjectMeta.GenerateName += "files-"
		secret, err := k.secrets.Create(secret)
		if err != nil {
			return driver.OperationResult{}, err
		}
		if !k.SkipCleanup {
			defer k.deleteSecret(secret.ObjectMeta.Name)
		}

		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, v1.Volume{
			Name: k8sFileSecretVolume,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: secret.ObjectMeta.Name,
				},
			},
		})
		container.VolumeMounts = mounts
	}

	job.Spec.Template.Spec.Containers = []v1.Container{container}
	job, err = k.jobs.Create(job)
	if err != nil {
		return driver.OperationResult{}, err
	}
	if !k.SkipCleanup {
		defer k.deleteJob(job.ObjectMeta.Name)
	}

	// Return early for unit testing purposes (the fake k8s client implementation just
	// hangs during watch because no events are ever created on the Job)
	if k.skipJobStatusCheck {
		return driver.OperationResult{}, nil
	}

	// Create a selector to detect the job just created
	jobSelector := metav1.ListOptions{
		LabelSelector: labels.Set(job.ObjectMeta.Labels).String(),
		FieldSelector: newSingleFieldSelector("metadata.name", job.ObjectMeta.Name),
	}

	// Prevent detecting pods from prior jobs by adding the job name to the labels
	podSelector := metav1.ListOptions{
		LabelSelector: newSingleFieldSelector("job-name", job.ObjectMeta.Name),
	}

	return driver.OperationResult{}, k.watchJobStatusAndLogs(podSelector, jobSelector, op.Out)
}

func (k *Driver) watchJobStatusAndLogs(podSelector metav1.ListOptions, jobSelector metav1.ListOptions, out io.Writer) error {
	// Stream Pod logs in the background
	logsStreamingComplete := make(chan bool)
	err := k.streamPodLogs(podSelector, out, logsStreamingComplete)
	if err != nil {
		return err
	}
	// Watch job events and exit on failure/success
	watch, err := k.jobs.Watch(jobSelector)
	if err != nil {
		return err
	}
	for event := range watch.ResultChan() {
		job, ok := event.Object.(*batchv1.Job)
		if !ok {
			return fmt.Errorf("unexpected type")
		}
		complete := false
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed {
				err = fmt.Errorf(cond.Message)
				complete = true
				break
			}
			if cond.Type == batchv1.JobComplete {
				complete = true
				break
			}
		}
		if complete {
			break
		}
	}

	// Wait for pod logs to finish printing
	for i := 0; i < int(k.requiredCompletions); i++ {
		<-logsStreamingComplete
	}

	return err
}

func (k *Driver) streamPodLogs(options metav1.ListOptions, out io.Writer, done chan bool) error {
	watcher, err := k.pods.Watch(options)
	if err != nil {
		return err
	}

	go func() {
		// Track pods whose logs have been streamed by pod name. We need to know when we've already
		// processed logs for a given pod, since multiple lifecycle events are received per pod.
		streamedLogs := map[string]bool{}
		for event := range watcher.ResultChan() {
			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				continue
			}
			podName := pod.GetName()
			if streamedLogs[podName] {
				// The event was for a pod whose logs have already been streamed, so do nothing.
				continue
			}

			for i := 0; i < numBackoffLoops; i++ {
				time.Sleep(time.Duration(i*i/2) * time.Second)
				req := k.pods.GetLogs(podName, &v1.PodLogOptions{
					Container: k8sContainerName,
					Follow:    true,
				})
				reader, err := req.Stream()
				if err != nil {
					// There was an error connecting to the pod, so continue the loop and attempt streaming
					// the logs again.
					continue
				}

				// Block the loop until all logs from the pod have been processed.
				bytesRead, err := io.Copy(out, reader)
				reader.Close()
				if err != nil {
					continue
				}
				if bytesRead == 0 {
					// There is a chance where we have connected to the pod, but it has yet to write something.
					// In that case, we continue to to keep streaming until it does.
					continue
				}
				// Set the pod to have successfully streamed data.
				streamedLogs[podName] = true
				break
			}

			done <- true
		}
	}()

	return nil
}

func (k *Driver) deleteSecret(name string) error {
	return k.secrets.Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &k.deletionPolicy,
	})
}

func (k *Driver) deleteJob(name string) error {
	return k.jobs.Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &k.deletionPolicy,
	})
}

const maxNameTemplateLength = 50

// generateNameTemplate returns a value suitable for the Kubernetes metav1.ObjectMeta.GenerateName
// field, that includes the operation action and installation names for debugging purposes.
//
// Note that the value returned may be truncated to conform to Kubernetes maximum resource name
// length constraints.
func generateNameTemplate(op *driver.Operation) string {
	const maxLength = maxNameTemplateLength - 1
	name := fmt.Sprintf("%s-%s", op.Action, op.Installation)
	if len(name) > maxLength {
		name = name[0:maxLength]
	}

	var result string
	for _, match := range dns1123Reg.FindAllString(strings.ToLower(name), maxLength) {
		// It's safe to add one character because we've already removed at least one character not matching our regex.
		result += match + "-"
	}

	return result
}

func generateMergedAnnotations(op *driver.Operation, mergeWith map[string]string) map[string]string {
	anno := map[string]string{
		"cnab.io/installation": op.Installation,
		"cnab.io/action":       op.Action,
		"cnab.io/revision":     op.Revision,
	}

	for k, v := range mergeWith {
		if strings.HasPrefix(k, cnabPrefix) {
			log.Printf("Annotations with prefix '%s' are reserved. Annotation '%s: %s' will not be applied.\n", cnabPrefix, k, v)
			continue
		}
		anno[k] = v
	}

	return anno
}

func generateFileSecret(files map[string]string) (*v1.Secret, []v1.VolumeMount) {
	size := len(files)
	data := make(map[string]string, size)
	mounts := make([]v1.VolumeMount, size)

	i := 0
	for path, contents := range files {
		key := strings.Replace(filepath.ToSlash(path), "/", "_", -1)
		data[key] = contents
		mounts[i] = v1.VolumeMount{
			Name:      k8sFileSecretVolume,
			MountPath: path,
			SubPath:   key,
		}
		i++
	}

	secret := &v1.Secret{
		StringData: data,
	}

	return secret, mounts
}

func newSingleFieldSelector(k, v string) string {
	return labels.Set(map[string]string{
		k: v,
	}).String()
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func imageWithDigest(img bundle.InvocationImage) (string, error) {
	// img.Image can be just the name, name:tag or name@digest
	ref, err := reference.ParseNormalizedNamed(img.Image)
	if err != nil {
		return "", errors.Wrapf(err, "could not parse %s as an OCI reference", img.Image)
	}

	var d digest.Digest
	if v, ok := ref.(reference.Digested); ok {
		// Check that the digests match since it's provided twice
		if img.Digest != "" && img.Digest != v.Digest().String() {
			return "", errors.Errorf("The digest %s for the image %s doesn't match the one specified in the image", img.Digest, img.Image)
		}
		d = v.Digest()
	} else if img.Digest != "" {
		d, err = digest.Parse(img.Digest)
		if err != nil {
			return "", errors.Wrapf(err, "invalid digest %s specified for invocation image %s", img.Digest, img.Image)
		}
	}

	// Digest was not supplied anywhere
	if d == "" {
		return img.Image, nil
	}

	digestedRef, err := reference.WithDigest(ref, d)
	return reference.FamiliarString(digestedRef), errors.Wrapf(err, "invalid image digest %s", d.String())
}
