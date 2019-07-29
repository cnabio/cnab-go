package action

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/stretchr/testify/assert"
)

// makes sure RunCustom implements Action interface
var _ Action = &RunCustom{}

func TestRunCustom(t *testing.T) {
	out := ioutil.Discard
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
