package bundle

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type Output struct {
	Definition  string   `json:"definition" yaml:"definition"`
	ApplyTo     []string `json:"applyTo,omitempty" yaml:"applyTo,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Path        string   `json:"path" yaml:"path"`
}

// GetApplyTo returns the list of actions that the Output applies to.
func (o Output) GetApplyTo() []string {
	return o.ApplyTo
}

// AppliesTo returns a boolean value specifying whether or not
// the Output applies to the provided action
func (o Output) AppliesTo(action string) bool {
	return AppliesTo(o, action)
}

// IsOutputSensitive is a convenience function that determines if an output's
// value is sensitive.
func (b Bundle) IsOutputSensitive(outputName string) (bool, error) {
	if output, ok := b.Outputs[outputName]; ok {
		if def, ok := b.Definitions[output.Definition]; ok {
			sensitive := def.WriteOnly != nil && *def.WriteOnly
			return sensitive, nil
		}

		return false, fmt.Errorf("output definition %q not found", output.Definition)
	}

	return false, fmt.Errorf("output %q not defined", outputName)
}

// Validate an Output
func (o *Output) Validate(name string, bun Bundle) error {
	if o.Definition == "" {
		return errors.New("output definition must be provided")
	}

	// Validate default against definition schema, if exists
	schema, ok := bun.Definitions[o.Definition]
	if !ok {
		return fmt.Errorf("unable to find definition for %s", name)
	}
	var valResult *multierror.Error
	if schema.Default != nil {
		valErrs, err := schema.Validate(schema.Default)
		if err != nil {
			valResult = multierror.Append(valResult, errors.Wrapf(err, "encountered an error validating output %s", name))
		}
		for _, valErr := range valErrs {
			valResult = multierror.Append(valResult, fmt.Errorf("encountered an error validating the default value %v for output %q: %v", schema.Default, name, valErr.Error))
		}
	}

	return valResult.ErrorOrNil()
}
