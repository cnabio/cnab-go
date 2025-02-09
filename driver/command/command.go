package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/cnabio/cnab-go/driver"
)

// Driver relies upon a system command to provide a driver implementation
type Driver struct {
	Name string

	// Path is the absolute path to the driver executable.
	// If unset, the executable is expected to be named "cnab-NAME" and be on the PATH.
	Path string

	outputDirName string
}

// Run executes the command
func (d *Driver) Run(_ context.Context, op *driver.Operation) (driver.OperationResult, error) {
	return d.exec(op)
}

// Handles executes the driver with `--handles` and parses the results
func (d *Driver) Handles(dt string) bool {
	out, err := exec.Command(d.cmd(), "--handles").CombinedOutput()
	if err != nil {
		return false
	}
	types := strings.Split(string(out), ",")
	for _, tt := range types {
		if dt == strings.TrimSpace(tt) {
			return true
		}
	}
	return false
}

// cmd is the command to run to execute the driver.
//
// When the driver does not have the path to the executable set,
// automatically prepend "cnab-" to the name of the driver
// so it can be found in PATH.
func (d *Driver) cmd() string {
	if d.Path != "" {
		return d.Path
	}
	return "cnab-" + strings.ToLower(d.Name)
}

func (d *Driver) exec(op *driver.Operation) (driver.OperationResult, error) {
	// We need to do two things here: We need to make it easier for the
	// command to access data, and we need to make it easy for the command
	// to pass that data on to the image it invokes. So we do some data
	// duplication.

	// Construct an environment for the subprocess by cloning our
	// environment and adding in all the extra env vars.
	pairs := os.Environ()
	added := []string{}
	for k, v := range op.Environment {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		added = append(added, k)
	}
	// Create a directory that can be used for outputs and then pass it as a command line argument
	if len(op.Outputs) > 0 {
		var err error
		d.outputDirName, err = ioutil.TempDir("", "bundleoutput")
		if err != nil {
			return driver.OperationResult{}, err
		}
		defer os.RemoveAll(d.outputDirName)
		// Set the env var CNAB_OUTPUT_DIR to the location of the folder
		pairs = append(pairs, fmt.Sprintf("%s=%s", "CNAB_OUTPUT_DIR", d.outputDirName))
		added = append(added, "CNAB_OUTPUT_DIR")
	}

	// CNAB_VARS is a list of variables we added to the env. This is to make
	// it easier for shell script drivers to clone the env vars.
	pairs = append(pairs, fmt.Sprintf("CNAB_VARS=%s", strings.Join(added, ",")))
	data, err := json.Marshal(op)
	if err != nil {
		return driver.OperationResult{}, err
	}

	args := []string{}
	cmd := exec.Command(d.cmd(), args...)
	cmd.Dir, err = os.Getwd()
	if err != nil {
		return driver.OperationResult{}, err
	}
	cmd.Env = pairs
	cmd.Stdin = bytes.NewBuffer(data)
	// Make stdout and stderr from driver available immediately
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("Setting up output handling for driver (%s) failed: %v", d.Name, err)
	}

	go func() {

		// Errors not handled here as they only prevent output from the driver being shown, errors in the command execution are handled when command is executed

		io.Copy(op.Out, stdout)
	}()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("Setting up error output handling for driver (%s) failed: %v", d.Name, err)
	}

	go func() {

		// Errors not handled here as they only prevent output from the driver being shown, errors in the command execution are handled when command is executed

		io.Copy(op.Err, stderr)
	}()

	if err = cmd.Start(); err != nil {
		return driver.OperationResult{}, fmt.Errorf("Start of driver (%s) failed: %v", d.Name, err)
	}

	if err = cmd.Wait(); err != nil {
		return driver.OperationResult{}, fmt.Errorf("Command driver (%s) failed executing bundle: %v", d.Name, err)
	}

	result, err := d.getOperationResult(op)
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("Command driver (%s) failed getting operation result: %v", d.Name, err)
	}
	return result, nil
}

func (d *Driver) getOperationResult(op *driver.Operation) (driver.OperationResult, error) {
	opResult := driver.OperationResult{
		Outputs: map[string]string{},
	}
	for outputPath, outputName := range op.Outputs {
		fileName := path.Join(d.outputDirName, outputPath)
		_, err := os.Stat(fileName)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return opResult, fmt.Errorf("Command driver (%s) failed checking for output file: %s Error: %v", d.Name, outputPath, err)
		}

		contents, err := ioutil.ReadFile(fileName)
		if err != nil {
			return opResult, fmt.Errorf("Command driver (%s) failed reading output file: %s Error: %v", d.Name, outputPath, err)
		}

		opResult.Outputs[outputName] = string(contents)
	}
	return opResult, nil
}
