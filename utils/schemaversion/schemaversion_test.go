package schemaversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		version SchemaVersion
		err     string
	}{{
		name:    "empty",
		version: SchemaVersion(""),
		err:     `invalid schema version "": Invalid Semantic Version`,
	}, {
		name:    "invalid",
		version: SchemaVersion("not-semver"),
		err:     `invalid schema version "not-semver": Invalid Semantic Version`,
	}, {
		name:    "valid",
		version: SchemaVersion("v1.0.0"),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.version.Validate()
			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
