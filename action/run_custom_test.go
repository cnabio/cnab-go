package action_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"

	"github.com/stretchr/testify/assert"
)

// makes sure RunCustom implements Action interface
var _ action.Action = &action.RunCustom{}

func TestRunCustom(t *testing.T) {
	out := ioutil.Discard
	is := assert.New(t)

	rc := &action.RunCustom{
		Driver: &driver.DebugDriver{},
		Action: "test",
	}
	c := &claim.Claim{
		Created:    time.Time{},
		Modified:   time.Time{},
		Name:       "runcustom",
		Revision:   "revision",
		Bundle:     mockBundle(),
		Parameters: map[string]interface{}{},
	}

	if err := rc.Run(c, mockSet, out); err != nil {
		t.Fatal(err)
	}
	is.Equal(claim.StatusSuccess, c.Result.Status)
	is.Equal("test", c.Result.Action)

	// Make sure we don't allow forbidden custom actions
	rc.Action = "install"
	is.Error(rc.Run(c, mockSet, out))

	// Get rid of custom actions, and this should fail
	rc.Action = "test"
	c.Bundle.Actions = map[string]bundle.Action{}
	if err := rc.Run(c, mockSet, out); err == nil {
		t.Fatal("Unknown action should fail")
	}
}

func TestRunCustom_WithUndefinedParams(t *testing.T) {
	rc := &action.RunCustom{
		Driver: &driver.DebugDriver{},
		Action: "test",
	}
	testActionWithUndefinedParams(t, rc)
}

func TestRunCustom_FromClaim(t *testing.T) {
	spyDriver := &spyDriver{}
	rc := &action.RunCustom{
		Driver: spyDriver,
		Action: "test",
	}
	testOpFromClaim(t, rc, spyDriver)
}

func TestRunCustom_FromClaimMissingRequiredParameter(t *testing.T) {
	rc := &action.RunCustom{
		Driver: &spyDriver{},
		Action: "test",
	}
	testOpFromClaimMissingRequiredParameter(t, rc, "test")
}

func TestRunCustom_FromClaimMissingRequiredParamSpecificToAction(t *testing.T) {
	rc := &action.RunCustom{
		Driver: &spyDriver{},
		Action: "test",
	}
	testOpFromClaimMissingRequiredParamSpecificToAction(t, rc)
}

func TestRunCustom_SelectInvocationImageEmptyInvocationImages(t *testing.T) {
	rc := &action.RunCustom{
		Driver: &spyDriver{},
		Action: "test",
	}
	testSelectInvocationImageEmptyInvocationImages(t, rc)
}

func TestRunCustom_SelectInvocationImageDriverIncompatible(t *testing.T) {
	rc := &action.RunCustom{
		Driver: &mockFailingDriver{},
		Action: "test",
	}
	testSelectInvocationImageDriverIncompatible(t, rc)
}
