package bundle

import "testing"

func TestOutputAppliesTo(t *testing.T) {
	o := Output{}

	// By default, output will apply to any action
	if !o.AppliesTo("install") {
		t.Errorf("Expected parameter to apply to action: install")
	}

	if !o.AppliesTo("status") {
		t.Errorf("Expected parameter to apply to action: status")
	}

	o.ApplyTo = []string{
		"install",
		"uninstall",
	}

	if !o.AppliesTo("install") {
		t.Errorf("Expected parameter to apply to action: install")
	}

	if o.AppliesTo("status") {
		t.Errorf("Expected parameter to not apply to action: status")
	}
}
