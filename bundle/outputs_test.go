package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cnabio/cnab-go/bundle/definition"
)

func TestOutputValidate(t *testing.T) {
	b := Bundle{
		Definitions: map[string]*definition.Schema{
			"output-definition": {
				Type: "string",
			},
		},
	}
	o := Output{}

	t.Run("empty output fails", func(t *testing.T) {
		err := o.Validate("output", b)
		assert.EqualError(t, err, "output definition must be provided")
	})

	t.Run("unsuccessful validation", func(t *testing.T) {
		o.Definition = "output-definition"
		b.Definitions["output-definition"].Default = 1
		err := o.Validate("output", b)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `encountered an error validating the default value 1 for output "output": at '': got number, want string`)
	})

	t.Run("successful validation", func(t *testing.T) {
		o.Definition = "output-definition"
		b.Definitions["output-definition"].Default = "foo"
		err := o.Validate("output", b)
		assert.NoError(t, err)
	})
}
