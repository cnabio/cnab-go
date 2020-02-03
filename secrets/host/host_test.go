package host

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveHostCredentials(t *testing.T) {
	is := assert.New(t)
	if err := os.Setenv("TEST_USE_VAR", "kakapu"); err != nil {
		t.Fatal("could not setup env")
	}
	defer os.Unsetenv("TEST_USE_VAR")

	h := &SecretStore{}
	for _, tt := range []struct {
		name     string
		keyName  string
		keyValue string
		expect   string
	}{
		{name: "run_program", keyName: SourceCommand, keyValue: "echo wildebeest", expect: "wildebeest"},
		{name: "use_var", keyName: SourceEnv, keyValue: "TEST_USE_VAR", expect: "kakapu"},
		{name: "read_file", keyName: SourcePath, keyValue: "../../credentials/testdata/someconfig.txt", expect: "serval"},
		{name: "plain_value", keyName: SourceValue, keyValue: "cassowary", expect: "cassowary"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dest, err := h.Resolve(tt.keyName, tt.keyValue)
			is.NoError(err)
			is.Equal(tt.expect, strings.TrimSpace(dest))
		})
	}
}

func TestSecretStore_Resolve_InvalidSource(t *testing.T) {
	h := &SecretStore{}
	_, err := h.Resolve("cmd", "kv get something")
	require.Error(t, err, "expected Resolve to return an error")
	assert.Equal(t, "invalid credential source: cmd", err.Error())
}
