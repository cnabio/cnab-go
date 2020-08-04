// +build integration

package docker

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/driver"
)

var defaultBaseImage = bundle.BaseImage{
	Image:  "pvtlmc/example-outputs",
	Digest: "sha256:568461508c8d220742add8abd226b33534d4269868df4b3178fae1cba3818a6e",
}

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
			BaseImage: defaultBaseImage,
		}
	}

	op := &driver.Operation{
		Installation: "example",
		Action:       "install",
		Image:        image,
		Outputs: map[string]string{
			"/cnab/app/outputs/output1": "output1",
			"/cnab/app/outputs/output2": "output2",
		},
		Bundle: &bundle.Bundle{
			Definitions: definition.Definitions{
				"output1": &definition.Schema{},
				"output2": &definition.Schema{},
			},
			Outputs: map[string]bundle.Output{
				"output1": {
					Definition: "output1",
				},
				"output2": {
					Definition: "output2",
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
	assert.Equal(t, 2, len(opResult.Outputs), "Expecting two output files")
	assert.Equal(t, map[string]string{
		"output1": "SOME INSTALL CONTENT 1\n",
		"output2": "SOME INSTALL CONTENT 2\n",
	}, opResult.Outputs)
}

func TestDriver_ValidateImageDigestFail(t *testing.T) {
	imageFromEnv, ok := os.LookupEnv("DOCKER_INTEGRATION_TEST_IMAGE")
	var image bundle.InvocationImage

	badDigest := "sha256:deadbeef"

	if ok {
		image = bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image:  imageFromEnv,
				Digest: badDigest,
			},
		}
	} else {
		image = bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image:  defaultBaseImage.Image,
				Digest: badDigest,
			},
		}
	}

	op := &driver.Operation{
		Image: image,
	}

	docker := &Driver{}

	_, err := docker.Run(op)
	assert.Error(t, err)
	// Not asserting actual image digests to support arbitrary integration test images
	assert.Contains(t, err.Error(),
		fmt.Sprintf("content digest mismatch: image %s has digest(s)", op.Image.Image))
	assert.Contains(t, err.Error(),
		fmt.Sprintf("but the digest should be %s according to the bundle file", badDigest))
}
