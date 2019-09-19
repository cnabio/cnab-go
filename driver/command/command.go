package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/deislabs/cnab-go/driver"
)

// Driver relies upon a system command to provide a driver implementation
type Driver struct {
	Name          string
	outputDirName string
}

// Run executes the command
func (d *Driver) Run(op *driver.Operation) (driver.OperationResult, error) {
	return d.exec(op)
}

// Handles executes the driver with `--handles` and parses the results
func (d *Driver) Handles(dt string) bool {
	out, err := exec.Command(d.cliName(), "--handles").CombinedOutput()
	if err != nil {
		fmt.Printf("%s --handles: %s", d.cliName(), err)
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

func (d *Driver) cliName() string {
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
	cmd := exec.Command(d.cliName(), args...)
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

		io.Copy(op.Out, stderr)
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
	if len(op.Outputs) == 0 {
		return opResult, nil
	}

	for _, item := range op.Outputs {
		// Check if a value is required for this output and get the default value if available
		valueRequired, defaultValue, err := checkIfOutputValueRequired(item, op, opResult)
		if err != nil {
			return opResult, fmt.Errorf("Command driver (%s) failed checking if output value required for item: %s Error: %v", d.Name, item, err)
		}

		fileName := path.Join(d.outputDirName, item)
		_, err = os.Stat(fileName)
		if err != nil {
			if os.IsNotExist(err) {
				// Set a default value if available and required otherwise fail
				if valueRequired {
					return opResult, fmt.Errorf("Command driver (%s) failed for item: %s no output value found and no default value set", d.Name, item)
				}
				opResult.Outputs[item] = defaultValue
				continue
			}
			return opResult, fmt.Errorf("Command driver (%s) failed checking for output file: %s Error: %v", d.Name, item, err)
		}

		contents, err := ioutil.ReadFile(fileName)
		if err != nil {
			return opResult, fmt.Errorf("Command driver (%s) failed reading output file: %s Error: %v", d.Name, item, err)
		}

		opResult.Outputs[item] = string(contents)
	}
	return opResult, nil
}
func checkIfOutputValueRequired(item string, op *driver.Operation, opResult driver.OperationResult) (bool, string, error) {
	// TODO Check if the output is required by the action and write test to validate (requires update to op definition in https://github.com/deislabs/cnab-go/pull/129 to be merged )
	// Until this check is inmplemented then all values are required for all bundles and actions
	// Output default values and applies to are ignored by this driver until this function is implemented
	// Function should return false if a output does not apply to an action or there is a default value (which it should return as the second return value)
	return true, "", nil

}
