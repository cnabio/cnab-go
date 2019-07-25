package action

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/bundle/definition"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDriver struct {
	shouldHandle bool
	Result       driver.OperationResult
	Error        error
}

func (d *mockDriver) Handles(imageType string) bool {
	return d.shouldHandle
}
func (d *mockDriver) Run(op *driver.Operation) (driver.OperationResult, error) {
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
				Type:    "null",
				Default: true,
			},
			"BooleanParam": {
				Type:    "boolean",
				Default: true,
			},
			"ObjectParam": {
				Type:    "object",
				Default: true,
			},
			"ArrayParam": {
				Type:    "array",
				Default: true,
			},
			"NumberParam": {
				Type:    "number",
				Default: true,
			},
			"IntegerParam": {
				Type:    "integer",
				Default: true,
			},
			"BooleanAndIntegerParam": {
				Type: []interface{}{"boolean", "integer"},
			},
			"StringAndBooleanParam": {
				Type: []interface{}{"string", "boolean"},
			},
		},
		Outputs: &bundle.OutputsDefinition{
			Fields: map[string]bundle.OutputDefinition{
				"some-output": {
					Path:       "/tmp/some/path",
					Definition: "ParamOne",
				},
			},
		},
		Parameters: &bundle.ParametersDefinition{
			Fields: map[string]bundle.ParameterDefinition{
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
	}
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet, os.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Name, op.Installation)
	is.Equal(c.Revision, op.Revision)
	is.Equal(invocImage.Image, op.Image)
	is.Equal(driver.ImageTypeDocker, op.ImageType)
	is.Equal(op.Environment["SECRET_ONE"], "I'm a secret")
	is.Equal(op.Environment["PARAM_TWO"], "twoval")
	is.Equal(op.Environment["CNAB_P_PARAM_ONE"], "oneval")
	is.Equal(op.Files["/secret/two"], "I'm also a secret")
	is.Equal(op.Files["/param/three"], "threeval")
	is.Contains(op.Files, "/cnab/app/image-map.json")
	is.Contains(op.Outputs, "/tmp/some/path")
	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/app/image-map.json"]), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)
	is.Len(op.Parameters, 3)
	is.Equal(os.Stdout, op.Out)
}

func TestOpFromClaim_NoOutputsOnBundle(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()
	c.Bundle.Outputs = nil
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet, os.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Name, op.Installation)
	is.Equal(c.Revision, op.Revision)
	is.Equal(invocImage.Image, op.Image)
	is.Equal(driver.ImageTypeDocker, op.ImageType)
	is.Equal(op.Environment["SECRET_ONE"], "I'm a secret")
	is.Equal(op.Files["/secret/two"], "I'm also a secret")
	is.Contains(op.Files, "/cnab/app/image-map.json")
	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/app/image-map.json"]), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)
	is.Len(op.Parameters, 0)
	is.Equal(os.Stdout, op.Out)
}

func TestOpFromClaim_NoParameter(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()
	c.Bundle.Parameters = nil
	invocImage := c.Bundle.InvocationImages[0]

	op, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet, os.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	is := assert.New(t)

	is.Equal(c.Name, op.Installation)
	is.Equal(c.Revision, op.Revision)
	is.Equal(invocImage.Image, op.Image)
	is.Equal(driver.ImageTypeDocker, op.ImageType)
	is.Equal(op.Environment["SECRET_ONE"], "I'm a secret")
	is.Equal(op.Files["/secret/two"], "I'm also a secret")
	is.Contains(op.Files, "/cnab/app/image-map.json")
	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/app/image-map.json"]), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)
	is.Len(op.Parameters, 0)
	is.Equal(os.Stdout, op.Out)
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

	_, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet, os.Stdout)
	assert.Error(t, err)
}

func TestOpFromClaim_MissingRequiredParameter(t *testing.T) {
	c := newClaim()
	c.Parameters = map[string]interface{}{
		"param_two":   "twoval",
		"param_three": "threeval",
	}
	c.Bundle = mockBundle()
	c.Bundle.Parameters.Required = []string{"param_one"}
	c.Bundle.Parameters.Fields["param_one"] = bundle.ParameterDefinition{}
	invocImage := c.Bundle.InvocationImages[0]

	// missing required parameter fails
	_, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet, os.Stdout)
	assert.EqualError(t, err, `missing required parameter "param_one" for action "install"`)

	// fill the missing parameter
	c.Parameters["param_one"] = "oneval"
	_, err = opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet, os.Stdout)
	assert.Nil(t, err)
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
	c.Bundle.Parameters.Fields["param_test"] = bundle.ParameterDefinition{
		ApplyTo: []string{"test"},
	}
	c.Bundle.Parameters.Required = []string{"param_test"}
	invocImage := c.Bundle.InvocationImages[0]

	// calling install action without the test required parameter for test action is ok
	_, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet, os.Stdout)
	assert.Nil(t, err)

	// test action needs the required parameter
	_, err = opFromClaim("test", stateful, c, invocImage, mockSet, os.Stdout)
	assert.EqualError(t, err, `missing required parameter "param_test" for action "test"`)

	c.Parameters["param_test"] = "only for test action"
	_, err = opFromClaim("test", stateful, c, invocImage, mockSet, os.Stdout)
	assert.Nil(t, err)
}

func TestSetOutputsOnClaim(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()

	// Just some text in a file is a valid string
	output := map[string]string{
		"/tmp/some/path": "a valid output",
	}
	outputErrors := setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	// Accidentally only putting a real JSON value in your "string" file is still a string
	output = map[string]string{
		"/tmp/some/path": "2",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	// Non strings given a good type should also work
	// Types to check here: "null", "boolean", "object", "array", "number", or "integer"

	field := c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "NullParam"
	c.Bundle.Outputs.Fields["some-output"] = field
	output = map[string]string{
		"/tmp/some/path": "null",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	field = c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "BooleanParam"
	c.Bundle.Outputs.Fields["some-output"] = field
	output = map[string]string{
		"/tmp/some/path": "true",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	field = c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "ObjectParam"
	c.Bundle.Outputs.Fields["some-output"] = field
	output = map[string]string{
		"/tmp/some/path": "{}",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	field = c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "ArrayParam"
	c.Bundle.Outputs.Fields["some-output"] = field
	output = map[string]string{
		"/tmp/some/path": "[]",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	field = c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "NumberParam"
	c.Bundle.Outputs.Fields["some-output"] = field
	output = map[string]string{
		"/tmp/some/path": "3.14",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	field = c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "IntegerParam"
	c.Bundle.Outputs.Fields["some-output"] = field
	output = map[string]string{
		"/tmp/some/path": "372",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)
}

func TestSetOutputsOnClaim_MultipleTypes(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()

	// accept boolean or integer
	field := c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "BooleanAndIntegerParam"
	c.Bundle.Outputs.Fields["some-output"] = field

	// passes with boolean
	output := map[string]string{
		"/tmp/some/path": "false",
	}

	outputErrors := setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	// passes with integer
	output = map[string]string{
		"/tmp/some/path": "5",
	}

	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	// accept string or integer
	field = c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "StringAndBooleanParam"
	c.Bundle.Outputs.Fields["some-output"] = field

	// passes with null because string accept anything
	output = map[string]string{
		"/tmp/some/path": "null",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)

	// passes with non-json string because string accept anything
	output = map[string]string{
		"/tmp/some/path": "XYZ is not a JSON value",
	}
	outputErrors = setOutputsOnClaim(c, output)
	assert.NoError(t, outputErrors)
}

func TestSetOutputsOnClaim_MismatchType(t *testing.T) {
	c := newClaim()
	c.Bundle = mockBundle()

	field := c.Bundle.Outputs.Fields["some-output"]
	field.Definition = "BooleanParam"
	c.Bundle.Outputs.Fields["some-output"] = field

	// error case: output content type does not match output definition
	invalidParsableOutput := map[string]string{
		"/tmp/some/path": "2",
	}

	outputErrors := setOutputsOnClaim(c, invalidParsableOutput)
	assert.EqualError(t, outputErrors, `error: ["some-output" is not any of the expected types (boolean) because it is "integer"]`)

	// error case: output content is not valid JSON and the definition is not a string
	invalidNonParsableOutput := map[string]string{
		"/tmp/some/path": "Not a boolean",
	}

	outputErrors = setOutputsOnClaim(c, invalidNonParsableOutput)
	assert.EqualError(t, outputErrors, `error: [failed to parse "some-output": invalid character 'N' looking for beginning of value]`)
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

func TestSelectInvocationImage_MapHaveImages_NotPresentInMap(t *testing.T) {
	c := &claim.Claim{
		Bundle: mockBundle(),
		RelocationMap: bundle.ImageRelocationMap{
			"some-image": "some-other-image",
		},
	}
	invImage, err := selectInvocationImage(&driver.DebugDriver{}, c)
	require.NoError(t, err)

	assert.Equal(t, "foo/bar:0.1.0", invImage.Image)
}

func TestSelectInvocationImage_MapHaveImages_IsPresentMap_returnsNewImageTag(t *testing.T) {
	c := &claim.Claim{
		Bundle: mockBundle(),
		RelocationMap: bundle.ImageRelocationMap{
			"foo/bar:0.1.0": "some/other:1.0",
		},
	}
	invImage, err := selectInvocationImage(&driver.DebugDriver{}, c)
	require.NoError(t, err)

	assert.Equal(t, "some/other:1.0", invImage.Image)
}
