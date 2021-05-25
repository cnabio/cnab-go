// +build integration

package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/client/conditions"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/driver"
)

func TestDriver_Run_Integration(t *testing.T) {
	k := &Driver{}
	k.ActiveDeadlineSeconds = 60

	cases := []struct {
		name                   string
		op                     *driver.Operation
		output                 string
		err                    error
		podAffinityMatchLabels string
	}{
		{
			name: "install",
			op: &driver.Operation{
				Installation: "example",
				Action:       "install",
				Bundle:       &bundle.Bundle{},
				Image: bundle.InvocationImage{
					BaseImage: bundle.BaseImage{
						Image:  "cnab/helloworld",
						Digest: "sha256:55f83710272990efab4e076f9281453e136980becfd879640b06552ead751284",
					},
				},
				Environment: map[string]string{
					"PORT": "3000",
				},
			},
			output: "Port parameter was set to 3000\nInstall action\nAction install complete for example\n",
			err:    nil,
		},
		{
			name: "install with affinity using single label",
			op: &driver.Operation{
				Installation: "example",
				Action:       "install",
				Bundle:       &bundle.Bundle{},
				Image: bundle.InvocationImage{
					BaseImage: bundle.BaseImage{
						Image:  "cnab/helloworld",
						Digest: "sha256:55f83710272990efab4e076f9281453e136980becfd879640b06552ead751284",
					},
				},
				Environment: map[string]string{
					"PORT": "3000",
				},
			},
			output:                 "Port parameter was set to 3000\nInstall action\nAction install complete for example\n",
			err:                    nil,
			podAffinityMatchLabels: "test=true",
		},
		{
			name: "install with affinity using multiple labels",
			op: &driver.Operation{
				Installation: "example",
				Action:       "install",
				Bundle:       &bundle.Bundle{},
				Image: bundle.InvocationImage{
					BaseImage: bundle.BaseImage{
						Image:  "cnab/helloworld",
						Digest: "sha256:55f83710272990efab4e076f9281453e136980becfd879640b06552ead751284",
					},
				},
				Environment: map[string]string{
					"PORT": "3000",
				},
			},
			output:                 "Port parameter was set to 3000\nInstall action\nAction install complete for example\n",
			err:                    nil,
			podAffinityMatchLabels: "test=true test1=true",
		},
		{
			name: "long installation name",
			op: &driver.Operation{
				Installation: "greater-than-300-length-and-special-chars/-*()+%@qcUYSfR9MS3BqR0kRDHe2K5EHJa8BJGrcoiDVvsDpATjIkrk4PWrdysIqFpJzrKHauRWfBjjF889Qdc5DUBQ6gKy8Qezkl9HyCmo88hMrkaeVPxknFt0nWRm0xqYhoaY0Db7ZcljchbBAufVvH5l0T7iBdg1E0iSCTZw0v5rCAEclNwzjpg7DfLq2SBdJ0W8XdyQSWVMpakjraXP9droq8ol70gX0QuqAZDkGtHyxet8Akv9lGCCVVFuY4kBdkW3LDHoxl0xz2EZzXja1GTlYui0Bpx0TGqMLish9tBOhuC7",
				Action:       "install",
				Bundle:       &bundle.Bundle{},
				Image: bundle.InvocationImage{
					BaseImage: bundle.BaseImage{
						Image:  "cnab/helloworld",
						Digest: "sha256:55f83710272990efab4e076f9281453e136980becfd879640b06552ead751284",
					},
				},
				Environment: map[string]string{
					"PORT": "3000",
				},
			},
			output: "Port parameter was set to 3000\nInstall action\nAction install complete for greater-than-300-length-and-special-chars/-*()+%@qcUYSfR9MS3BqR0kRDHe2K5EHJa8BJGrcoiDVvsDpATjIkrk4PWrdysIqFpJzrKHauRWfBjjF889Qdc5DUBQ6gKy8Qezkl9HyCmo88hMrkaeVPxknFt0nWRm0xqYhoaY0Db7ZcljchbBAufVvH5l0T7iBdg1E0iSCTZw0v5rCAEclNwzjpg7DfLq2SBdJ0W8XdyQSWVMpakjraXP9droq8ol70gX0QuqAZDkGtHyxet8Akv9lGCCVVFuY4kBdkW3LDHoxl0xz2EZzXja1GTlYui0Bpx0TGqMLish9tBOhuC7\n",
			err:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var output bytes.Buffer
			tc.op.Out = &output
			if tc.op.Environment == nil {
				tc.op.Environment = map[string]string{}
			}
			tc.op.Environment["CNAB_ACTION"] = tc.op.Action
			tc.op.Environment["CNAB_INSTALLATION_NAME"] = tc.op.Installation

			// Create a volume to share data with the invocation image
			pvc, cleanup := createTestPVC(t, ctx)
			defer cleanup()

			// Simulate mounting the shared volume
			sharedDir, err := ioutil.TempDir("", "cnab-go")
			require.NoError(t, err, "could not create test directory")
			defer os.RemoveAll(sharedDir)

			testNameLabel := getTestNameLabel(tc.name)

			err = k.SetConfig(map[string]string{
				SettingJobVolumePath:          sharedDir,
				SettingJobVolumeName:          pvc,
				SettingKubeNamespace:          "default",
				SettingKubeconfig:             os.Getenv("KUBECONFIG"),
				SettingPodAffinityMatchLabels: tc.podAffinityMatchLabels,
				SettingLabels:                 fmt.Sprintf("testname=%s", testNameLabel),
				SettingCleanupJobs: func() string {
					if tc.podAffinityMatchLabels == "" {
						return "true"
					}
					return "false"
				}(),
			})
			require.NoError(t, err, "SetConfig failed")
			hostname := ""
			var deletePod func()
			if tc.podAffinityMatchLabels != "" {
				hostname, deletePod = createTestPod(t, ctx, tc.podAffinityMatchLabels)
				defer deletePod()
			}

			_, err = k.Run(tc.op)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Contains(t, output.String(), tc.output)
			if hostname != "" {
				checkPodAffinity(t, ctx, testNameLabel, hostname)
			}

		})
	}
}

func getTestNameLabel(testName string) string {
	return fmt.Sprintf("%s.%d", strings.ReplaceAll(testName, " ", "."), time.Now().Unix())
}

func checkPodAffinity(t *testing.T, ctx context.Context, testname string, hostname string) {
	coreClient := getCoreClient(t)
	podClient := coreClient.Pods("default")
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"testname": testname}}
	pods, err := podClient.List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	require.NoError(t, err, "List Pods by label failed %v", pods)
	require.Equal(t, 1, len(pods.Items), "Only expected one pod with label: %s", testname)
	driverHostName := getPodNodeName(t, ctx, pods.Items[0].Name)
	require.Equal(t, hostname, driverHostName, "Pod hostname expected:%s actual:%s", hostname, driverHostName)
}

func createTestPod(t *testing.T, ctx context.Context, matchLabels string) (string, func()) {
	labels := make(map[string]string)
	for _, i := range strings.Split(matchLabels, " ") {
		kv := strings.Split(i, "=")
		labels[kv[0]] = kv[1]
	}

	podDefinition := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "affinity-test-pod",
			Namespace:    "default",
			Labels:       labels,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "affinity-test-container",
					Image:   "alpine",
					Command: []string{"tail", "-f", "/dev/null"},
				},
			},
			RestartPolicy: "Always",
		},
	}

	coreClient := getCoreClient(t)
	podClient := coreClient.Pods("default")
	pod, err := podClient.Create(ctx, podDefinition, metav1.CreateOptions{})
	require.NoError(t, err, "Create pod failed")

	return getPodNodeName(t, ctx, pod.Name), func() {
		podClient.Delete(ctx, pod.Name, metav1.DeleteOptions{})
	}

}

func getPodNodeName(t *testing.T, ctx context.Context, podName string) string {
	coreClient := getCoreClient(t)
	podClient := coreClient.Pods("default")
	wait.PollImmediate(time.Second, time.Second*60, func() (bool, error) {
		pod, err := podClient.Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, conditions.ErrPodCompleted
		}
		return false, nil
	})

	pod, err := podClient.Get(ctx, podName, metav1.GetOptions{})
	require.NoError(t, err, "Get running pod failed")

	return pod.Spec.NodeName

}

func createTestPVC(t *testing.T, ctx context.Context) (string, func()) {
	pvcDefinition := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cnab-driver-shared",
			Namespace:    "default",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{Requests: map[v1.ResourceName]resource.Quantity{
				v1.ResourceStorage: resource.MustParse("64Mi"),
			}},
		},
	}
	coreClient := getCoreClient(t)
	pvcClient := coreClient.PersistentVolumeClaims("default")
	pvc, err := pvcClient.Create(ctx, pvcDefinition, metav1.CreateOptions{})
	require.NoError(t, err, "create pvc failed")

	return pvc.Name, func() {
		pvcClient.Delete(ctx, pvc.Name, metav1.DeleteOptions{})
	}
}

func getCoreClient(t *testing.T) *coreclientv1.CoreV1Client {
	kubeconfig := os.Getenv("KUBECONFIG")
	conf, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err, "BuildConfigFromFlags failed")
	coreClient, err := coreclientv1.NewForConfig(conf)
	require.NoError(t, err, "NewForConfig failed")
	return coreClient
}

func TestDriver_InitClient(t *testing.T) {
	t.Run("kubeconfig", func(t *testing.T) {
		d := Driver{
			Kubeconfig: os.Getenv("KUBECONFIG"),
		}
		err := d.initClient()
		require.NoError(t, err)
	})
}
