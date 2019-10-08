// +build integration

package docker

import (
	"bytes"
	"os"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/bundle/definition"
	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
)

func TestDriver_Run(t *testing.T) {
	imageFromEnv, ok := os.LookupEnv("DOCKER_INTEGRATION_TEST_IMAGE")
	var image bundle.InvocationImage

	if ok {
		image = bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image: imageFromEnv,
			},
		}
	} else {
		image = bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image:  "pvtlmc/example-outputs",
				Digest: "sha256:568461508c8d220742add8abd226b33534d4269868df4b3178fae1cba3818a6e",
			},
		}
	}

	op := &driver.Operation{
		Installation: "example",
		Action:       "install",
		Image:        image,
		Outputs: []string{
			"/cnab/app/outputs/output1",
			"/cnab/app/outputs/output2",
			"/cnab/app/outputs/missingApplicableOutputWithDefault",
			"/cnab/app/outputs/missingNonApplicableOutputWithDefault",
		},
		Bundle: &bundle.Bundle{
			Definitions: definition.Definitions{
				"output1": &definition.Schema{},
				"output2": &definition.Schema{},
				"missingApplicableOutputWithDefault": &definition.Schema{
					Default: "foo",
				},
				"missingNonApplicableOutputWithDefault": &definition.Schema{
					Default: "bar",
				},
			},
			Outputs: map[string]bundle.Output{
				"output1": {
					Definition: "output1",
				},
				"output2": {
					Definition: "output2",
				},
				"missingApplicableOutputWithDefault": {
					Definition: "missingApplicableOutputWithDefault",
				},
				"missingNonApplicableOutputWithDefault": {
					Definition: "missingApplicableOutputWithDefault",
					ApplyTo:    []string{"upgrade"},
				},
			},
		},
	}

	var output bytes.Buffer
	op.Out = &output
	op.Environment = map[string]string{
		"CNAB_ACTION":            op.Action,
		"CNAB_INSTALLATION_NAME": op.Installation,
	}

	docker := &Driver{}
	docker.SetContainerOut(op.Out) // Docker driver writes container stdout to driver.containerOut.
	opResult, err := docker.Run(op)

	assert.NoError(t, err)
	assert.Equal(t, "Install action\nAction install complete for example\n", output.String())
	assert.Equal(t, 3, len(opResult.Outputs), "Expecting three output files")
	assert.Equal(t, map[string]string{
		"/cnab/app/outputs/output1":                            "SOME INSTALL CONTENT 1\n",
		"/cnab/app/outputs/output2":                            "SOME INSTALL CONTENT 2\n",
		"/cnab/app/outputs/missingApplicableOutputWithDefault": "foo",
	}, opResult.Outputs)
}

func TestDriver_Run_MissingRequiredOutput(t *testing.T) {
	imageFromEnv, ok := os.LookupEnv("DOCKER_INTEGRATION_TEST_IMAGE")
	var image bundle.InvocationImage

	if ok {
		image = bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image: imageFromEnv,
			},
		}
	} else {
		image = bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image:  "pvtlmc/example-outputs",
				Digest: "sha256:568461508c8d220742add8abd226b33534d4269868df4b3178fae1cba3818a6e",
			},
		}
	}

	op := &driver.Operation{
		Installation: "example",
		Action:       "install",
		Image:        image,
		Outputs: []string{
			"/cnab/app/outputs/missingApplicableOutputSansDefault",
		},
		Bundle: &bundle.Bundle{
			Definitions: definition.Definitions{
				"missingApplicableOutputSansDefault": &definition.Schema{},
			},
			Outputs: map[string]bundle.Output{
				"missingApplicableOutputSansDefault": {
					Definition: "missingApplicableOutputSansDefault",
				},
			},
		},
	}

	var output bytes.Buffer
	op.Out = &output
	op.Environment = map[string]string{
		"CNAB_ACTION":            op.Action,
		"CNAB_INSTALLATION_NAME": op.Installation,
	}

	docker := &Driver{}
	docker.SetContainerOut(op.Out) // Docker driver writes container stdout to driver.containerOut.
	_, err := docker.Run(op)
	assert.EqualError(t, err, "required output missingApplicableOutputSansDefault is missing and has no default")
}
