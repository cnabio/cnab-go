package action

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makes sure Status implements Action interface
var _ Action = &Status{}

func TestStatus_Run(t *testing.T) {
	out := func(op *driver.Operation) error {
		op.Out = ioutil.Discard
		return nil
	}

	t.Run("happy-path", func(t *testing.T) {
		st := &Status{
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
		c := newClaim()
		err := st.Run(c, mockSet, out)
		assert.NoError(t, err)
		// Status is not a modifying action
		assert.Empty(t, c.Outputs)
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
		inst := &Status{Driver: d}
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
		inst := &Status{Driver: d}
		sabotage := func(op *driver.Operation) error {
			return errors.New("oops")
		}
		require.EqualError(t, inst.Run(c, mockSet, out, sabotage), "oops")
	})

	t.Run("error case: driver doesn't handle image", func(t *testing.T) {
		c := newClaim()
		st := &Status{Driver: &mockDriver{Error: errors.New("I always fail")}}
		err := st.Run(c, mockSet, out)
		assert.Error(t, err)
	})
}
