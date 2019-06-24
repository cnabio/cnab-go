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

type mockFailingDriver struct {
	shouldHandle bool
}

var mockSet = credentials.Set{
	"secret_one": "I'm a secret",
	"secret_two": "I'm also a secret",
}

func (d *mockFailingDriver) Handles(imageType string) bool {
	return d.shouldHandle
}
func (d *mockFailingDriver) Run(op *driver.Operation) error {
	return errors.New("I always fail")
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

func getEnv(name string, op *driver.Operation) string {
	for _, env := range op.Environment {
		if env.Name == name {
			return env.Value
		}
	}
	return ""
}

func getFile(path string, op *driver.Operation) []byte {
	for _, file := range op.Files {
		if file.Path == path {
			return file.Content
		}
	}
	return nil
}

func TestOpFromClaim(t *testing.T) {
	now := time.Now()
	c := &claim.Claim{
		Created:  now,
		Modified: now,
		Name:     "name",
		Revision: "revision",
		Bundle:   mockBundle(),
		Parameters: map[string]interface{}{
			"param_one":   "oneval",
			"param_two":   "twoval",
			"param_three": "threeval",
		},
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
	is.Equal(getEnv("SECRET_ONE", op), "I'm a secret")
	is.Equal(getEnv("PARAM_TWO", op), "twoval")
	is.Equal(getEnv("CNAB_P_PARAM_ONE", op), "oneval")
	is.Equal(string(getFile("/secret/two", op)), "I'm also a secret")
	is.Equal(string(getFile("/param/three", op)), "threeval")
	is.NotNil(getFile("/cnab/app/image-map.json", op))
	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal(getFile("/cnab/app/image-map.json", op), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)
	is.Equal(os.Stdout, op.Out)
}

func TestOpFromClaim_NoParameter(t *testing.T) {
	now := time.Now()
	b := mockBundle()
	b.Parameters = nil
	c := &claim.Claim{
		Created:  now,
		Modified: now,
		Name:     "name",
		Revision: "revision",
		Bundle:   b,
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
	is.Equal("I'm a secret", getEnv("SECRET_ONE", op))
	is.Equal("I'm also a secret", string(getFile("/secret/two", op)))
	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal(getFile("/cnab/app/image-map.json", op), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)
	is.Equal(os.Stdout, op.Out)
}
func TestOpFromClaim_UndefinedParams(t *testing.T) {
	now := time.Now()
	c := &claim.Claim{
		Created:  now,
		Modified: now,
		Name:     "name",
		Revision: "revision",
		Bundle:   mockBundle(),
		Parameters: map[string]interface{}{
			"param_one":         "oneval",
			"param_two":         "twoval",
			"param_three":       "threeval",
			"param_one_million": "this is not a valid parameter",
		},
	}
	invocImage := c.Bundle.InvocationImages[0]

	_, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, mockSet, os.Stdout)
	assert.Error(t, err)
}

func TestOpFromClaim_MissingRequiredParameter(t *testing.T) {
	now := time.Now()
	b := mockBundle()
	b.Parameters.Required = []string{"param_one"}
	b.Parameters.Fields["param_one"] = bundle.ParameterDefinition{}

	c := &claim.Claim{
		Created:  now,
		Modified: now,
		Name:     "name",
		Revision: "revision",
		Bundle:   b,
		Parameters: map[string]interface{}{
			"param_two":   "twoval",
			"param_three": "threeval",
		},
	}
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
	now := time.Now()
	b := mockBundle()
	// Add a required parameter only defined for the test action
	b.Parameters.Fields["param_test"] = bundle.ParameterDefinition{
		ApplyTo: []string{"test"},
	}
	b.Parameters.Required = []string{"param_test"}
	c := &claim.Claim{
		Created:  now,
		Modified: now,
		Name:     "name",
		Revision: "revision",
		Bundle:   b,
		Parameters: map[string]interface{}{
			"param_one":   "oneval",
			"param_two":   "twoval",
			"param_three": "threeval",
		},
	}
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
	_, err := selectInvocationImage(&mockFailingDriver{}, c)
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
