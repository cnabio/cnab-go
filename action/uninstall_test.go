package action

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makes sure Uninstall implements Action interface
var _ Action = &Uninstall{}

func TestUninstall_Run(t *testing.T) {
	out := func(op *driver.Operation) error {
		op.Out = ioutil.Discard
		return nil
	}

	t.Run("happy-path", func(t *testing.T) {
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
	})

	t.Run("configure operation", func(t *testing.T) {
		c := newClaim()
		d := &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"/tmp/some/path": "SOME CONTENT",
				},
			},
			Error: nil,
		}
		inst := &Uninstall{Driver: d}
		addFile := func(op *driver.Operation) error {
			op.Files["/tmp/another/path"] = "ANOTHER FILE"
			return nil
		}
		require.NoError(t, inst.Run(c, mockSet, out, addFile))
		assert.Contains(t, d.Operation.Files, "/tmp/another/path")
	})

	t.Run("error case: configure operation", func(t *testing.T) {
		c := newClaim()
		d := &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"/tmp/some/path": "SOME CONTENT",
				},
			},
			Error: nil,
		}
		inst := &Uninstall{Driver: d}
		sabotage := func(op *driver.Operation) error {
			return errors.New("oops")
		}
		require.EqualError(t, inst.Run(c, mockSet, out, sabotage), "oops")
	})

	t.Run("when there are no outputs in the bundle", func(t *testing.T) {
		c := newClaim()
		c.Bundle.Outputs = nil
		uninst := &Uninstall{
			Driver: &mockDriver{
				shouldHandle: true,
				Result:       driver.OperationResult{},
				Error:        nil,
			},
		}
		err := uninst.Run(c, mockSet, out)
		assert.NoError(t, err)
		assert.NotEqual(t, c.Created, c.Modified, "Claim was not updated with modified time stamp during uninstall after uninstall action")
		assert.Equal(t, claim.ActionUninstall, c.Result.Action, "Claim result action not successfully updated.")
		assert.Equal(t, claim.StatusSuccess, c.Result.Status, "Claim result status not successfully updated.")
		assert.Empty(t, c.Outputs)
	})

	t.Run("error case: driver doesn't handle image", func(t *testing.T) {
		c := newClaim()
		uninst := &Uninstall{Driver: &mockDriver{
			Error:        errors.New("I always fail"),
			shouldHandle: false,
		}}
		err := uninst.Run(c, mockSet, out)
		assert.Error(t, err)
		assert.Empty(t, c.Outputs)
	})

	t.Run("error case: driver does handle image", func(t *testing.T) {
		c := newClaim()
		uninst := &Uninstall{Driver: &mockDriver{
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"/tmp/some/path": "SOME CONTENT",
				},
			},
			Error:        errors.New("I always fail"),
			shouldHandle: true,
		}}
		err := uninst.Run(c, mockSet, out)
		assert.Error(t, err)
		assert.NotEqual(t, "", c.Result.Message, "Expected error message in claim result message")
		assert.Equal(t, claim.ActionUninstall, c.Result.Action)
		assert.Equal(t, claim.StatusFailure, c.Result.Status)
		assert.Equal(t, map[string]interface{}{"some-output": "SOME CONTENT"}, c.Outputs)
	})
}
