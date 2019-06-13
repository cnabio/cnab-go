package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteCredDefinition(t *testing.T) {
	payload := `{
		"credentials": {
			"something": { 
				"description" : "wicked this way comes",
				"path" : "/cnab/app/a/credential",
				"required" : true
			}
		}
	}`

	definitions, err := Unmarshal([]byte(payload))
	require.NoError(t, err, "given credentials payload was valid json")

	something, ok := definitions.Credentials["something"]
	assert.True(t, ok, "should have found the `something` entry")

	assert.Equal(t, "/cnab/app/a/credential", something.Path, "did not contain the expected path")
	assert.Equal(t, "wicked this way comes", something.Description, "did not contain the expected description")

}

func TestHandleMultipleCreds(t *testing.T) {
	payload := `{
		"credentials": {
			"something": { },
			"else": { }
		}
	}`

	definitions, err := Unmarshal([]byte(payload))
	require.NoError(t, err, "given credentials payload was valid json")

	assert.Equal(t, 2, len(definitions.Credentials), "credentials should have contained two entries")

	_, ok := definitions.Credentials["something"]
	assert.True(t, ok, "should have found the `something` entry")

	_, ok = definitions.Credentials["else"]
	assert.True(t, ok, "should have found the `else` entry")

}

func TestNotRequiredIsFalse(t *testing.T) {
	payload := `{
		"credentials": {
			"something": { 
				"path" : "/cnab/app/a/credential",
				"required" : false
			}
		}
	}`

	definitions, err := Unmarshal([]byte(payload))
	require.NoError(t, err, "given credentials payload was valid json")

	something, ok := definitions.Credentials["something"]
	assert.True(t, ok, "should have found the credential")
	assert.False(t, something.Required, "required was set to `false` in the json")
}

func TestUnspecifiedRequiredIsFalse(t *testing.T) {
	payload := `{
		"credentials": {
			"something": { 
				"path" : "/cnab/app/a/credential"
			}
		}
	}`

	definitions, err := Unmarshal([]byte(payload))
	require.NoError(t, err, "given credentials payload was valid json")

	something, ok := definitions.Credentials["something"]
	assert.True(t, ok, "should have found the credential")
	assert.False(t, something.Required, "required was unspecified in the json")
}

func TestRequiredIsTrue(t *testing.T) {
	payload := `{
		"credentials": {
			"something": { 
				"path" : "/cnab/app/a/credential",
				"required" : true
			}
		}
	}`

	definitions, err := Unmarshal([]byte(payload))
	require.NoError(t, err, "given credentials payload was valid json")

	something, ok := definitions.Credentials["something"]
	assert.True(t, ok, "should have found the credential")
	assert.True(t, something.Required, "required was set to `true` in the json")
}
