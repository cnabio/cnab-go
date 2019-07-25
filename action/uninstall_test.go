package action

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"

	"github.com/stretchr/testify/assert"
)

// makes sure Uninstall implements Action interface
var _ Action = &Uninstall{}

func TestUninstall_Run(t *testing.T) {
	out := ioutil.Discard

	// happy path
	c := newClaim()
	uninst := &Uninstall{
		Driver: &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"/tmp/some/path": "SOME CONTENT",
				},
			},
			Error: nil,
		},
	}
	err := uninst.Run(c, mockSet, out)
	assert.NoError(t, err)
	assert.NotEqual(t, c.Created, c.Modified, "Claim was not updated with modified time stamp during uninstall after uninstall action")
	assert.Equal(t, claim.ActionUninstall, c.Result.Action, "Claim result action not successfully updated.")
	assert.Equal(t, claim.StatusSuccess, c.Result.Status, "Claim result status not successfully updated.")
	assert.Equal(t, map[string]interface{}{"some-output": "SOME CONTENT"}, c.Outputs)

	// when there are no outputs in the bundle
	c = newClaim()
	c.Bundle.Outputs = nil
	uninst = &Uninstall{
		Driver: &mockDriver{
			shouldHandle: true,
			Result:       driver.OperationResult{},
			Error:        nil,
		},
	}
	err = uninst.Run(c, mockSet, out)
	assert.NoError(t, err)
	assert.NotEqual(t, c.Created, c.Modified, "Claim was not updated with modified time stamp during uninstall after uninstall action")
	assert.Equal(t, claim.ActionUninstall, c.Result.Action, "Claim result action not successfully updated.")
	assert.Equal(t, claim.StatusSuccess, c.Result.Status, "Claim result status not successfully updated.")
	assert.Empty(t, c.Outputs)

	// error case: driver doesn't handle image
	c = newClaim()
	uninst = &Uninstall{Driver: &mockDriver{
		Error:        errors.New("I always fail"),
		shouldHandle: false,
	}}
	err = uninst.Run(c, mockSet, out)
	assert.Error(t, err)
	assert.Empty(t, c.Outputs)

	// error case: driver does handle image
	c = newClaim()
	uninst = &Uninstall{Driver: &mockDriver{
		Result: driver.OperationResult{
			Outputs: map[string]string{
				"/tmp/some/path": "SOME CONTENT",
			},
		},
		Error:        errors.New("I always fail"),
		shouldHandle: true,
	}}
	err = uninst.Run(c, mockSet, out)
	assert.Error(t, err)
	assert.NotEqual(t, "", c.Result.Message, "Expected error message in claim result message")
	assert.Equal(t, claim.ActionUninstall, c.Result.Action)
	assert.Equal(t, claim.StatusFailure, c.Result.Status)
	assert.Equal(t, map[string]interface{}{"some-output": "SOME CONTENT"}, c.Outputs)
}
