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

// makes sure Status implements Action interface
var _ action.Action = &action.Status{}

func TestStatus_Run(t *testing.T) {
	out := ioutil.Discard

	st := &action.Status{Driver: &driver.DebugDriver{}}
	c := &claim.Claim{
		Created:    time.Time{},
		Modified:   time.Time{},
		Name:       "name",
		Revision:   "revision",
		Bundle:     mockBundle(),
		Parameters: map[string]interface{}{},
	}

	if err := st.Run(c, mockSet, out); err != nil {
		t.Fatal(err)
	}

	st = &action.Status{Driver: &mockFailingDriver{}}
	assert.Error(t, st.Run(c, mockSet, out))
}

func TestStatus_WithUndefinedParams(t *testing.T) {
	inst := &action.Status{Driver: &mockFailingDriver{}}
	testActionWithUndefinedParams(t, inst)
}

func TestStatusFromClaim(t *testing.T) {
	spyDriver := &spyDriver{}
	rc := &action.Status{Driver: spyDriver}
	testOpFromClaim(t, rc, spyDriver)
}
