// +build !windows

package command

import (
	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"

	"os"
	"testing"
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
			Outputs: []string{"/cnab/app/outputs/output1", "/cnab/app/outputs/output2"},
			Out:     os.Stdout,
		}
		opResult, err := cmddriver.Run(&op)
		if err != nil {
			t.Fatalf("Driver Run failed %v", err)
		}
		assert.Equal(t, 2, len(opResult.Outputs), "Expecting two output files")
		assert.Equal(t, map[string]string{
			"/cnab/app/outputs/output1": "TEST_OUTPUT_1\n",
			"/cnab/app/outputs/output2": "TEST_OUTPUT_2\n",
		}, opResult.Outputs)
	}
	CreateAndRunTestCommandDriver(t, name, content, testfunc)
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
			Outputs: []string{"/cnab/app/outputs/output1", "/cnab/app/outputs/output2"},
			Out:     os.Stdout,
		}
		_, err := cmddriver.Run(&op)
		assert.Errorf(t, err, "Command driver (test-outputs-missing.sh) failed for item: /cnab/app/outputs/output2 no output value found and no default value set")
	}
	CreateAndRunTestCommandDriver(t, name, content, testfunc)
}
