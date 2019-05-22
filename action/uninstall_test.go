package action_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"

	"github.com/stretchr/testify/assert"
)

// makes sure Uninstall implements Action interface
var _ action.Action = &action.Uninstall{}

func TestUninstall_Run(t *testing.T) {
	out := ioutil.Discard

	c := &claim.Claim{
		Created:    time.Time{},
		Modified:   time.Time{},
		Name:       "name",
		Revision:   "revision",
		Bundle:     mockBundle(),
		Parameters: map[string]interface{}{},
	}

	uninst := &action.Uninstall{Driver: &driver.DebugDriver{}}
	assert.NoError(t, uninst.Run(c, mockSet, out))
	if c.Created == c.Modified {
		t.Error("Claim was not updated with modified time stamp during uninstallafter uninstall action")
	}

	if c.Result.Action != claim.ActionUninstall {
		t.Errorf("Claim result action not successfully updated. Expected %v, got %v", claim.ActionUninstall, c.Result.Action)
	}
	if c.Result.Status != claim.StatusSuccess {
		t.Errorf("Claim result status not successfully updated. Expected %v, got %v", claim.StatusSuccess, c.Result.Status)
	}

	uninst = &action.Uninstall{Driver: &mockFailingDriver{}}
	assert.Error(t, uninst.Run(c, mockSet, out))

	uninst = &action.Uninstall{Driver: &mockFailingDriver{shouldHandle: true}}
	assert.Error(t, uninst.Run(c, mockSet, out))
	if c.Result.Message == "" {
		t.Error("Expected error message in claim result message")
	}

	if c.Result.Action != claim.ActionUninstall {
		t.Errorf("Expected claim result action to be %v, got %v", claim.ActionUninstall, c.Result.Action)
	}

	if c.Result.Status != claim.StatusFailure {
		t.Errorf("Expected claim result status to be %v, got %v", claim.StatusFailure, c.Result.Status)
	}
}

func TestUninstall_WithUndefinedParams(t *testing.T) {
	inst := &action.Uninstall{Driver: &mockFailingDriver{}}
	testActionWithUndefinedParams(t, inst)
}

func TestUninstallFromClaim(t *testing.T) {
	spyDriver := &spyDriver{}
	rc := &action.Uninstall{Driver: spyDriver}
	testOpFromClaim(t, rc, spyDriver)
}

func TestUninstall_FromClaimMissingRequiredParameter(t *testing.T) {
	inst := &action.Uninstall{Driver: &spyDriver{}}
	testOpFromClaimMissingRequiredParameter(t, inst, "uninstall")
}

func TestUninstall_FromClaimMissingRequiredParamSpecificToAction(t *testing.T) {
	inst := &action.Uninstall{Driver: &spyDriver{}}
	testOpFromClaimMissingRequiredParamSpecificToAction(t, inst)
}

func TestUnintall_SelectInvocationImageEmptyInvocationImages(t *testing.T) {
	inst := &action.Uninstall{Driver: &spyDriver{}}
	testSelectInvocationImageEmptyInvocationImages(t, inst)
}

func TestUninstall_SelectInvocationImageDriverIncompatible(t *testing.T) {
	inst := &action.Uninstall{Driver: &mockFailingDriver{}}
	testSelectInvocationImageDriverIncompatible(t, inst)
}
