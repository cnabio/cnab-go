// +build !windows

package command

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/driver"
)

func TestCommandDriverOutputs(t *testing.T) {
	content := `#!/bin/sh
		mkdir -p "${CNAB_OUTPUT_DIR}/cnab/app/outputs"
		echo "TEST_OUTPUT_1" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output1"
		echo "TEST_OUTPUT_2" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output2"
	`
	name := "test-outputs-exist.sh"
	testfunc := func(t *testing.T, cmddriver *Driver) {
		if !cmddriver.CheckDriverExists() {
			t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
		}
		op := driver.Operation{
			Action:       "install",
			Installation: "test",
			Parameters:   map[string]interface{}{},
			Image: bundle.InvocationImage{
				BaseImage: bundle.BaseImage{
					Image:     "cnab/helloworld:latest",
					ImageType: "docker",
				},
			},
			Revision:    "01DDY0MT808KX0GGZ6SMXN4TW",
			Environment: map[string]string{},
			Files: map[string]string{
				"/cnab/app/image-map.json": "{}",
			},
			Outputs: map[string]string{"/cnab/app/outputs/output1": "output1", "/cnab/app/outputs/output2": "output2"},
			Out:     os.Stdout,
			Bundle: &bundle.Bundle{
				Definitions: definition.Definitions{
					"output1": &definition.Schema{},
					"output2": &definition.Schema{},
				},
				Outputs: map[string]bundle.Output{
					"output1": {
						Definition: "output1",
						Path:       "/cnab/app/outputs/output1",
					},
					"output2": {
						Definition: "output2",
						Path:       "/cnab/app/outputs/output2",
					},
				},
			},
		}
		opResult, err := cmddriver.Run(context.Background(), &op)
		if err != nil {
			t.Fatalf("Driver Run failed %v", err)
		}
		assert.Equal(t, 2, len(opResult.Outputs), "Expecting two output files")
		assert.Equal(t, map[string]string{
			"output1": "TEST_OUTPUT_1\n",
			"output2": "TEST_OUTPUT_2\n",
		}, opResult.Outputs)
	}
	CreateAndRunTestCommandDriver(t, name, content, testfunc)
	// Test for an output missing and no defaults
	content = `#!/bin/sh
		mkdir -p "${CNAB_OUTPUT_DIR}/cnab/app/outputs"
		echo "TEST_OUTPUT_1" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output1"
	`
	name = "test-outputs-missing.sh"
	testfunc = func(t *testing.T, cmddriver *Driver) {
		if !cmddriver.CheckDriverExists() {
			t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
		}
		op := driver.Operation{
			Action:       "install",
			Installation: "test",
			Parameters:   map[string]interface{}{},
			Image: bundle.InvocationImage{
				BaseImage: bundle.BaseImage{
					Image:     "cnab/helloworld:latest",
					ImageType: "docker",
				},
			},
			Revision:    "01DDY0MT808KX0GGZ6SMXN4TW",
			Environment: map[string]string{},
			Files: map[string]string{
				"/cnab/app/image-map.json": "{}",
			},
			Outputs: map[string]string{"/cnab/app/outputs/output1": "output1", "/cnab/app/outputs/output2": "output2"},
			Out:     os.Stdout,
			Bundle: &bundle.Bundle{
				Definitions: definition.Definitions{
					"output1": &definition.Schema{},
					"output2": &definition.Schema{},
				},
				Outputs: map[string]bundle.Output{
					"output1": {
						Definition: "output1",
						Path:       "/cnab/app/outputs/output1",
					},
					"output2": {
						Definition: "output2",
						Path:       "/cnab/app/outputs/output2",
					},
				},
			},
		}
		_, err := cmddriver.Run(context.Background(), &op)
		assert.NoError(t, err)
	}
	CreateAndRunTestCommandDriver(t, name, content, testfunc)
	// Test for an output missing with default value present
	content = `#!/bin/sh
		mkdir -p "${CNAB_OUTPUT_DIR}/cnab/app/outputs"
		echo "TEST_OUTPUT_1" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output1"
	`
	name = "test-outputs-missing.sh"
	testfunc = func(t *testing.T, cmddriver *Driver) {
		if !cmddriver.CheckDriverExists() {
			t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
		}
		op := driver.Operation{
			Action:       "install",
			Installation: "test",
			Parameters:   map[string]interface{}{},
			Image: bundle.InvocationImage{
				BaseImage: bundle.BaseImage{
					Image:     "cnab/helloworld:latest",
					ImageType: "docker",
				},
			},
			Revision:    "01DDY0MT808KX0GGZ6SMXN4TW",
			Environment: map[string]string{},
			Files: map[string]string{
				"/cnab/app/image-map.json": "{}",
			},
			Outputs: map[string]string{"/cnab/app/outputs/output1": "output1", "/cnab/app/outputs/output2": "output2"},
			Out:     os.Stdout,
			Bundle: &bundle.Bundle{
				Definitions: definition.Definitions{
					"output1": &definition.Schema{},
					"output2": &definition.Schema{
						Default: "DEFAULT OUTPUT 2",
					},
				},
				Outputs: map[string]bundle.Output{
					"output1": {
						Definition: "output1",
						Path:       "/cnab/app/outputs/output1",
					},
					"output2": {
						Definition: "output2",
						Path:       "/cnab/app/outputs/output2",
					},
				},
			},
		}
		opResult, err := cmddriver.Run(context.Background(), &op)
		if err != nil {
			t.Fatalf("Driver Run failed %v", err)
		}
		assert.Equal(t, 1, len(opResult.Outputs), "Expecting one output files")
		assert.Equal(t, map[string]string{
			"output1": "TEST_OUTPUT_1\n",
		}, opResult.Outputs)
	}
	CreateAndRunTestCommandDriver(t, name, content, testfunc)
}

func TestCommandDriverCancellation(t *testing.T) {
	content := `#!/bin/sh
		echo command executed
	`
	name := "test-command.sh"
	output := bytes.Buffer{}
	testfunc := func(t *testing.T, cmddriver *Driver) {
		if !cmddriver.CheckDriverExists() {
			t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
		}
		op := driver.Operation{
			Action:       "install",
			Installation: "test",
			Parameters:   map[string]interface{}{},
			Image: bundle.InvocationImage{
				BaseImage: bundle.BaseImage{
					Image:     "cnab/helloworld:latest",
					ImageType: "docker",
				},
			},
			Revision:    "01DDY0MT808KX0GGZ6SMXN4TW",
			Environment: map[string]string{},
			Files: map[string]string{
				"/cnab/app/image-map.json": "{}",
			},
			Out:    &output,
			Bundle: &bundle.Bundle{Name: "mybun"},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := cmddriver.Run(ctx, &op)
		require.EqualError(t, err, "Start of driver (test-command.sh) failed: context canceled")
		assert.NotContains(t, output.String(), "command executed")
	}
	CreateAndRunTestCommandDriver(t, name, content, testfunc)
}
