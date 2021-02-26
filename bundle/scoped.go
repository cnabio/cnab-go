package bundle

// Scoped represents an item whose scope is limited to a set of actions.
type Scoped interface {
	// GetApplyTo returns the list of applicable actions.
	GetApplyTo() []string
}

// AppliesTo returns a boolean value specifying whether or not
// the scoped item applies to the provided action.
func AppliesTo(s Scoped, action string) bool {
	applyTo := s.GetApplyTo()
	if len(applyTo) == 0 {
		return true
	}
	for _, act := range applyTo {
		if action == act {
			return true
		}
	}
	return false
}
