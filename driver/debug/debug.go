package debug

import (
	"encoding/json"
	"fmt"

	"github.com/cnabio/cnab-go/driver"
)

// Driver prints the information passed to a driver
//
// It does not ever run the image.
type Driver struct {
	config map[string]string
}

// Run executes the operation on the Debug driver
func (d *Driver) Run(op *driver.Operation) (driver.OperationResult, error) {
	data, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return driver.OperationResult{}, err
	}

	result := driver.OperationResult{}
	result.Logs.Write(data)

	fmt.Fprintln(op.Out, result.Logs.String())

	return result, nil
}

// Handles always returns true, effectively claiming to work for any image type
func (d *Driver) Handles(dt string) bool {
	return true
}

// Config returns the configuration help text
func (d *Driver) Config() map[string]string {
	return map[string]string{
		"VERBOSE": "Increase verbosity. true, false are supported values",
	}
}

// SetConfig sets configuration for this driver
func (d *Driver) SetConfig(settings map[string]string) error {
	d.config = settings
	return nil
}
