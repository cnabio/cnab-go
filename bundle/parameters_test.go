package bundle

import (
	"fmt"
	"testing"
)

func TestCanReadParameterNames(t *testing.T) {
	json := `{
		"parameters": {
			"foo": { },
			"bar": { }
		}
	}`
	definitions, err := Unmarshal([]byte(json))
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions.Parameters) != 2 {
		t.Fatalf("Expected 2 parameter definitons, got %d", len(definitions.Parameters))
	}
	if _, ok := definitions.Parameters["foo"]; !ok {
		t.Errorf("Expected an entry with name 'foo' but didn't get one")
	}
	if _, ok := definitions.Parameters["bar"]; !ok {
		t.Errorf("Expected an entry with name 'bar' but didn't get one")
	}
}

func TestCanReadParameterDefinition(t *testing.T) {
	definition := "cooldef"
	description := "some description"
	action0 := "action0"
	action1 := "action1"
	destinationEnvValue := "BACKEND_PORT"
	destinationPathValue := "/some/path"

	json := fmt.Sprintf(`{
		"parameters": {
			"test": {
				"definition": "%s",
				"destination": {
					"env": "%s",
					"path": "%s"
				},
				"description": "%s",
				"applyTo": [ "%s", "%s" ],
				"required": true
			}
		}
	}`,
		definition, destinationEnvValue, destinationPathValue,
		description, action0, action1)

	definitions, err := Unmarshal([]byte(json))
	if err != nil {
		t.Fatal(err)
	}

	p := definitions.Parameters["test"]
	if p.Definition != definition {
		t.Errorf("Expected definition'%s' but got '%s'", definition, p.Definition)
	}
	if p.Destination.EnvironmentVariable != destinationEnvValue {
		t.Errorf("Expected destination environment value '%s' but got '%s'", destinationEnvValue, p.Destination.EnvironmentVariable)
	}
	if p.Destination.Path != destinationPathValue {
		t.Errorf("Expected destination path value '%s' but got '%s'", destinationPathValue, p.Destination.Path)
	}
	if p.Description != description {
		t.Errorf("Expected description '%s' but got '%s'", description, p.Description)
	}
	if len(p.ApplyTo) != 2 {
		t.Errorf("Expected 2 applyTo actions but got %d", len(p.ApplyTo))
	}
	if p.ApplyTo[0] != action0 {
		t.Errorf("Expected action '%s' but got '%s'", action0, p.ApplyTo[0])
	}
	if p.ApplyTo[1] != action1 {
		t.Errorf("Expected action '%s' but got '%s'", action1, p.ApplyTo[1])
	}
	if !p.Required {
		t.Errorf("Expected parameter to be required")
	}
}

func TestParameterAppliesTo(t *testing.T) {
	p := Parameter{}

	// By default, parameter will apply to any action
	if !p.AppliesTo("install") {
		t.Errorf("Expected parameter to apply to action: install")
	}

	if !p.AppliesTo("status") {
		t.Errorf("Expected parameter to apply to action: status")
	}

	p.ApplyTo = []string{
		"install",
		"uninstall",
	}

	if !p.AppliesTo("install") {
		t.Errorf("Expected parameter to apply to action: install")
	}

	if p.AppliesTo("status") {
		t.Errorf("Expected parameter to not apply to action: status")
	}
}
