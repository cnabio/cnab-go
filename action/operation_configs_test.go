package action

import (
	"errors"
	"os"
	"testing"

	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationConfigs_ApplyConfig(t *testing.T) {
	t.Run("no config defined", func(t *testing.T) {
		a := OperationConfigs{}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		assert.NoError(t, err, "ApplyConfig should not have returned an error")
	})

	t.Run("all config is persisted", func(t *testing.T) {
		a := OperationConfigs{
			func(op *driver.Operation) error {
				if op.Files == nil {
					op.Files = make(map[string]string, 1)
				}
				op.Files["a"] = "b"
				return nil
			},
			func(op *driver.Operation) error {
				op.Out = os.Stdout
				return nil
			},
		}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		require.NoError(t, err, "ApplyConfig should not have returned an error")
		assert.Contains(t, op.Files, "a", "Changes from the first config function were not persisted")
		assert.Equal(t, os.Stdout, op.Out, "Changes from the second config function were not persisted")
	})

	t.Run("error is returned immediately", func(t *testing.T) {
		a := OperationConfigs{
			func(op *driver.Operation) error {
				return errors.New("oops")
			},
			func(op *driver.Operation) error {
				op.Out = os.Stdout
				return nil
			},
		}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		require.EqualError(t, err, "oops")
		require.Nil(t, op.Out, "Changes from the second config function should not have been applied")
	})
}
