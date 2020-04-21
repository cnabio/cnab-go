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

func TestGetSemver(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		expected Version
		err      string
	}{{
		name:     "empty",
		version:  "",
		expected: Version(""),
		err:      `no semver submatch for schemaVersion "" using regex "^cnab-[a-z]+-(.*)"`,
	}, {
		name:     "no match",
		version:  "cnabby-core-1.0.0",
		expected: Version(""),
		err:      `no semver submatch for schemaVersion "cnabby-core-1.0.0" using regex "^cnab-[a-z]+-(.*)"`,
	}, {
		name:     "match but invalid",
		version:  "cnab-core-1.0.0.0",
		expected: Version(""),
		err:      `invalid schema version "1.0.0.0": Invalid Semantic Version`,
	}, {
		name:     "match and valid",
		version:  "cnab-core-1.0.0",
		expected: Version("1.0.0"),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := GetSemver(tc.version)
			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, v)
		})
	}
}
