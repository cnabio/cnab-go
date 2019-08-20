package action

import (
	"errors"
	"testing"

	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurableAction_ApplyConfig(t *testing.T) {
	t.Run("no config defined", func(t *testing.T) {
		a := ConfigurableAction{}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		assert.NoError(t, err, "ApplyConfig should not have returned an error")
	})

	t.Run("config is persisted", func(t *testing.T) {
		a := ConfigurableAction{
			OperationConfig: func(op *driver.Operation) error {
				if op.Files == nil {
					op.Files = make(map[string]string, 1)
				}
				op.Files["a"] = "b"
				return nil
			},
		}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		assert.NoError(t, err, "ApplyConfig should not have returned an error")
		assert.Contains(t, op.Files, "a", "Changes from the config function were not persisted")
	})

	t.Run("error is returned", func(t *testing.T) {
		a := ConfigurableAction{
			OperationConfig: func(op *driver.Operation) error {
				return errors.New("oops")
			},
		}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		require.EqualError(t, err, "oops")
	})
}
