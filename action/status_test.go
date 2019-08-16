package action

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
)

// makes sure Status implements Action interface
var _ Action = &Status{}

func TestStatus_Run(t *testing.T) {
	out := ioutil.Discard

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

	t.Run("error case: driver doesn't handle image", func(t *testing.T) {
		c := newClaim()
		st := &Status{Driver: &mockDriver{Error: errors.New("I always fail")}}
		err := st.Run(c, mockSet, out)
		assert.Error(t, err)
	})
}
