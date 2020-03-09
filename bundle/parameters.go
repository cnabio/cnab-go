package bundle

import "errors"

// Parameter defines a single parameter for a CNAB bundle
type Parameter struct {
	Definition  string    `json:"definition" yaml:"definition"`
	ApplyTo     []string  `json:"applyTo,omitempty" yaml:"applyTo,omitempty"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Destination *Location `json:"destination" yaml:"destination"`
	Required    bool      `json:"required,omitempty" yaml:"required,omitempty"`
}

// AppliesTo returns a boolean value specifying whether or not
// the Parameter applies to the provided action
func (p *Parameter) AppliesTo(action string) bool {
	if len(p.ApplyTo) == 0 {
		return true
	}
	for _, act := range p.ApplyTo {
		if action == act {
			return true
		}
	}
	return false
}

// Validate a Parameter
func (p *Parameter) Validate() error {
	if p.Definition == "" {
		return errors.New("parameter definition must be provided")
	}
	if p.Destination == nil {
		return errors.New("parameter destination must be provided")
	}
	return p.Destination.Validate()
}
