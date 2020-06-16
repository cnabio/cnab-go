package bundle

import (
	"fmt"
)

type Output struct {
	Definition  string   `json:"definition" yaml:"definition"`
	ApplyTo     []string `json:"applyTo,omitempty" yaml:"applyTo,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Path        string   `json:"path" yaml:"path"`
}

// AppliesTo returns a boolean value specifying whether or not
// the Output applies to the provided action
func (output *Output) AppliesTo(action string) bool {
	if len(output.ApplyTo) == 0 {
		return true
	}
	for _, act := range output.ApplyTo {
		if action == act {
			return true
		}
	}
	return false
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
