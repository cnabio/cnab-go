package kubernetes

import (
	"os"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDriver_Run(t *testing.T) {
	client := fake.NewSimpleClientset()
	namespace := "default"
	k := Driver{
		Namespace:          namespace,
		jobs:               client.BatchV1().Jobs(namespace),
		secrets:            client.CoreV1().Secrets(namespace),
		pods:               client.CoreV1().Pods(namespace),
		SkipCleanup:        true,
		skipJobStatusCheck: true,
	}
	op := driver.Operation{
		Action: "install",
		Out:    os.Stdout,
		Environment: map[string]string{
			"foo": "bar",
		},
	}

	_, err := k.Run(&op)
	assert.NoError(t, err)

	jobList, _ := k.jobs.List(metav1.ListOptions{})
	assert.Equal(t, len(jobList.Items), 1, "expected one job to be created")

	secretList, _ := k.secrets.List(metav1.ListOptions{})
	assert.Equal(t, len(secretList.Items), 1, "expected one secret to be created")
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
		"foo/bar:baz@sha:a1b2c3": {
			BaseImage: bundle.BaseImage{
				Image:  "foo/bar:baz",
				Digest: "sha:a1b2c3",
			},
		},
		"foo/bar@sha:a1b2c3": {
			BaseImage: bundle.BaseImage{
				Image:  "foo/bar",
				Digest: "sha:a1b2c3",
			},
		},
	}

	for expectedImageRef, img := range testCases {
		t.Run(expectedImageRef, func(t *testing.T) {
			assert.Equal(t, expectedImageRef, imageWithDigest(img))
		})
	}
}
