// +build integration

package kubernetes

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/driver"
)

func TestDriver_Run_Integration(t *testing.T) {
	k := &Driver{}
	k.SetConfig(map[string]string{
		"KUBE_NAMESPACE": "default",
		"KUBECONFIG":     os.Getenv("KUBECONFIG"),
	})
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

			_, err := k.Run(tc.op)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.output, output.String())
		})
	}
}

func TestDriver_SetConfig(t *testing.T) {
	t.Run("kubeconfig", func(t *testing.T) {

		d := Driver{}
		err := d.SetConfig(map[string]string{
			"KUBECONFIG": os.Getenv("KUBECONFIG"),
		})
		require.NoError(t, err)
	})
}
