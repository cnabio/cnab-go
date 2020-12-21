package action

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/driver"
)

func TestOperationConfigs_ApplyConfig(t *testing.T) {
	t.Run("no config defined", func(t *testing.T) {
		a := OperationConfigs{}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		assert.NoError(t, err, "ApplyConfig should not have returned an error")
		assert.Equal(t, os.Stdout, op.Out, "write to stdout when output is undefined")
		assert.Equal(t, os.Stderr, op.Err, "write to stderr when output is undefined")
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
				op.Out = ioutil.Discard
				return nil
			},
		}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		require.NoError(t, err, "ApplyConfig should not have returned an error")
		assert.Contains(t, op.Files, "a", "Changes from the first config function were not persisted")
		assert.Equal(t, ioutil.Discard, op.Out, "Changes from the second config function were not persisted")
		assert.Equal(t, os.Stderr, op.Err, "Changes from the second config function were not persisted")
	})

	t.Run("error is returned immediately", func(t *testing.T) {
		a := OperationConfigs{
			func(op *driver.Operation) error {
				return errors.New("oops")
			},
			func(op *driver.Operation) error {
				op.Out = ioutil.Discard
				return nil
			},
		}
		op := &driver.Operation{}
		err := a.ApplyConfig(op)
		require.EqualError(t, err, "oops")
		require.Nil(t, op.Out, "Changes from the second config function should not have been applied")
	})
}
