package bundle

type Output struct {
	Definition  string   `json:"definition" yaml:"definition"`
	ApplyTo     []string `json:"applyTo,omitempty" yaml:"applyTo,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Path        string   `json:"path" yaml:"path"`
}

// AppliesTo satisfies the ActionApplicable interface
// by determining whether or not the Output applies to the provided action
func (o *Output) AppliesTo(action string) bool {
	if len(o.ApplyTo) == 0 {
		return true
	}
	for _, act := range o.ApplyTo {
		if action == act {
			return true
		}
	}
	return false
}
