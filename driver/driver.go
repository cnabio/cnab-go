package driver

import (
	"bytes"
	"fmt"
	"io"

	"github.com/cnabio/cnab-go/bundle"
)

// ImageType constants provide some of the image types supported
// TODO: I think we can remove all but Docker, since the rest are supported externally
const (
	ImageTypeDocker = "docker"
	ImageTypeOCI    = "oci"
	ImageTypeQCOW   = "qcow"
)

// Operation describes the data passed into the driver to run an operation
type Operation struct {
	// Installation is the name of this installation
	Installation string `json:"installation_name"`
	// The revision ID for this installation
	Revision string `json:"revision"`
	// Action is the action to be performed
	Action string `json:"action"`
	// Parameters are the parameters to be injected into the container
	Parameters map[string]interface{} `json:"parameters"`
	// Image is the invocation image
	Image bundle.InvocationImage `json:"image"`
	// Environment contains environment variables that should be injected into the invocation image
	Environment map[string]string `json:"environment"`
	// Files contains files that should be injected into the invocation image.
	Files map[string]string `json:"files"`
	// Outputs map of output paths (e.g. /cnab/app/outputs/NAME) to the name of the output.
	// Indicates which outputs the driver should return the contents of in the OperationResult.
	Outputs map[string]string `json:"outputs"`
	// Output stream for log messages from the driver
	Out io.Writer `json:"-"`
	// Output stream for error messages from the driver
	Err io.Writer `json:"-"`
	// Bundle represents the bundle information for use by the operation
	Bundle *bundle.Bundle
}

// ResolvedCred is a credential that has been resolved and is ready for injection into the runtime.
type ResolvedCred struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// OperationResult is the output of the Driver running an Operation.
type OperationResult struct {
	// Outputs maps from the name of the output to its content.
	Outputs map[string]string

	// Logs is the combined logs from the bundle execution.
	Logs bytes.Buffer

	// Error is any errors from executing the operation.
	Error error
}

// SetDefaultOutputValues for an output when it does not exist and it has a
// non-empty default value.
func (r *OperationResult) SetDefaultOutputValues(op Operation) error {
	if r.Outputs == nil {
		r.Outputs = make(map[string]string)
	}

	for name, output := range op.Bundle.Outputs {
		_, hasOutput := r.Outputs[name]
		if hasOutput || !output.AppliesTo(op.Action) {
			continue
		}

		if outputDefinition, exists := op.Bundle.Definitions[output.Definition]; exists {
			outputDefault := outputDefinition.Default
			if outputDefault != nil {
				contents := fmt.Sprintf("%v", outputDefault)
				r.Outputs[name] = contents
			} else {
				return fmt.Errorf("required output %s is missing and has no default", name)
			}
		}
	}

	return nil
}

// Driver is capable of running a invocation image
type Driver interface {
	// Run executes the operation inside of the invocation image
	Run(*Operation) (OperationResult, error)
	// Handles receives an ImageType* and answers whether this driver supports that type
	Handles(string) bool
}

// Configurable drivers can explain their configuration, and have it explicitly set
type Configurable interface {
	// Config returns a map of configuration names and values that can be set via environment variable
	Config() map[string]string
	// SetConfig allows setting configuration, where name corresponds to the key in Config, and value is
	// the value to be set.
	SetConfig(map[string]string) error
}
