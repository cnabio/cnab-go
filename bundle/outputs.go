package bundle

import (
	"fmt"
	"path"
	"strings"

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
		if def, ok := b.Definitions[output.Definition]; ok && def != nil {
			sensitive := def.WriteOnly != nil && *def.WriteOnly
			return sensitive, nil
		}

		return false, fmt.Errorf("output definition %q not found", output.Definition)
	}

	return false, fmt.Errorf("output %q not defined", outputName)
}

// outputsDir is the directory that CNAB output paths must live under, per
// the CNAB spec.
const outputsDir = "/cnab/app/outputs/"

// ValidatePath checks that the Output's Path, if set, is a clean, absolute
// path under the CNAB outputs directory. This rejects traversal segments
// ("..") that would otherwise let a bundle read files outside the
// designated outputs location.
func (o Output) ValidatePath() error {
	if o.Path != "" {
		if path.Clean(o.Path) != o.Path || !strings.HasPrefix(o.Path, outputsDir) {
			return fmt.Errorf("path %q must be a clean path under %q", o.Path, outputsDir)
		}
	}
	return nil
}

// Validate an Output
func (o *Output) Validate(name string, bun Bundle) error {
	if o.Definition == "" {
		return errors.New("output definition must be provided")
	}

	if err := o.ValidatePath(); err != nil {
		return fmt.Errorf("output %q has invalid %s", name, err)
	}

	// Validate default against definition schema, if exists
	schema, ok := bun.Definitions[o.Definition]
	if !ok || schema == nil {
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
