package action

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/driver"
	"github.com/cnabio/cnab-go/driver/debug"
	"github.com/cnabio/cnab-go/valuesource"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDriver struct {
	shouldHandle bool
	Operation    *driver.Operation
	Result       driver.OperationResult
	Error        error
}

func (d *mockDriver) Handles(imageType string) bool {
	return d.shouldHandle
}
func (d *mockDriver) Run(op *driver.Operation) (driver.OperationResult, error) {
	d.Operation = op
	return d.Result, d.Error
}

var mockSet = valuesource.Set{
	"secret_one": "I'm a secret",
	"secret_two": "I'm also a secret",
}

const (
	someContent       = "SOME CONTENT"
	someContentDigest = "sha256:296580c88c4c54fb13cf0458d7b490bd8cec3f87e0dffd5c7b6bb4b66bfdf825"
)

func newClaim(action string) claim.Claim {
	now := time.Now()
	schemaVersion, _ := claim.GetDefaultSchemaVersion()
	return claim.Claim{
		SchemaVersion: schemaVersion,
		ID:            "id",
		Created:       now,
		Installation:  "name",
		Action:        action,
		Revision:      "revision",
		Bundle:        mockBundle(),
		Parameters:    map[string]interface{}{},
	}
}

func mockBundle() bundle.Bundle {
	return bundle.Bundle{
		Name:    "bar",
		Version: "0.1.0",
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{Image: "foo/bar:0.1.0", ImageType: "docker"},
			},
		},
		Credentials: map[string]bundle.Credential{
			"secret_one": {
				Location: bundle.Location{
					EnvironmentVariable: "SECRET_ONE",
					Path:                "/foo/bar",
				},
			},
			"secret_two": {
				Location: bundle.Location{
					EnvironmentVariable: "SECRET_TWO",
					Path:                "/secret/two",
				},
			},
		},
		Definitions: map[string]*definition.Schema{
			"ParamOne": {
				Type:    "string",
				Default: "one",
			},
			"ParamTwo": {
				Type:    "string",
				Default: "two",
			},
			"ParamThree": {
				Type:    "string",
				Default: "three",
			},
			"NullParam": {
				Type: "null",
			},
			"BooleanParam": {
				Type:    "boolean",
				Default: true,
			},
			"ObjectParam": {
				Type: "object",
			},
			"ArrayParam": {
				Type: "array",
			},
			"NumberParam": {
				Type: "number",
			},
			"IntegerParam": {
				Type: "integer",
			},
			"StringParam": {
				Type: "string",
			},
			"BooleanAndIntegerParam": {
				Type: []interface{}{"boolean", "integer"},
			},
			"StringAndBooleanParam": {
				Type: []interface{}{"string", "boolean"},
			},
		},
		Outputs: map[string]bundle.Output{
			"some-output": {
				Path:       "/tmp/some/path",
				Definition: "ParamOne",
			},
		},
		Parameters: map[string]bundle.Parameter{
			"param_one": {
				Definition: "ParamOne",
			},
			"param_two": {
				Definition: "ParamTwo",
				Destination: &bundle.Location{
					EnvironmentVariable: "PARAM_TWO",
				},
			},
			"param_three": {
				Definition: "ParamThree",
				Destination: &bundle.Location{
					Path: "/param/three",
				},
			},
			"param_array": {
				Definition: "ArrayParam",
				Destination: &bundle.Location{
					Path: "/param/array",
				},
			},
			"param_object": {
				Definition: "ObjectParam",
				Destination: &bundle.Location{
					Path: "/param/object",
				},
			},
			"param_escaped_quotes": {
				Definition: "StringParam",
				Destination: &bundle.Location{
					Path: "/param/param_escaped_quotes",
				},
			},
			"param_quoted_string": {
				Definition: "StringParam",
				Destination: &bundle.Location{
					Path: "/param/param_quoted_string",
				},
			},
		},
		Actions: map[string]bundle.Action{
			"logs": {Modifies: false},
			"test": {Modifies: true},
		},
		Images: map[string]bundle.Image{
			"image-a": {
				BaseImage: bundle.BaseImage{
					Image: "foo/bar:0.1.0", ImageType: "docker",
				},
				Description: "description",
			},
		},
	}
}

func TestOpFromClaim(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	// the monotonic clock reading from time.Now() proves problematic
	// (it is lost after json.Unmarshal), so just set to static date for testing
	created := time.Date(2020, time.March, 3, 1, 2, 3, 4, time.UTC)
	c.Created = created
	c.Parameters = map[string]interface{}{
		"param_one":   "oneval",
		"param_two":   "twoval",
		"param_three": "threeval",
		"param_array": []interface{}{"first-value", "second-value"},
		"param_object": map[string]interface{}{
			"first-key":  "first-value",
			"second-key": "second-value",
		},
		"param_escaped_quotes": `\"escaped value\"`,
		"param_quoted_string":  `"quoted value"`,
	}
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(stateful, c, invocImage, mockSet)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Installation, op.Installation)
	is.Equal(c.Revision, op.Revision)
	is.Equal(invocImage.Image, op.Image.Image)
	is.Equal(driver.ImageTypeDocker, op.Image.ImageType)
	is.Equal(op.Environment["SECRET_ONE"], "I'm a secret")
	is.Equal(op.Environment["PARAM_TWO"], "twoval")
	is.Equal(op.Environment["CNAB_P_PARAM_ONE"], "oneval")
	is.Equal(op.Files["/secret/two"], "I'm also a secret")
	is.Equal(op.Files["/param/three"], "threeval")
	is.Equal(op.Files["/param/array"], "[\"first-value\",\"second-value\"]")
	is.Equal(op.Files["/param/object"], `{"first-key":"first-value","second-key":"second-value"}`)
	is.Equal(op.Files["/param/param_escaped_quotes"], `\"escaped value\"`)
	is.Equal(op.Files["/param/param_quoted_string"], `"quoted value"`)
	is.Contains(op.Files, "/cnab/app/image-map.json")
	is.Contains(op.Files, "/cnab/bundle.json")
	is.Contains(op.Files, "/cnab/claim.json")
	is.Contains(op.Outputs, "/tmp/some/path")

	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/app/image-map.json"]), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)

	var bundle bundle.Bundle
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/bundle.json"]), &bundle))
	is.Equal(c.Bundle, bundle)

	var claim claim.Claim
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/claim.json"]), &claim))
	is.Equal(c, claim)

	is.Len(op.Parameters, 7)
	is.Nil(op.Out)
}

func TestOpFromClaim_NoOutputsOnBundle(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	c.Bundle.Outputs = nil
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(stateful, c, invocImage, mockSet)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Installation, op.Installation)
	is.Equal(c.Revision, op.Revision)
	is.Equal(invocImage.Image, op.Image.Image)
	is.Equal(driver.ImageTypeDocker, op.Image.ImageType)
	is.Equal(op.Environment["SECRET_ONE"], "I'm a secret")
	is.Equal(op.Files["/secret/two"], "I'm also a secret")
	is.Contains(op.Files, "/cnab/app/image-map.json")
	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/app/image-map.json"]), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)
	is.Len(op.Parameters, 0)
	is.Nil(op.Out)
}

func TestOpFromClaim_NoParameter(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	c.Bundle.Parameters = nil
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(stateful, c, invocImage, mockSet)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Installation, op.Installation)
	is.Equal(c.Revision, op.Revision)
	is.Equal(invocImage.Image, op.Image.Image)
	is.Equal(driver.ImageTypeDocker, op.Image.ImageType)
	is.Equal(op.Environment["SECRET_ONE"], "I'm a secret")
	is.Equal(op.Files["/secret/two"], "I'm also a secret")
	is.Contains(op.Files, "/cnab/app/image-map.json")
	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/app/image-map.json"]), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)
	is.Len(op.Parameters, 0)
	is.Nil(op.Out)
}

func TestOpFromClaim_UndefinedParams(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	c.Parameters = map[string]interface{}{
		"param_one":         "oneval",
		"param_two":         "twoval",
		"param_three":       "threeval",
		"param_one_million": "this is not a valid parameter",
	}
	invocImage := c.Bundle.InvocationImages[0]

	_, err := opFromClaim(stateful, c, invocImage, mockSet)
	require.Error(t, err)
}

func TestOpFromClaim_MissingRequiredParameter(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	c.Parameters = map[string]interface{}{
		"param_two":   "twoval",
		"param_three": "threeval",
	}
	c.Bundle.Parameters["param_one"] = bundle.Parameter{Definition: "ParamOne", Required: true}
	invocImage := c.Bundle.InvocationImages[0]

	t.Run("missing required parameter fails", func(t *testing.T) {
		_, err := opFromClaim(stateful, c, invocImage, mockSet)
		assert.EqualError(t, err, `missing required parameter "param_one" for action "install"`)
	})

	t.Run("fill the missing parameter", func(t *testing.T) {
		c.Parameters["param_one"] = "oneval"
		_, err := opFromClaim(stateful, c, invocImage, mockSet)
		assert.Nil(t, err)
	})
}

func TestOpFromClaim_MissingRequiredParamSpecificToAction(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	c.Parameters = map[string]interface{}{
		"param_one":   "oneval",
		"param_two":   "twoval",
		"param_three": "threeval",
	}
	// Add a required parameter only defined for the test action
	c.Bundle.Parameters["param_test"] = bundle.Parameter{
		Definition: "StringParam",
		Required:   true,
		ApplyTo:    []string{"test"},
	}
	invocImage := c.Bundle.InvocationImages[0]

	t.Run("if param is not required for this action, succeed", func(t *testing.T) {
		_, err := opFromClaim(stateful, c, invocImage, mockSet)
		assert.Nil(t, err)
	})

	t.Run("if param is required for this action and is missing, error", func(t *testing.T) {
		c.Action = "test"
		_, err := opFromClaim(stateful, c, invocImage, mockSet)
		assert.EqualError(t, err, `missing required parameter "param_test" for action "test"`)
	})

	t.Run("if param is required for this action and is set, succeed", func(t *testing.T) {
		c.Action = "test"
		c.Parameters["param_test"] = "only for test action"
		_, err := opFromClaim(stateful, c, invocImage, mockSet)
		assert.Nil(t, err)
	})
}

func TestOpFromClaim_NotApplicableToAction(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	invocImage := c.Bundle.InvocationImages[0]

	c.Bundle.Outputs = map[string]bundle.Output{
		"some-output": {
			Path:    "/path/to/some-output",
			ApplyTo: []string{"install"},
		},
	}

	t.Run("output is added to the operation when it applies to the action", func(t *testing.T) {
		op, err := opFromClaim(stateful, c, invocImage, mockSet)
		require.NoError(t, err)
		gotOutputs := op.Outputs
		assert.Contains(t, gotOutputs, "/path/to/some-output", "some-output should be listed in op.Outputs")
	})

	t.Run("output not added to the operation when it doesn't apply to the action", func(t *testing.T) {
		c.Action = claim.ActionUninstall
		op, err := opFromClaim(stateful, c, invocImage, mockSet)
		require.NoError(t, err)
		gotOutputs := op.Outputs
		assert.NotContains(t, gotOutputs, "/path/to/some-output", "some-output should not be listed in op.Outputs")
	})
}

func TestOpFromClaim_Environment(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	invocImage := c.Bundle.InvocationImages[0]

	schemaVersion, _ := claim.GetDefaultSchemaVersion()
	expectedEnv := map[string]string{
		"CNAB_ACTION":            "install",
		"CNAB_BUNDLE_NAME":       "bar",
		"CNAB_BUNDLE_VERSION":    "0.1.0",
		"CNAB_CLAIMS_VERSION":    string(schemaVersion),
		"CNAB_INSTALLATION_NAME": "name",
		"CNAB_REVISION":          "revision",
		"SECRET_ONE":             "I'm a secret",
		"SECRET_TWO":             "I'm also a secret",
	}

	op, err := opFromClaim(stateful, c, invocImage, mockSet)
	require.NoError(t, err)
	assert.Equal(t, expectedEnv, op.Environment, "operation env does not match expected")
}

func TestSetOutputsOnResult(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	r, err := c.NewResult(claim.StatusSucceeded)
	require.NoError(t, err, "NewResult failed")

	t.Run("any text in a file is a valid string", func(t *testing.T) {
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "a valid output",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("a non-string JSON value is still a string", func(t *testing.T) {
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "2",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	// Types to check here: "null", "boolean", "object", "array", "number", or "integer"

	// Non strings given a good type should also work
	t.Run("null succeeds", func(t *testing.T) {
		o := c.Bundle.Outputs["some-output"]
		o.Definition = "NullParam"
		c.Bundle.Outputs["some-output"] = o
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "null",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("boolean succeeds", func(t *testing.T) {
		o := c.Bundle.Outputs["some-output"]
		o.Definition = "BooleanParam"
		c.Bundle.Outputs["some-output"] = o
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "true",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("object succeeds", func(t *testing.T) {
		o := c.Bundle.Outputs["some-output"]
		o.Definition = "ObjectParam"
		c.Bundle.Outputs["some-output"] = o
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "{}",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("array succeeds", func(t *testing.T) {
		field := c.Bundle.Outputs["some-output"]
		field.Definition = "ArrayParam"
		c.Bundle.Outputs["some-output"] = field
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "[]",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("number succeeds", func(t *testing.T) {
		field := c.Bundle.Outputs["some-output"]
		field.Definition = "NumberParam"
		c.Bundle.Outputs["some-output"] = field
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "3.14",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("integer as number succeeds", func(t *testing.T) {
		field := c.Bundle.Outputs["some-output"]
		field.Definition = "NumberParam"
		c.Bundle.Outputs["some-output"] = field
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "372",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("integer succeeds", func(t *testing.T) {
		o := c.Bundle.Outputs["some-output"]
		o.Definition = "IntegerParam"
		c.Bundle.Outputs["some-output"] = o
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "372",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})
}

func TestSetOutputsOnClaim_MultipleTypes(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	o := c.Bundle.Outputs["some-output"]
	o.Definition = "BooleanAndIntegerParam"
	c.Bundle.Outputs["some-output"] = o

	r, err := c.NewResult(claim.StatusSucceeded)
	require.NoError(t, err, "NewResult failed")

	t.Run("BooleanOrInteger, so boolean succeeds", func(t *testing.T) {
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "false",
			},
		}

		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("BooleanOrInteger, so integer succeeds", func(t *testing.T) {
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "5",
			},
		}

		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})
}

// Tests that strings accept anything even as part of a list of types.
func TestSetOutputsOnClaim_MultipleTypesWithString(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	o := c.Bundle.Outputs["some-output"]
	o.Definition = "StringAndBooleanParam"
	c.Bundle.Outputs["some-output"] = o

	r, err := c.NewResult(claim.StatusSucceeded)
	require.NoError(t, err, "NewResult failed")

	t.Run("null succeeds", func(t *testing.T) {
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "null",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})

	t.Run("non-json string succeeds", func(t *testing.T) {
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "XYZ is not a JSON value",
			},
		}
		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		require.NoError(t, outputErrors)
	})
}

func TestSetOutputsOnClaim_MismatchType(t *testing.T) {
	c := newClaim(claim.ActionInstall)
	o := c.Bundle.Outputs["some-output"]
	o.Definition = "BooleanParam"
	c.Bundle.Outputs["some-output"] = o

	r, err := c.NewResult(claim.StatusSucceeded)
	require.NoError(t, err, "NewResult failed")

	t.Run("error case: content type does not match output definition", func(t *testing.T) {
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "2",
			},
		}

		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		assert.EqualError(t, outputErrors, `error: ["some-output" is not any of the expected types (boolean) because it is "integer"]`)
	})

	t.Run("error case: content is not valid JSON and definition is not string", func(t *testing.T) {
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": "Not a boolean",
			},
		}

		outputErrors := setOutputsOnClaimResult(c, &r, opResult)
		assert.EqualError(t, outputErrors, `error: [failed to parse "some-output": invalid character 'N' looking for beginning of value]`)
	})
}

func TestSelectInvocationImage_EmptyInvocationImages(t *testing.T) {
	d := &debug.Driver{}
	a := New(d, nil)
	c := claim.Claim{
		Bundle: bundle.Bundle{},
	}
	_, err := a.selectInvocationImage(c)
	if err == nil {
		t.Fatal("expected an error")
	}
	want := "no invocationImages are defined"
	got := err.Error()
	if !strings.Contains(got, want) {
		t.Fatalf("expected an error containing %q but got %q", want, got)
	}
}

func TestSelectInvocationImage_DriverIncompatible(t *testing.T) {
	d := &mockDriver{Error: errors.New("I always fail")}
	a := New(d, nil)
	c := claim.Claim{
		Bundle: mockBundle(),
	}
	_, err := a.selectInvocationImage(c)
	if err == nil {
		t.Fatal("expected an error")
	}
	want := "driver is not compatible"
	got := err.Error()
	if !strings.Contains(got, want) {
		t.Fatalf("expected an error containing %q but got %q", want, got)
	}
}

func TestAction_RunAction(t *testing.T) {
	out := func(op *driver.Operation) error {
		op.Out = ioutil.Discard
		return nil
	}

	t.Run("happy-path", func(t *testing.T) {
		c := newClaim(claim.ActionInstall)
		d := &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"some-output": someContent,
				},
			},
			Error: nil,
		}
		inst := New(d, nil)

		opResult, claimResult, err := inst.Run(c, mockSet, out)
		require.NoError(t, err)
		require.NoError(t, opResult.Error)
		assert.Equal(t, claim.ActionInstall, c.Action)
		assert.Equal(t, claim.StatusSucceeded, claimResult.Status)
		assert.Contains(t, opResult.Outputs, "some-output", "the operation result should have captured the output")
		assert.Equal(t, someContent, opResult.Outputs["some-output"], "the operation result should have the output contents")
		contentDigest, _ := claimResult.OutputMetadata.GetContentDigest("some-output")
		assert.Equal(t, someContentDigest, contentDigest, "invalid output content digest")
	})

	t.Run("configure operation", func(t *testing.T) {
		c := newClaim(claim.ActionInstall)
		d := &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"some-output": someContent,
				},
			},
			Error: nil,
		}
		inst := New(d, nil)

		addFile := func(op *driver.Operation) error {
			op.Files["/tmp/another/path"] = "ANOTHER FILE"
			return nil
		}
		_, _, err := inst.Run(c, mockSet, out, addFile)
		require.NoError(t, err)
		assert.Contains(t, d.Operation.Files, "/tmp/another/path")
	})

	t.Run("error case: configure operation", func(t *testing.T) {
		c := newClaim(claim.ActionInstall)
		d := &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"some-output": someContent,
				},
			},
			Error: nil,
		}
		inst := New(d, nil)
		sabotage := func(op *driver.Operation) error {
			return errors.New("oops")
		}
		_, _, err := inst.Run(c, mockSet, out, sabotage)
		require.EqualError(t, err, "oops")
	})

	t.Run("when the bundle has no outputs", func(t *testing.T) {
		c := newClaim(claim.ActionInstall)
		c.Bundle.Outputs = nil
		d := &mockDriver{
			shouldHandle: true,
			Result:       driver.OperationResult{},
			Error:        nil,
		}
		inst := New(d, nil)
		_, claimResult, err := inst.Run(c, mockSet, out)
		require.NoError(t, err)
		assert.Equal(t, claim.ActionInstall, c.Action)
		assert.Equal(t, claim.StatusSucceeded, claimResult.Status)
		assert.Empty(t, claimResult.OutputMetadata)
	})

	t.Run("when an output with a default isn't generated", func(t *testing.T) {
		c := newClaim(claim.ActionInstall)

		// Add an output with a default value, the mock driver doesn't generate outputs
		// it only prints the bundle definition
		c.Bundle.Outputs["hasDefault1"] = bundle.Output{
			Definition: "hasDefault1",
		}
		c.Bundle.Definitions["hasDefault1"] = &definition.Schema{
			Default: "some default1",
		}

		// This output applies to the active action and should also be defaulted
		c.Bundle.Outputs["hasDefault2"] = bundle.Output{
			ApplyTo:    []string{claim.ActionInstall},
			Definition: "hasDefault2",
		}
		c.Bundle.Definitions["hasDefault2"] = &definition.Schema{
			Default: "some default2",
		}

		// This output does NOT apply and should NOT be defaulted
		c.Bundle.Outputs["hasDefault3"] = bundle.Output{
			ApplyTo:    []string{claim.ActionUpgrade},
			Definition: "hasDefault3",
		}
		c.Bundle.Definitions["hasDefault3"] = &definition.Schema{
			Default: "some default3",
		}

		d := &mockDriver{
			shouldHandle: true,
			Result:       driver.OperationResult{},
			Error:        nil,
		}
		inst := New(d, nil)
		opResult, _, err := inst.Run(c, mockSet, out)
		require.NoError(t, err)

		assert.Contains(t, opResult.Outputs, "hasDefault1", "the output always applies so an output value should have been set")
		assert.Equal(t, "some default1", opResult.Outputs["hasDefault1"], "the output value should be the bundle default")

		assert.Contains(t, opResult.Outputs, "hasDefault2", "the output applies to the install action so an output value should have been set")
		assert.Equal(t, "some default2", opResult.Outputs["hasDefault2"], "the output value should be the bundle default")

		assert.NotContains(t, opResult.Outputs, "hasDefault3", "the output applies only to upgrade so an output value should not have been set")
	})

	t.Run("error case: required output not generated", func(t *testing.T) {
		c := newClaim(claim.ActionInstall)

		// Add a required output that the runtime cannot default for us (and the mock driver won't generate)
		c.Bundle.Outputs["noDefault"] = bundle.Output{Definition: "noDefault"}
		c.Bundle.Definitions["noDefault"] = &definition.Schema{Type: "string"}

		d := &mockDriver{
			shouldHandle: true,
			Result:       driver.OperationResult{},
			Error:        nil,
		}
		inst := New(d, nil)
		opResult, _, err := inst.Run(c, mockSet, out)
		require.NoError(t, err)
		require.Contains(t, opResult.Error.Error(), "required output noDefault is missing and has no default")
	})

	t.Run("error case: driver can't handle image", func(t *testing.T) {
		c := newClaim(claim.ActionInstall)
		d := &mockDriver{
			shouldHandle: false,
			Error:        errors.New("I always fail"),
		}
		inst := New(d, nil)
		_, _, err := inst.Run(c, mockSet, out)
		require.Error(t, err)
	})

	t.Run("error case: driver returns error", func(t *testing.T) {
		c := newClaim(claim.ActionInstall)
		d := &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"some-output": someContent,
				},
			},
			Error: errors.New("I always fail"),
		}
		inst := New(d, nil)
		opResult, claimResult, err := inst.Run(c, mockSet, out)
		require.NoError(t, err)
		require.Contains(t, opResult.Error.Error(), "I always fail")
		assert.Equal(t, claim.ActionInstall, c.Action)
		assert.Equal(t, claim.StatusFailed, claimResult.Status)
		contentDigest, _ := claimResult.OutputMetadata.GetContentDigest("some-output")
		assert.Equal(t, someContentDigest, contentDigest, "invalid output content digest")
	})

	t.Run("error case: unknown actions should fail", func(t *testing.T) {
		c := newClaim("missing")
		d := &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"some-output": someContent,
				},
			},
			Error: errors.New("I always fail"),
		}
		inst := New(d, nil)
		opResult, claimResult, err := inst.Run(c, mockSet, out)
		require.Error(t, err, "Unknown action should fail")
		require.NoError(t, opResult.Error)
		assert.Empty(t, claimResult)
	})
}

func TestAction_ShouldSaveOutput(t *testing.T) {
	t.Run("save all", func(t *testing.T) {
		a := Action{
			SaveAllOutputs: true,
			SaveOutputs:    []string{"output1", "output3"},
		}
		result := a.shouldSaveOutput("output1")
		assert.True(t, result)
	})

	t.Run("name in list", func(t *testing.T) {
		a := Action{
			SaveAllOutputs: false,
			SaveOutputs:    []string{"output1", "output3"},
		}
		result := a.shouldSaveOutput("output1")
		assert.True(t, result)
	})

	t.Run("name not list", func(t *testing.T) {
		a := Action{
			SaveAllOutputs: false,
			SaveOutputs:    []string{"output1", "output3"},
		}
		result := a.shouldSaveOutput("output2")
		assert.False(t, result)
	})

	t.Run("save all, name not list", func(t *testing.T) {
		a := Action{
			SaveAllOutputs: true,
			SaveOutputs:    []string{"output1", "output3"},
		}
		result := a.shouldSaveOutput("output2")
		assert.True(t, result)
	})
}

func TestBuildClaimResult(t *testing.T) {
	t.Run("successful operation", func(t *testing.T) {
		updatedClaim := newClaim(claim.ActionInstall)
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": someContent,
			},
		}
		opErr := &multierror.Error{}

		claimResult, err := buildClaimResult(updatedClaim, opResult, opErr)

		require.NoError(t, err, "buildClaimResult failed")
		assert.NoError(t, opErr.ErrorOrNil(), "an error was logged on the operational result")
		assert.Equal(t, claim.StatusSucceeded, claimResult.Status, "the operation should have been recorded as a success")
		assert.Empty(t, claimResult.Message, "an error message was recorded")
		digest, ok := claimResult.OutputMetadata.GetContentDigest("some-output")
		assert.True(t, ok, "the content digest for the output was not recorded")
		assert.Equal(t, someContentDigest, digest, "the content digest for the output was invalid")
	})

	t.Run("failed operation", func(t *testing.T) {
		updatedClaim := newClaim(claim.ActionInstall)
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": someContent,
			},
		}
		opErr := &multierror.Error{
			Errors: []error{errors.New("bundle failed")},
		}

		claimResult, err := buildClaimResult(updatedClaim, opResult, opErr)

		require.NoError(t, err, "buildClaimResult failed")
		assert.Equal(t, claim.StatusFailed, claimResult.Status, "the operation should have been recorded as a failure")
		assert.Contains(t, claimResult.Message, "bundle failed", "the operation error should have been recorded")
		digest, ok := claimResult.OutputMetadata.GetContentDigest("some-output")
		assert.True(t, ok, "the content digest for the output was not recorded")
		assert.Equal(t, someContentDigest, digest, "the content digest for the output was invalid")
	})
}

func TestGetOutputsGeneratedByAction(t *testing.T) {
	b := bundle.Bundle{
		Outputs: map[string]bundle.Output{
			"output1": {
				ApplyTo: []string{"install"},
				Path:    "/cnab/app/outputs/output1",
			},
			"output2": {
				Path: "/cnab/app/outputs/output2",
			},
			"output3": {
				ApplyTo: []string{"install", "upgrade"},
				Path:    "/cnab/app/outputs/output3",
			},
		},
	}

	gotOutputs := getOutputsGeneratedByAction("install", b)
	wantOutputs := map[string]string{
		"/cnab/app/outputs/output1": "output1",
		"/cnab/app/outputs/output2": "output2",
		"/cnab/app/outputs/output3": "output3",
	}
	assert.Equal(t, wantOutputs, gotOutputs)

	gotOutputs = getOutputsGeneratedByAction("custom", b)
	wantOutputs = map[string]string{
		"/cnab/app/outputs/output2": "output2",
	}
	assert.Equal(t, wantOutputs, gotOutputs)

	gotOutputs = getOutputsGeneratedByAction("upgrade", b)
	wantOutputs = map[string]string{
		"/cnab/app/outputs/output2": "output2",
		"/cnab/app/outputs/output3": "output3",
	}
	assert.Equal(t, wantOutputs, gotOutputs)
}

func TestSaveAction(t *testing.T) {
	t.Run("save output", func(t *testing.T) {
		cp := claim.NewMockStore(nil, nil)
		c := newClaim(claim.ActionInstall)
		r, err := c.NewResult(claim.StatusSucceeded)
		require.NoError(t, err, "NewResult failed")

		a := New(nil, cp)
		a.SaveAllOutputs = true
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": someContent,
			},
		}

		err = a.SaveInitialClaim(c, claim.StatusRunning)
		require.NoError(t, err, "SaveInitiaClaim failed")

		err = a.SaveOperationResult(opResult, c, r)
		require.NoError(t, err, "SaveOperationResult failed")

		_, err = cp.ReadClaim(c.ID)
		assert.NoError(t, err, "the claim was not persisted")

		_, err = cp.ReadResult(r.ID)
		assert.NoError(t, err, "the result was not persisted")

		_, err = cp.ReadOutput(c, r, "some-output")
		assert.NoError(t, err, "the output was not persisted")
	})

	t.Run("do not save output", func(t *testing.T) {
		cp := claim.NewMockStore(nil, nil)
		c := newClaim(claim.ActionInstall)
		r, err := c.NewResult(claim.StatusSucceeded)
		require.NoError(t, err, "NewResult failed")

		a := New(nil, cp)
		opResult := driver.OperationResult{
			Outputs: map[string]string{
				"some-output": someContent,
			},
		}
		err = a.SaveInitialClaim(c, claim.StatusRunning)
		require.NoError(t, err, "SaveInitiaClaim failed")

		err = a.SaveOperationResult(opResult, c, r)
		require.NoError(t, err, "SaveOperationResult failed")

		_, err = cp.ReadClaim(c.ID)
		assert.NoError(t, err, "the claim was not persisted")

		_, err = cp.ReadResult(r.ID)
		assert.NoError(t, err, "the result was not persisted")

		_, err = cp.ReadOutput(c, r, "some-output")
		assert.Error(t, err, "the output should NOT be persisted")
	})
}

func TestExpandCredentials(t *testing.T) {
	t.Run("all creds expanded", func(t *testing.T) {
		b := bundle.Bundle{
			Name: "knapsack",
			Credentials: map[string]bundle.Credential{
				"first": {
					Location: bundle.Location{
						EnvironmentVariable: "FIRST_VAR",
					},
				},
				"second": {
					Location: bundle.Location{
						Path: "/second/path",
					},
				},
				"third": {
					Location: bundle.Location{
						EnvironmentVariable: "/THIRD_VAR",
						Path:                "/third/path",
					},
				},
			},
		}

		set := valuesource.Set{
			"first":  "first",
			"second": "second",
			"third":  "third",
		}

		env, path, err := expandCredentials(b, set, false, "install")
		is := assert.New(t)
		is.NoError(err)
		for k, v := range b.Credentials {
			if v.EnvironmentVariable != "" {
				is.Equal(env[v.EnvironmentVariable], set[k])
			}
			if v.Path != "" {
				is.Equal(path[v.Path], set[k])
			}
		}
	})

	t.Run("missing required cred", func(t *testing.T) {
		b := bundle.Bundle{
			Name: "knapsack",
			Credentials: map[string]bundle.Credential{
				"first": {
					Location: bundle.Location{
						EnvironmentVariable: "FIRST_VAR",
					},
					Required: true,
				},
			},
		}
		set := valuesource.Set{}
		_, _, err := expandCredentials(b, set, false, "install")
		assert.EqualError(t, err, `credential "first" is missing from the user-supplied credentials`)
		_, _, err = expandCredentials(b, set, true, "install")
		assert.NoError(t, err)
	})

	t.Run("missing optional cred", func(t *testing.T) {
		b := bundle.Bundle{
			Name: "knapsack",
			Credentials: map[string]bundle.Credential{
				"first": {
					Location: bundle.Location{
						EnvironmentVariable: "FIRST_VAR",
					},
				},
			},
		}
		set := valuesource.Set{}
		_, _, err := expandCredentials(b, set, false, "install")
		assert.NoError(t, err)
		_, _, err = expandCredentials(b, set, true, "install")
		assert.NoError(t, err)
	})

	t.Run("missing cred with ApplyTo", func(t *testing.T) {
		b := bundle.Bundle{
			Name: "knapsack",
			Credentials: map[string]bundle.Credential{
				"first": {
					Location: bundle.Location{
						EnvironmentVariable: "FIRST_VAR",
					},
					Required: true,
					ApplyTo:  []string{"install"},
				},
			},
		}
		set := valuesource.Set{}
		_, _, err := expandCredentials(b, set, false, "install")
		assert.EqualError(t, err, `credential "first" is missing from the user-supplied credentials`)
		_, _, err = expandCredentials(b, set, false, "upgrade")
		assert.NoError(t, err)
	})
}
