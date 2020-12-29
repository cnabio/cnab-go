// +build !windows

package command

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/driver"
)

func TestCommandDriverOutputs(t *testing.T) {
	buildOp := func() *driver.Operation {
		return &driver.Operation{
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
	}

	t.Run("output exists", func(t *testing.T) {
		content := `#!/bin/sh
		mkdir -p "${CNAB_OUTPUT_DIR}/cnab/app/outputs"
		echo "TEST_OUTPUT_1" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output1"
		echo "TEST_OUTPUT_2" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output2"
	`
		name := "test-outputs-exist.sh"
		testfunc := func(cmddriver *Driver) {
			if !cmddriver.CheckDriverExists() {
				t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
			}
			op := buildOp()
			opResult, err := cmddriver.Run(op)
			if err != nil {
				t.Fatalf("Driver Run failed %v", err)
			}
			assert.Equal(t, 2, len(opResult.Outputs), "Expecting two output files")
			assert.Equal(t, map[string]string{
				"output1": "TEST_OUTPUT_1\n",
				"output2": "TEST_OUTPUT_2\n",
			}, opResult.Outputs)
		}
		CreateAndRunTestCommandDriver(t, name, false, content, testfunc)
	})

	t.Run("output exists - path set", func(t *testing.T) {
		content := `#!/bin/sh
		mkdir -p "${CNAB_OUTPUT_DIR}/cnab/app/outputs"
		echo "TEST_OUTPUT_1" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output1"
		echo "TEST_OUTPUT_2" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output2"
	`
		name := "test-outputs-exist.sh"
		testfunc := func(cmddriver *Driver) {
			if !cmddriver.CheckDriverExists() {
				t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
			}
			op := buildOp()
			opResult, err := cmddriver.Run(op)
			if err != nil {
				t.Fatalf("Driver Run failed %v", err)
			}
			assert.Equal(t, 2, len(opResult.Outputs), "Expecting two output files")
			assert.Equal(t, map[string]string{
				"output1": "TEST_OUTPUT_1\n",
				"output2": "TEST_OUTPUT_2\n",
			}, opResult.Outputs)
		}
		CreateAndRunTestCommandDriver(t, name, true, content, testfunc)
	})

	// Test for an output missing and no defaults
	t.Run("output missing - no defaults", func(t *testing.T) {
		content := `#!/bin/sh
		mkdir -p "${CNAB_OUTPUT_DIR}/cnab/app/outputs"
		echo "TEST_OUTPUT_1" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output1"
	`
		name := "test-outputs-missing.sh"
		testfunc := func(cmddriver *Driver) {
			if !cmddriver.CheckDriverExists() {
				t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
			}
			op := buildOp()
			_, err := cmddriver.Run(op)
			assert.NoError(t, err)
		}
		CreateAndRunTestCommandDriver(t, name, false, content, testfunc)
	})

	// Test for an output missing with default value present
	t.Run("output missing - default set", func(t *testing.T) {
		content := `#!/bin/sh
		mkdir -p "${CNAB_OUTPUT_DIR}/cnab/app/outputs"
		echo "TEST_OUTPUT_1" >> "${CNAB_OUTPUT_DIR}/cnab/app/outputs/output1"
	`
		name := "test-outputs-missing.sh"
		testfunc := func(cmddriver *Driver) {
			if !cmddriver.CheckDriverExists() {
				t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
			}
			op := buildOp()
			op.Bundle.Definitions["output2"].Default = "DEFAULT OUTPUT 2"
			opResult, err := cmddriver.Run(op)
			if err != nil {
				t.Fatalf("Driver Run failed %v", err)
			}
			assert.Equal(t, 1, len(opResult.Outputs), "Expecting one output files")
			assert.Equal(t, map[string]string{
				"output1": "TEST_OUTPUT_1\n",
			}, opResult.Outputs)
		}
		CreateAndRunTestCommandDriver(t, name, false, content, testfunc)
	})
}
