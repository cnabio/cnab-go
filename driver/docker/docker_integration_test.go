//go:build integration
// +build integration

package docker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/driver"
)

// The bundles in this file are located in /testdata/bundles/example-outputs
// Refresh by changing the registry to one you can push to, and running /testdata/bundles/example-outputs/build.sh
var defaultBaseImage = bundle.BaseImage{
	Image:  "carolynvs/example-outputs:v1.0.0",
	Digest: "sha256:b4a6e86cde93a7ab0b7953b2463e6547aaa08331876b0d0896e3cdc85de4363e",
}

func TestDockerDriver_Run(t *testing.T) {
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

	runDriverTest(t, image, false)
}

// Test running a bundle where the invocation image runs as a nonroot user
func TestDockerDriver_RunNonrootInvocationImage(t *testing.T) {
	image := bundle.InvocationImage{
		BaseImage: bundle.BaseImage{
			Image:  "carolynvs/example-outputs:v1.0.0-nonroot",
			Digest: "sha256:6dd56d1974fcd3436d12856f621e78a43e62caf94cb4ab9f830c6e03e0c5fdb2",
		},
	}

	runDriverTest(t, image, false)
}

func runDriverTest(t *testing.T, image bundle.InvocationImage, skipValidations bool) {
	op := &driver.Operation{
		Installation: "example",
		Action:       "install",
		Image:        image,
		Files: map[string]string{
			"/cnab/app/inputs/input1": "input1",
		},
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
	opResult, err := docker.Run(op)

	if skipValidations {
		return
	}

	require.NoError(t, err)
	assert.Equal(t, "Install action\n\nListing inputs\ninput1\n\nGenerating outputs\nAction install complete for example\n", output.String())
	assert.Equal(t, 2, len(opResult.Outputs), "Expecting two output files")
	assert.Equal(t, map[string]string{
		"output1": "input1\n",
		"output2": "SOME INSTALL CONTENT 2\n",
	}, opResult.Outputs)
}

func TestDockerDriver_NoOpErrSet(t *testing.T) {
	// Validate that when op.Err is not set that we print to stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStderr := os.Stderr
	defer func() {
		os.Stderr = origStderr
	}()
	os.Stderr = w

	image := bundle.InvocationImage{
		BaseImage: defaultBaseImage,
	}
	image.Image += "-oops-this-does-not-exist"
	runDriverTest(t, image, true)
	w.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Unable to find image 'carolynvs/example-outputs:v1.0.0-oops-this-does-not-exist'")
}

func TestDriver_Run_CaptureOutput(t *testing.T) {
	image := bundle.InvocationImage{
		BaseImage: bundle.BaseImage{
			Image:  "carolynvs/cnab-bad-invocation-image:v1",
			Digest: "sha256:0705cbb5fdeec6752a0a0f707b8e1f7ad63070bf64713a4d23b69ca452fe3d37",
		},
	}

	op := &driver.Operation{
		Installation: "example",
		Action:       "install",
		Image:        image,
		Bundle:       &bundle.Bundle{},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	op.Out = &stdout
	op.Err = &stderr
	op.Environment = map[string]string{
		"CNAB_ACTION":            op.Action,
		"CNAB_INSTALLATION_NAME": op.Installation,
	}

	docker := &Driver{}
	_, err := docker.Run(op)

	assert.NoError(t, err)
	assert.Equal(t, "installing bundle...\n", stdout.String())
	assert.Equal(t, "cat: can't open 'missing-file.txt': No such file or directory\n", stderr.String())
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
	require.Error(t, err, "expected an error")
	// Not asserting actual image digests to support arbitrary integration test images
	assert.Contains(t, err.Error(),
		fmt.Sprintf("content digest mismatch: invocation image %s was defined in the bundle with the digest %s but no matching repoDigest was found upon inspecting the image", op.Image.Image, badDigest))
}
