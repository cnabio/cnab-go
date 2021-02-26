// +build integration

package kubernetes

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/driver"
)

func TestDriver_Run_Integration(t *testing.T) {
	k := &Driver{}
	k.ActiveDeadlineSeconds = 60

	cases := []struct {
		name   string
		op     *driver.Operation
		output string
		err    error
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
			var output bytes.Buffer
			tc.op.Out = &output
			if tc.op.Environment == nil {
				tc.op.Environment = map[string]string{}
			}
			tc.op.Environment["CNAB_ACTION"] = tc.op.Action
			tc.op.Environment["CNAB_INSTALLATION_NAME"] = tc.op.Installation

			// Create a volume to share data with the invocation image
			pvc, cleanup := createTestPVC(t)
			defer cleanup()

			// Simulate mounting the shared volume
			sharedDir, err := ioutil.TempDir("", "cnab-go")
			require.NoError(t, err, "could not create test directory")
			defer os.RemoveAll(sharedDir)

			err = k.SetConfig(map[string]string{
				SettingJobVolumePath: sharedDir,
				SettingJobVolumeName: pvc,
				SettingKubeNamespace: "default",
				SettingKubeconfig:    os.Getenv("KUBECONFIG"),
			})
			require.NoError(t, err, "SetConfig failed")

			_, err = k.Run(tc.op)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Contains(t, output.String(), tc.output)
		})
	}
}

func createTestPVC(t *testing.T) (string, func()) {
	pvc := &v1.PersistentVolumeClaim{
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
	kubeconfig := os.Getenv("KUBECONFIG")
	conf, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err, "BuildConfigFromFlags failed")
	coreClient, err := coreclientv1.NewForConfig(conf)
	pvcClient := coreClient.PersistentVolumeClaims("default")
	pvc, err = pvcClient.Create(pvc)
	require.NoError(t, err, "create pvc failed")

	return pvc.Name, func() {
		pvcClient.Delete(pvc.Name, &metav1.DeleteOptions{})
	}
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
