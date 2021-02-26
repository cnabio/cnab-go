package bundle

import "errors"

// Credential represents the definition of a CNAB credential
type Credential struct {
	Location    `yaml:",inline"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool     `json:"required,omitempty" yaml:"required,omitempty"`
	ApplyTo     []string `json:"applyTo,omitempty" yaml:"applyTo,omitempty"`
}

// GetApplyTo returns the list of actions that the Credential applies to.
func (c *Credential) GetApplyTo() []string {
	return c.ApplyTo
}

// AppliesTo returns a boolean value specifying whether or not
// the Credential applies to the provided action
func (c *Credential) AppliesTo(action string) bool {
	return AppliesTo(c, action)
}

// Validate a Credential
func (c *Credential) Validate() error {
	if c.Location.EnvironmentVariable == "" && c.Location.Path == "" {
		return errors.New("credential env or path must be supplied")
	}
	return c.Location.Validate()
}
