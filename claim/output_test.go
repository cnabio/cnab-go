package claim

import (
	"sort"
	"testing"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutput(t *testing.T) {
	c := exampleClaim
	c.Bundle = bundle.Bundle{
		Definitions: map[string]*definition.Schema{
			"color": {
				Type:    "string",
				Default: "blue",
			},
		},
		Outputs: map[string]bundle.Output{
			"color": {
				Definition: "color",
				ApplyTo:    []string{ActionInstall},
			},
		},
	}
	r, err := c.NewResult(StatusSucceeded)
	require.NoError(t, err)

	o := NewOutput(c, r, "color", nil)

	schema, ok := o.GetSchema()
	require.True(t, ok, "GetSchema failed")
	assert.Equal(t, "string", schema.Type)
	assert.Equal(t, "blue", schema.Default)

	def, ok := o.GetDefinition()
	require.True(t, ok, "GetDefinition failed")
	assert.Equal(t, []string{ActionInstall}, def.ApplyTo)
}

func TestOutputs_GetByName(t *testing.T) {

}

func TestOutputs_GetByIndex(t *testing.T) {

}

func TestOutputs_Sort(t *testing.T) {
	o := NewOutputs([]Output{
		{Name: "a"},
		{Name: "c"},
		{Name: "b"},
	})

	sort.Sort(o)

	wantNames := []string{"a", "b", "c"}
	gotNames := make([]string, 0, 3)
	for i := 0; i < o.Len(); i++ {
		output, ok := o.GetByIndex(i)
		require.True(t, ok, "GetByIndex failed")
		gotNames = append(gotNames, output.Name)
	}

	assert.Equal(t, wantNames, gotNames)
}
