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

// makes sure Install implements Action interface
var _ action.Action = &action.Install{}

func TestInstall_Run(t *testing.T) {
	out := ioutil.Discard

	c := &claim.Claim{
		Created:    time.Time{},
		Modified:   time.Time{},
		Name:       "name",
		Revision:   "revision",
		Bundle:     mockBundle(),
		Parameters: map[string]interface{}{},
	}

	inst := &action.Install{Driver: &driver.DebugDriver{}}
	assert.NoError(t, inst.Run(c, mockSet, out))

	inst = &action.Install{Driver: &mockFailingDriver{}}
	assert.Error(t, inst.Run(c, mockSet, out))

	inst = &action.Install{Driver: &mockFailingDriver{shouldHandle: true}}
	assert.Error(t, inst.Run(c, mockSet, out))
}

func TestInstall_WithUndefinedParams(t *testing.T) {
	inst := &action.Install{Driver: &mockFailingDriver{}}
	testActionWithUndefinedParams(t, inst)
}

func TestInstallFromClaim(t *testing.T) {
	spyDriver := &spyDriver{}
	inst := &action.Install{Driver: spyDriver}
	testOpFromClaim(t, inst, spyDriver)
}

func TestInstall_FromClaimMissingRequiredParameter(t *testing.T) {
	inst := &action.Install{Driver: &spyDriver{}}
	testOpFromClaimMissingRequiredParameter(t, inst, "install")
}

func TestInstall_FromClaimMissingRequiredParamSpecificToAction(t *testing.T) {
	inst := &action.Install{Driver: &spyDriver{}}
	testOpFromClaimMissingRequiredParamSpecificToAction(t, inst)
}

func TestInstall_SelectInvocationImageEmptyInvocationImages(t *testing.T) {
	inst := &action.Install{Driver: &spyDriver{}}
	testSelectInvocationImageEmptyInvocationImages(t, inst)
}

func TestInstall_SelectInvocationImageDriverIncompatible(t *testing.T) {
	inst := &action.Install{Driver: &mockFailingDriver{}}
	testSelectInvocationImageDriverIncompatible(t, inst)
}
