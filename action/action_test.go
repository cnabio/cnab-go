package action

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/bundle/definition"

	"github.com/stretchr/testify/assert"
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

var mockSet = credentials.Set{
	"secret_one": "I'm a secret",
	"secret_two": "I'm also a secret",
}

func newClaim() *claim.Claim {
	now := time.Now()
	return &claim.Claim{
		Created:    now,
		Modified:   now,
		Name:       "name",
		Revision:   "revision",
		Bundle:     mockBundle(),
		Parameters: map[string]interface{}{},
	}
}

func mockBundle() *bundle.Bundle {
	return &bundle.Bundle{
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
	c := newClaim()
	c.Parameters = map[string]interface{}{
		"param_one":   "oneval",
		"param_two":   "twoval",
		"param_three": "threeval",
		"param_array": []string{"first-value", "second-value"},
		"param_object": map[string]string{
			"first-key":  "first-value",
			"second-key": "second-value",
		},
		"param_escaped_quotes": `\"escaped value\"`,
		"param_quoted_string":  `"quoted value"`,
	}
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Name, op.Installation)
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
	is.Contains(op.Outputs, "/tmp/some/path")

	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/app/image-map.json"]), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)

	var bundle *bundle.Bundle
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/bundle.json"]), &bundle))
	is.Equal(c.Bundle, bundle)

	is.Len(op.Parameters, 7)
	is.Nil(op.Out)
}

func TestOpFromClaim_NoOutputsOnBundle(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()
	c.Bundle.Outputs = nil
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Name, op.Installation)
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
	c := newClaim()
	c.Bundle = mockBundle()
	c.Bundle.Parameters = nil
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Name, op.Installation)
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
	c := newClaim()
	c.Parameters = map[string]interface{}{
		"param_one":         "oneval",
		"param_two":         "twoval",
		"param_three":       "threeval",
		"param_one_million": "this is not a valid parameter",
	}
	invocImage := c.Bundle.InvocationImages[0]

	_, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet)
	assert.Error(t, err)
}

func TestOpFromClaim_MissingRequiredParameter(t *testing.T) {
	c := newClaim()
	c.Parameters = map[string]interface{}{
		"param_two":   "twoval",
		"param_three": "threeval",
	}
	c.Bundle = mockBundle()
	c.Bundle.Parameters["param_one"] = bundle.Parameter{Definition: "ParamOne", Required: true}
	invocImage := c.Bundle.InvocationImages[0]

	t.Run("missing required parameter fails", func(t *testing.T) {
		_, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet)
		assert.EqualError(t, err, `missing required parameter "param_one" for action "install"`)
	})

	t.Run("fill the missing parameter", func(t *testing.T) {
		c.Parameters["param_one"] = "oneval"
		_, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet)
		assert.Nil(t, err)
	})
}

func TestOpFromClaim_MissingRequiredParamSpecificToAction(t *testing.T) {
	c := newClaim()
	c.Parameters = map[string]interface{}{
		"param_one":   "oneval",
		"param_two":   "twoval",
		"param_three": "threeval",
	}
	c.Bundle = mockBundle()
	// Add a required parameter only defined for the test action
	c.Bundle.Parameters["param_test"] = bundle.Parameter{
		Definition: "StringParam",
		Required:   true,
		ApplyTo:    []string{"test"},
	}
	invocImage := c.Bundle.InvocationImages[0]

	t.Run("if param is not required for this action, succeed", func(t *testing.T) {
		_, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet)
		assert.Nil(t, err)
	})

	t.Run("if param is required for this action and is missing, error", func(t *testing.T) {
		_, err := opFromClaim("test", stateful, c, invocImage, mockSet)
		assert.EqualError(t, err, `missing required parameter "param_test" for action "test"`)
	})

	t.Run("if param is required for this action and is set, succeed", func(t *testing.T) {
		c.Parameters["param_test"] = "only for test action"
		_, err := opFromClaim("test", stateful, c, invocImage, mockSet)
		assert.Nil(t, err)
	})
}

func TestSetOutputsOnClaim(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()

	t.Run("any text in a file is a valid string", func(t *testing.T) {
		output := map[string]string{
			"/tmp/some/path": "a valid output",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("a non-string JSON value is still a string", func(t *testing.T) {
		output := map[string]string{
			"/tmp/some/path": "2",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	// Types to check here: "null", "boolean", "object", "array", "number", or "integer"

	// Non strings given a good type should also work
	t.Run("null succeeds", func(t *testing.T) {
		o := c.Bundle.Outputs["some-output"]
		o.Definition = "NullParam"
		c.Bundle.Outputs["some-output"] = o
		output := map[string]string{
			"/tmp/some/path": "null",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("boolean succeeds", func(t *testing.T) {
		o := c.Bundle.Outputs["some-output"]
		o.Definition = "BooleanParam"
		c.Bundle.Outputs["some-output"] = o
		output := map[string]string{
			"/tmp/some/path": "true",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("object succeeds", func(t *testing.T) {
		o := c.Bundle.Outputs["some-output"]
		o.Definition = "ObjectParam"
		c.Bundle.Outputs["some-output"] = o
		output := map[string]string{
			"/tmp/some/path": "{}",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("array succeeds", func(t *testing.T) {
		field := c.Bundle.Outputs["some-output"]
		field.Definition = "ArrayParam"
		c.Bundle.Outputs["some-output"] = field
		output := map[string]string{
			"/tmp/some/path": "[]",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("number succeeds", func(t *testing.T) {
		field := c.Bundle.Outputs["some-output"]
		field.Definition = "NumberParam"
		c.Bundle.Outputs["some-output"] = field
		output := map[string]string{
			"/tmp/some/path": "3.14",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("integer as number succeeds", func(t *testing.T) {
		field := c.Bundle.Outputs["some-output"]
		field.Definition = "NumberParam"
		c.Bundle.Outputs["some-output"] = field
		output := map[string]string{
			"/tmp/some/path": "372",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("integer succeeds", func(t *testing.T) {
		o := c.Bundle.Outputs["some-output"]
		o.Definition = "IntegerParam"
		c.Bundle.Outputs["some-output"] = o
		output := map[string]string{
			"/tmp/some/path": "372",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})
}

func TestSetOutputsOnClaim_MultipleTypes(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()
	o := c.Bundle.Outputs["some-output"]
	o.Definition = "BooleanAndIntegerParam"
	c.Bundle.Outputs["some-output"] = o

	t.Run("BooleanOrInteger, so boolean succeeds", func(t *testing.T) {
		output := map[string]string{
			"/tmp/some/path": "false",
		}

		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("BooleanOrInteger, so integer succeeds", func(t *testing.T) {
		output := map[string]string{
			"/tmp/some/path": "5",
		}

		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})
}

// Tests that strings accept anything even as part of a list of types.
func TestSetOutputsOnClaim_MultipleTypesWithString(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()
	o := c.Bundle.Outputs["some-output"]
	o.Definition = "StringAndBooleanParam"
	c.Bundle.Outputs["some-output"] = o

	t.Run("null succeeds", func(t *testing.T) {
		output := map[string]string{
			"/tmp/some/path": "null",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})

	t.Run("non-json string succeeds", func(t *testing.T) {
		output := map[string]string{
			"/tmp/some/path": "XYZ is not a JSON value",
		}
		outputErrors := setOutputsOnClaim(c, output)
		assert.NoError(t, outputErrors)
	})
}

func TestSetOutputsOnClaim_MismatchType(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()

	o := c.Bundle.Outputs["some-output"]
	o.Definition = "BooleanParam"
	c.Bundle.Outputs["some-output"] = o

	t.Run("error case: content type does not match output definition", func(t *testing.T) {
		invalidParsableOutput := map[string]string{
			"/tmp/some/path": "2",
		}

		outputErrors := setOutputsOnClaim(c, invalidParsableOutput)
		assert.EqualError(t, outputErrors, `error: ["some-output" is not any of the expected types (boolean) because it is "integer"]`)
	})

	t.Run("error case: content is not valid JSON and definition is not string", func(t *testing.T) {
		invalidNonParsableOutput := map[string]string{
			"/tmp/some/path": "Not a boolean",
		}

		outputErrors := setOutputsOnClaim(c, invalidNonParsableOutput)
		assert.EqualError(t, outputErrors, `error: [failed to parse "some-output": invalid character 'N' looking for beginning of value]`)
	})
}

func TestSelectInvocationImage_EmptyInvocationImages(t *testing.T) {
	c := &claim.Claim{
		Bundle: &bundle.Bundle{},
	}
	_, err := selectInvocationImage(&driver.DebugDriver{}, c)
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
	c := &claim.Claim{
		Bundle: mockBundle(),
	}
	_, err := selectInvocationImage(&mockDriver{Error: errors.New("I always fail")}, c)
	if err == nil {
		t.Fatal("expected an error")
	}
	want := "driver is not compatible"
	got := err.Error()
	if !strings.Contains(got, want) {
		t.Fatalf("expected an error containing %q but got %q", want, got)
	}
}
