package action

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makes sure RunCustom implements Action interface
var _ Action = &RunCustom{}

func TestRunCustom(t *testing.T) {
	out := func(op *driver.Operation) error {
		op.Out = ioutil.Discard
		return nil
	}

	rc := &RunCustom{
		Driver: &mockDriver{
			shouldHandle: true,
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"/tmp/some/path": "SOME CONTENT",
				},
			},
			Error: nil,
		},
		Action: "test",
	}

	t.Run("happy-path", func(t *testing.T) {
		c := newClaim()
		err := rc.Run(c, mockSet, out)
		assert.NoError(t, err)
		assert.Equal(t, claim.StatusSuccess, c.Result.Status)
		assert.Equal(t, "test", c.Result.Action)
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
		inst := &RunCustom{Driver: d, Action: "test"}
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
		inst := &RunCustom{Driver: d, Action: "test"}
		sabotage := func(op *driver.Operation) error {
			return errors.New("oops")
		}
		require.EqualError(t, inst.Run(c, mockSet, out, sabotage), "oops")
	})

	t.Run("when there are no outputs in the bundle", func(t *testing.T) {
		c := newClaim()
		c.Bundle.Outputs = nil
		rc.Driver = &mockDriver{
			shouldHandle: true,
			Result:       driver.OperationResult{},
			Error:        nil,
		}
		err := rc.Run(c, mockSet, out)
		assert.NoError(t, err)
		assert.NotEqual(t, c.Created, c.Modified, "Claim was not updated with modified timestamp after custom action")
		assert.Equal(t, claim.StatusSuccess, c.Result.Status)
		assert.Equal(t, "test", c.Result.Action)
		assert.Empty(t, c.Outputs)
	})

	t.Run("error case: driver doesn't handle image", func(t *testing.T) {
		c := newClaim()
		rc.Driver = &mockDriver{
			Error:        errors.New("I always fail"),
			shouldHandle: false,
		}
		err := rc.Run(c, mockSet, out)
		assert.Error(t, err)
		assert.Empty(t, c.Outputs)
	})

	t.Run("error case: driver returns error", func(t *testing.T) {
		c := newClaim()
		rc.Driver = &mockDriver{
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"/tmp/some/path": "SOME CONTENT",
				},
			},
			Error:        errors.New("I always fail"),
			shouldHandle: true,
		}
		err := rc.Run(c, mockSet, out)
		assert.Error(t, err)
		assert.NotEqual(t, "", c.Result.Message, "Expected error message in claim result message")
		assert.Equal(t, "test", c.Result.Action)
		assert.Equal(t, claim.StatusFailure, c.Result.Status)
		assert.Equal(t, map[string]interface{}{"some-output": "SOME CONTENT"}, c.Outputs)
	})

	t.Run("error case: driver returns an error but the action does not modify", func(t *testing.T) {
		c := newClaim()
		action := c.Bundle.Actions["test"]
		action.Modifies = false
		c.Bundle.Actions["test"] = action

		rc.Driver = &mockDriver{
			Result: driver.OperationResult{
				Outputs: map[string]string{
					"/tmp/some/path": "SOME CONTENT",
				},
			},
			Error:        errors.New("I always fail"),
			shouldHandle: true,
		}
		err := rc.Run(c, mockSet, out)
		assert.Error(t, err)
		assert.Empty(t, c.Result, "Expected claim results not to be tracked when the action does not modify")
		assert.Empty(t, c.Outputs, "Expected output results not to be tracked with the action does not modify")
	})

	t.Run("error case: forbidden custom actions should fail", func(t *testing.T) {
		c := newClaim()
		rc.Action = "install"
		err := rc.Run(c, mockSet, out)
		assert.Error(t, err)
		assert.Empty(t, c.Outputs)
	})

	t.Run("error case: unknown actions should fail", func(t *testing.T) {
		c := newClaim()
		rc.Action = "test"
		c.Bundle.Actions = map[string]bundle.Action{}
		err := rc.Run(c, mockSet, out)
		assert.Error(t, err, "Unknown action should fail")
		assert.Empty(t, c.Outputs)
	})
}
