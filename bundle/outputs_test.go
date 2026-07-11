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
		assert.Contains(t, err.Error(), `encountered an error validating the default value 1 for output "output": type should be string`)
	})

	t.Run("successful validation", func(t *testing.T) {
		o.Definition = "output-definition"
		b.Definitions["output-definition"].Default = "foo"
		err := o.Validate("output", b)
		assert.NoError(t, err)
	})
}

func TestOutputValidatePath(t *testing.T) {
	testCases := []struct {
		name string
		path string
		err  string
	}{
		{
			name: "empty path",
			path: "",
		},
		{
			name: "valid path",
			path: "/cnab/app/outputs/foo",
		},
		{
			name: "relative path traversal",
			path: "../../../etc/passwd",
			err:  `path "../../../etc/passwd" must be a clean path under "/cnab/app/outputs/"`,
		},
		{
			name: "absolute path traversal",
			path: "/cnab/app/outputs/../../../etc/shadow",
			err:  `path "/cnab/app/outputs/../../../etc/shadow" must be a clean path under "/cnab/app/outputs/"`,
		},
		{
			name: "not under outputs dir",
			path: "/tmp/some/path",
			err:  `path "/tmp/some/path" must be a clean path under "/cnab/app/outputs/"`,
		},
		{
			name: "outputs dir itself, no suffix",
			path: "/cnab/app/outputs",
			err:  `path "/cnab/app/outputs" must be a clean path under "/cnab/app/outputs/"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := Output{Path: tc.path}
			err := o.ValidatePath()
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
		})
	}
}
