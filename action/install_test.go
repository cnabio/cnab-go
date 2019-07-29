package action

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"

	"github.com/stretchr/testify/assert"
)

// makes sure Install implements Action interface
var _ Action = &Install{}

func TestInstall_Run(t *testing.T) {
	out := ioutil.Discard

	t.Run("happy-path", func(t *testing.T) {
		c := newClaim()
		inst := &Install{Driver: &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"/tmp/some/path": "SOME CONTENT",
				},
			},
			Error: nil,
		}}
		assert.NoError(t, inst.Run(c, mockSet, out))
		assert.Equal(t, claim.StatusSuccess, c.Result.Status)
		assert.Equal(t, claim.ActionInstall, c.Result.Action)
		assert.Equal(t, map[string]interface{}{"some-output": "SOME CONTENT"}, c.Outputs)
	})

	t.Run("when the bundle has no outputs", func(t *testing.T) {
		c := newClaim()
		c.Bundle.Outputs = nil
		inst := &Install{
			Driver: &mockDriver{
				shouldHandle: true,
				Result:       driver.OperationResult{},
				Error:        nil,
			},
		}
		assert.NoError(t, inst.Run(c, mockSet, out))
		assert.Equal(t, claim.StatusSuccess, c.Result.Status)
		assert.Equal(t, claim.ActionInstall, c.Result.Action)
		assert.Empty(t, c.Outputs)
	})

	t.Run("error case: driver can't handle image", func(t *testing.T) {
		c := newClaim()
		inst := &Install{
			Driver: &mockDriver{
				shouldHandle: false,
				Error:        errors.New("I always fail"),
			},
		}
		assert.Error(t, inst.Run(c, mockSet, out))
	})

	t.Run("error case: driver returns error", func(t *testing.T) {
		c := newClaim()
		inst := &Install{
			Driver: &mockDriver{
				shouldHandle: true,
				Result: driver.OperationResult{
					Outputs: map[string]string{
						"/tmp/some/path": "SOME CONTENT",
					},
				},
				Error: errors.New("I always fail"),
			},
		}
		assert.Error(t, inst.Run(c, mockSet, out))
		assert.Equal(t, claim.StatusFailure, c.Result.Status)
		assert.Equal(t, claim.ActionInstall, c.Result.Action)
		assert.Equal(t, map[string]interface{}{"some-output": "SOME CONTENT"}, c.Outputs)
	})
}
