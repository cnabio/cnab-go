package action_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"

	"github.com/stretchr/testify/assert"
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
		Credentials: map[string]bundle.Location{
			"secret_one": {
				EnvironmentVariable: "SECRET_ONE",
				Path:                "/foo/bar",
			},
			"secret_two": {
				EnvironmentVariable: "SECRET_TWO",
				Path:                "/secret/two",
			},
		},
		Parameters: map[string]bundle.ParameterDefinition{
			"param_one": {
				DefaultValue: "one",
			},
			"param_two": {
				DefaultValue: "two",
				Destination: &bundle.Location{
					EnvironmentVariable: "PARAM_TWO",
				},
			},
			"param_three": {
				DefaultValue: "three",
				Destination: &bundle.Location{
					Path: "/param/three",
				},
			},
		},
		Actions: map[string]bundle.Action{
			"test":        {Modifies: true},
			"action_test": {Modifies: true},
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

func testActionWithUndefinedParams(t *testing.T, inst action.Action) {
	out := ioutil.Discard
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

	assert.Error(t, inst.Run(c, mockSet, out))
}

type spyDriver struct {
	RunWasCalledWith *driver.Operation
}

func (f *spyDriver) Run(op *driver.Operation) error {
	f.RunWasCalledWith = op
	return nil
}

func (f *spyDriver) Handles(string) bool {
	return true
}

func testOpFromClaim(t *testing.T, inst action.Action, spyDriver *spyDriver) {
	out := os.Stdout
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

	assert.NoError(t, inst.Run(c, mockSet, out))

	is := assert.New(t)
	op := spyDriver.RunWasCalledWith

	is.Equal(c.Name, op.Installation)
	is.Equal("revision", op.Revision)
	is.Equal(invocImage.Image, op.Image)
	is.Equal(driver.ImageTypeDocker, op.ImageType)
	is.Equal(op.Environment["SECRET_ONE"], "I'm a secret")
	is.Equal(op.Environment["PARAM_TWO"], "twoval")
	is.Equal(op.Environment["CNAB_P_PARAM_ONE"], "oneval")
	is.Equal(op.Files["/secret/two"], "I'm also a secret")
	is.Equal(op.Files["/param/three"], "threeval")
	is.Contains(op.Files, "/cnab/app/image-map.json")
	var imgMap map[string]bundle.Image
	is.NoError(json.Unmarshal([]byte(op.Files["/cnab/app/image-map.json"]), &imgMap))
	is.Equal(c.Bundle.Images, imgMap)
	is.Len(op.Parameters, 3)
	is.Equal(out, op.Out)
}

func testOpFromClaimMissingRequiredParameter(t *testing.T, inst action.Action, actionName string) {
	now := time.Now()
	b := mockBundle()
	b.Parameters["param_one"] = bundle.ParameterDefinition{Required: true}

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

	// missing required parameter fails
	err := inst.Run(c, mockSet, os.Stdout)
	assert.EqualError(t, err, fmt.Sprintf(`missing required parameter "param_one" for action "%s"`, actionName))

	// fill the missing parameter
	c.Parameters["param_one"] = "oneval"
	err = inst.Run(c, mockSet, os.Stdout)
	assert.Nil(t, err)
}

func testOpFromClaimMissingRequiredParamSpecificToAction(t *testing.T, inst action.Action) {
	now := time.Now()
	b := mockBundle()
	// Add a required parameter only defined for the test action
	b.Parameters["param_action_test"] = bundle.ParameterDefinition{
		ApplyTo:  []string{"action_test"},
		Required: true,
	}
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

	// calling install action without the test required parameter for test action is ok
	err := inst.Run(c, mockSet, os.Stdout)
	assert.Nil(t, err)

	// test action needs the required parameter
	act := &action.RunCustom{
		Driver: &spyDriver{},
		Action: "action_test",
	}
	err = act.Run(c, mockSet, os.Stdout)
	assert.EqualError(t, err, `missing required parameter "param_action_test" for action "action_test"`)

	c.Parameters["param_action_test"] = "only for test action"
	err = act.Run(c, mockSet, os.Stdout)
	assert.Nil(t, err)
}

func testSelectInvocationImageEmptyInvocationImages(t *testing.T, inst action.Action) {
	c := &claim.Claim{
		Bundle: &bundle.Bundle{
			Actions: map[string]bundle.Action{
				"test": {Modifies: true},
			},
		},
	}
	err := inst.Run(c, mockSet, os.Stdout)
	assert.NotNil(t, err)

	want := "no invocationImages are defined"
	got := err.Error()
	if !strings.Contains(got, want) {
		t.Fatalf("expected an error containing %q but got %q", want, got)
	}
}

func testSelectInvocationImageDriverIncompatible(t *testing.T, inst action.Action) {
	c := &claim.Claim{
		Bundle: mockBundle(),
	}
	err := inst.Run(c, mockSet, os.Stdout)
	assert.NotNil(t, err)

	want := "driver is not compatible"
	got := err.Error()
	if !strings.Contains(got, want) {
		t.Fatalf("expected an error containing %q but got %q", want, got)
	}
}
