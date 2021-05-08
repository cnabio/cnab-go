package kubernetes

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/driver"
)

func TestDriver_Run(t *testing.T) {
	ctx := context.Background()
	// Simulate the shared volume
	sharedDir, err := ioutil.TempDir("", "cnab-go")
	require.NoError(t, err, "could not create test directory")
	defer os.RemoveAll(sharedDir)

	client := fake.NewSimpleClientset()
	namespace := "default"
	k := Driver{
		Namespace:          namespace,
		jobs:               client.BatchV1().Jobs(namespace),
		secrets:            client.CoreV1().Secrets(namespace),
		pods:               client.CoreV1().Pods(namespace),
		JobVolumePath:      sharedDir,
		JobVolumeName:      "cnab-driver-shared",
		SkipCleanup:        true,
		skipJobStatusCheck: true,
	}
	op := driver.Operation{
		Action: "install",
		Bundle: &bundle.Bundle{},
		Image:  bundle.InvocationImage{BaseImage: bundle.BaseImage{Image: "foo/bar"}},
		Out:    os.Stdout,
		Environment: map[string]string{
			"foo": "bar",
		},
	}

	_, err = k.Run(&op)
	assert.NoError(t, err)

	jobList, _ := k.jobs.List(ctx, metav1.ListOptions{})
	assert.Equal(t, len(jobList.Items), 1, "expected one job to be created")

	secretList, _ := k.secrets.List(ctx, metav1.ListOptions{})
	assert.Equal(t, len(secretList.Items), 1, "expected one secret to be created")
}

func TestDriver_RunWithSharedFiles(t *testing.T) {
	ctx := context.Background()
	// Simulate the shared volume
	sharedDir, err := ioutil.TempDir("", "cnab-go")
	require.NoError(t, err, "could not create test directory")
	defer os.RemoveAll(sharedDir)

	// Simulate that the bundle generated output "foo"
	err = os.Mkdir(filepath.Join(sharedDir, "outputs"), 0755)
	require.NoError(t, err, "could not create outputs directory")
	err = ioutil.WriteFile(filepath.Join(sharedDir, "outputs/foo"), []byte("foobar"), 0644)
	require.NoError(t, err, "could not write output foo")

	client := fake.NewSimpleClientset()
	namespace := "default"
	k := Driver{
		Namespace:          namespace,
		jobs:               client.BatchV1().Jobs(namespace),
		secrets:            client.CoreV1().Secrets(namespace),
		pods:               client.CoreV1().Pods(namespace),
		JobVolumePath:      sharedDir,
		JobVolumeName:      "cnab-driver-shared",
		SkipCleanup:        true,
		skipJobStatusCheck: true,
	}
	op := driver.Operation{
		Action: "install",
		Image:  bundle.InvocationImage{BaseImage: bundle.BaseImage{Image: "foo/bar"}},
		Bundle: &bundle.Bundle{
			Outputs: map[string]bundle.Output{
				"foo": {
					Definition: "foo",
					Path:       "/cnab/app/outputs/foo",
				},
			},
		},
		Out: os.Stdout,
		Outputs: map[string]string{
			"/cnab/app/outputs/foo": "foo",
		},
		Environment: map[string]string{
			"foo": "bar",
		},
		Files: map[string]string{
			"/cnab/app/someinput": "input value",
		},
	}

	opResult, err := k.Run(&op)
	require.NoError(t, err)

	jobList, _ := k.jobs.List(ctx, metav1.ListOptions{})
	assert.Equal(t, len(jobList.Items), 1, "expected one job to be created")

	secretList, _ := k.secrets.List(ctx, metav1.ListOptions{})
	assert.Equal(t, len(secretList.Items), 1, "expected one secret to be created")

	require.Contains(t, opResult.Outputs, "foo", "expected the foo output to be collected")
	assert.Equal(t, "foobar", opResult.Outputs["foo"], "invalid output value for foo ")

	wantInputFile := filepath.Join(sharedDir, "inputs/cnab/app/someinput")
	inputContents, err := ioutil.ReadFile(wantInputFile)
	require.NoErrorf(t, err, "could not read generated input file %s on shared volume", wantInputFile)
	assert.Equal(t, "input value", string(inputContents), "invalid input file contents")
}

func TestImageWithDigest(t *testing.T) {
	testCases := map[string]bundle.InvocationImage{
		"foo": {
			BaseImage: bundle.BaseImage{
				Image: "foo",
			},
		},
		"foo/bar": {
			BaseImage: bundle.BaseImage{
				Image: "foo/bar",
			},
		},
		"foo/bar:baz": {
			BaseImage: bundle.BaseImage{
				Image: "foo/bar:baz",
			},
		},
		"foo/bar:baz@sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21": {
			BaseImage: bundle.BaseImage{
				Image:  "foo/bar:baz",
				Digest: "sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21",
			},
		},
		"foo/fun@sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21": {
			BaseImage: bundle.BaseImage{
				Image:  "foo/fun@sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21",
				Digest: "",
			},
		},
		"taco/truck@sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21": {
			BaseImage: bundle.BaseImage{
				Image:  "taco/truck",
				Digest: "sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21",
			},
		},
		"foo/baz@sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21": {
			BaseImage: bundle.BaseImage{
				Image:  "foo/baz@sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21",
				Digest: "sha256:9cfb3575ae5ff2b23ffa3c8e9514d818a9028a71b1d1e3b56b31937188a70b21",
			},
		},
	}

	for expectedImageRef, img := range testCases {
		t.Run(expectedImageRef, func(t *testing.T) {
			img, err := imageWithDigest(img)
			require.NoError(t, err)
			assert.Equal(t, expectedImageRef, img)
		})
	}
}

func TestImageWithDigest_Failures(t *testing.T) {
	testcases := []struct {
		image     string
		digest    string
		wantError string
	}{
		{"foo/bar@sha:invalid", "",
			"could not parse foo/bar@sha:invalid as an OCI reference"},
		{"foo/bar:baz", "sha:invalid",
			"invalid digest sha:invalid specified for invocation image foo/bar:baz"},
		{"foo/bar@sha256:276f1974b4749003bc6c934593983314227cc9a1e6b922396fff59647b82dc4e", "sha256:176f1974b4749003bc6c934593983314227cc9a1e6b922396fff59647b82dc4e",
			"The digest sha256:176f1974b4749003bc6c934593983314227cc9a1e6b922396fff59647b82dc4e for the image foo/bar@sha256:276f1974b4749003bc6c934593983314227cc9a1e6b922396fff59647b82dc4e doesn't match the one specified in the image"},
	}

	for _, tc := range testcases {
		input := bundle.InvocationImage{}
		input.Image = tc.image
		input.Digest = tc.digest
		_, err := imageWithDigest(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), tc.wantError)
	}
}

func TestGenerateNameTemplate(t *testing.T) {
	testCases := map[string]struct {
		op       *driver.Operation
		expected string
	}{
		"short name": {
			op: &driver.Operation{
				Action:       "install",
				Installation: "foo",
			},
			expected: "install-foo-",
		},
		"special chars": {
			op: &driver.Operation{
				Action:       "example.com/liftoff",
				Installation: "🚀 me to the 🌙",
			},
			expected: "example.com-liftoff-me-to-the-",
		},
		"long installation name": {
			op: &driver.Operation{
				Action:       "install",
				Installation: "this-should-be-truncated-qcUYSfR9MS3BqR0kRDHe2K5EHJa8BJGrcoiDVvsDpATjIkr",
			},
			expected: "install-this-should-be-truncated-qcuysfr9ms3bqr0k-",
		},
		"maximum matching segments": {
			op: &driver.Operation{
				Action:       "a",
				Installation: "b c d e f g h i j k l m n o p q r s t u v w x y z",
			},
			expected: "a-b-c-d-e-f-g-h-i-j-k-l-m-n-o-p-q-r-s-t-u-v-w-x-y-",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			actual := generateNameTemplate(tc.op)
			assert.Equal(t, tc.expected, actual)
			assert.True(t, len(actual) <= maxNameTemplateLength)
		})
	}
}

func TestDriver_ConfigureJob(t *testing.T) {
	ctx := context.Background()
	// Simulate the shared volume
	sharedDir, err := ioutil.TempDir("", "cnab-go")
	require.NoError(t, err, "could not create test directory")
	defer os.RemoveAll(sharedDir)

	client := fake.NewSimpleClientset()
	namespace := "myns"
	k := Driver{
		Namespace:             namespace,
		ActiveDeadlineSeconds: 0,
		Annotations:           map[string]string{"b": "2"},
		Labels:                []string{"a=1"},
		jobs:                  client.BatchV1().Jobs(namespace),
		secrets:               client.CoreV1().Secrets(namespace),
		pods:                  client.CoreV1().Pods(namespace),
		JobVolumePath:         sharedDir,
		JobVolumeName:         "cnab-driver-shared",
		SkipCleanup:           true,
		skipJobStatusCheck:    true,
	}
	op := driver.Operation{
		Action:       "install",
		Installation: "mybundle",
		Revision:     "abc123",
		Bundle:       &bundle.Bundle{},
		Image:        bundle.InvocationImage{BaseImage: bundle.BaseImage{Image: "foo/bar"}},
	}

	_, err = k.Run(&op)
	assert.NoError(t, err)

	jobList, _ := k.jobs.List(ctx, metav1.ListOptions{})
	assert.Len(t, jobList.Items, 1, "expected one job to be created")

	job := jobList.Items[0]
	assert.Nil(t, job.Spec.ActiveDeadlineSeconds, "incorrect Job ActiveDeadlineSeconds")
	assert.Equal(t, int32(1), *job.Spec.Completions, "incorrect Job Completions")
	assert.Equal(t, int32(0), *job.Spec.BackoffLimit, "incorrect Job BackoffLimit")

	wantLabels := map[string]string{
		"a":              "1",
		"cnab.io/driver": "kubernetes"}
	assert.Equal(t, wantLabels, job.Labels, "Incorrect Job Labels")

	wantAnnotations := map[string]string{
		"b":                    "2",
		"cnab.io/action":       "install",
		"cnab.io/installation": "mybundle",
		"cnab.io/revision":     "abc123"}
	assert.Equal(t, wantAnnotations, job.Annotations, "Incorrect Job Annotations")

	pod := job.Spec.Template
	assert.Equal(t, wantLabels, pod.Labels, "incorrect Pod Labels")
	assert.Equal(t, wantAnnotations, pod.Annotations, "incorrect Pod Annotations")
	assert.Len(t, pod.Spec.Containers, 1, "expected one container in the pod")

	container := pod.Spec.Containers[0]
	assert.Empty(t, container.Resources.Limits, "incorrect Limits")
}

func TestDriver_SetConfig(t *testing.T) {
	validSettings := func() map[string]string {
		return map[string]string{
			SettingInCluster:              "true",
			SettingKubeconfig:             "/tmp/kube.config",
			SettingMasterURL:              "http://example.com",
			SettingKubeNamespace:          "default",
			SettingJobVolumeName:          "cnab-driver-shared",
			SettingJobVolumePath:          "/tmp",
			SettingCleanupJobs:            "false",
			SettingLabels:                 "a=1 b=2",
			SettingServiceAccount:         "myacct",
			SettingPodAffinityMatchLabels: "a=b x=y",
		}
	}

	t.Run("valid config", func(t *testing.T) {
		d := Driver{}
		err := d.SetConfig(validSettings())
		require.NoError(t, err)

		assert.Equal(t, d.Namespace, "default", "incorrect Namespace value")
		assert.Equal(t, d.JobVolumeName, "cnab-driver-shared", "incorrect JobVolumeName value")
		assert.Equal(t, d.JobVolumePath, "/tmp", "incorrect JobVolumePath value")
		assert.True(t, d.SkipCleanup, "incorrect SkipCleanup value")
		assert.Equal(t, []string{"a=1", "b=2"}, d.Labels, "incorrect Labels value")
		assert.Equal(t, "myacct", d.ServiceAccountName, "incorrect ServiceAccountName")
		assert.Equal(t, int64(0), d.ActiveDeadlineSeconds, "ActiveDeadlineSeconds should be defaulted to 0 so bundle runs are not cut off")
	})

	t.Run("incluster config", func(t *testing.T) {
		d := Driver{}
		settings := validSettings()
		settings[SettingInCluster] = "true"
		err := d.SetConfig(settings)
		require.NoError(t, err)

		assert.True(t, d.InCluster, "incorrect InCluster value")
		assert.Empty(t, d.Kubeconfig, "incorrect Kubeconfig value")
		assert.Empty(t, d.MasterURL, "incorrect MasterUrl value")
	})

	t.Run("kubeconfig", func(t *testing.T) {
		d := Driver{}
		settings := validSettings()
		settings[SettingInCluster] = "false"
		err := d.SetConfig(settings)
		require.NoError(t, err)

		assert.False(t, d.InCluster, "incorrect InCluster value")
		assert.Equal(t, "/tmp/kube.config", d.Kubeconfig, "incorrect Kubeconfig value")
		assert.Equal(t, "http://example.com", d.MasterURL, "incorrect MasterUrl value")
	})

	t.Run("master url optional", func(t *testing.T) {
		d := Driver{}
		settings := validSettings()
		settings[SettingInCluster] = "false"
		settings[SettingMasterURL] = ""
		err := d.SetConfig(settings)
		require.NoError(t, err)

		assert.False(t, d.InCluster, "incorrect InCluster value")
		assert.Equal(t, "/tmp/kube.config", d.Kubeconfig, "incorrect Kubeconfig value")
		assert.Empty(t, d.MasterURL, "incorrect MasterUrl value")
	})

	t.Run("job volume name missing", func(t *testing.T) {
		d := Driver{}
		settings := validSettings()
		settings[SettingJobVolumeName] = ""
		err := d.SetConfig(settings)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "setting JOB_VOLUME_NAME is required")
	})

	t.Run("job volume path missing", func(t *testing.T) {
		d := Driver{}
		settings := validSettings()
		settings[SettingJobVolumePath] = ""
		err := d.SetConfig(settings)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "setting JOB_VOLUME_PATH is required")
	})

	t.Run("invalid PodAffinity match labels ", func(t *testing.T) {
		d := Driver{}
		settings := validSettings()
		settings[SettingPodAffinityMatchLabels] = "AB"
		err := d.SetConfig(settings)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AFFINITY_MATCH_LABELS is incorrectly formattted each value should be in the form X=Y, got")
	})

	t.Run("job volume path missing", func(t *testing.T) {
		d := Driver{}
		settings := validSettings()
		settings[SettingPodAffinityMatchLabels] = "A=B%C"
		err := d.SetConfig(settings)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character")
	})
}
