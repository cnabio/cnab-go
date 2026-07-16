//go:build !windows
// +build !windows

package command

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/driver"
)

// TestCheckDriverExists_NoShellInjection guards against GHSA-6pq9-97jw-vrmr:
// a driver Name containing shell metacharacters must never be interpreted
// by a shell.
//
// The temp dir must be all-lowercase: d.cmd() lowercases the whole Name
// (including this embedded path) before use, so a mixed-case path (e.g.
// from t.TempDir(), which embeds the test name) would check the wrong
// path and mask the vulnerability.
func TestCheckDriverExists_NoShellInjection(t *testing.T) {
	tmp, err := os.MkdirTemp("", "cnab-injection-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	marker := filepath.Join(tmp, "injected")

	d := &Driver{Name: "x; touch " + marker + "; echo y"}
	exists := d.CheckDriverExists()

	assert.False(t, exists, "expected no driver to be found for the malicious name")
	_, statErr := os.Stat(marker)
	assert.True(t, os.IsNotExist(statErr), "marker file must not have been created")
}

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
			opResult, err := cmddriver.Run(context.Background(), op)
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
			opResult, err := cmddriver.Run(context.Background(), op)
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
			_, err := cmddriver.Run(context.Background(), op)
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
			opResult, err := cmddriver.Run(context.Background(), op)
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

	// GHSA-x74g-hj34-9579: a bundle-declared output path with traversal
	// segments must not let the driver read files outside its own
	// temporary output directory.
	t.Run("output path traversal is rejected", func(t *testing.T) {
		secret, err := os.CreateTemp("", "cnab-poc-secret")
		require.NoError(t, err, "could not create secret file")
		defer os.Remove(secret.Name())
		_, err = secret.WriteString("TOP-SECRET-HOST-DATA")
		require.NoError(t, err)
		require.NoError(t, secret.Close())

		traversalPaths := []string{
			"/../../../../../../../.." + secret.Name(),
			"../../../../../../../.." + secret.Name(),
			"/cnab/app/outputs/../../../../../../../.." + secret.Name(),
		}

		// Create the conventional outputs directory structure, so the third
		// (prefixed) traversal path actually walks through real directories
		// on its way out, rather than failing early on a missing "cnab" dir.
		content := `#!/bin/sh
		mkdir -p "${CNAB_OUTPUT_DIR}/cnab/app/outputs"
	`
		name := "test-outputs-traversal.sh"
		for _, traversalPath := range traversalPaths {
			t.Run(traversalPath, func(t *testing.T) {
				testfunc := func(cmddriver *Driver) {
					if !cmddriver.CheckDriverExists() {
						t.Fatalf("Expected driver %s to exist Driver Name %s ", name, cmddriver.Name)
					}
					op := buildOp()
					op.Outputs = map[string]string{traversalPath: "leaked"}

					opResult, err := cmddriver.Run(context.Background(), op)
					assert.Error(t, err, "expected Run to reject a path-traversal output file")
					assert.NotContains(t, opResult.Outputs, "leaked")
				}
				CreateAndRunTestCommandDriver(t, name, false, content, testfunc)
			})
		}
	})
}
