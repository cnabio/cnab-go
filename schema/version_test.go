package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		version Version
		err     string
	}{{
		name:    "empty",
		version: Version(""),
		err:     `invalid schema version "": Invalid Semantic Version`,
	}, {
		name:    "invalid",
		version: Version("not-semver"),
		err:     `invalid schema version "not-semver": Invalid Semantic Version`,
	}, {
		name:    "valid",
		version: Version("v1.0.0"),
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
