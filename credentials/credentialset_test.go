package credentials

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/secrets/host"
)

func TestCredentialSet_ResolveCredentials(t *testing.T) {
	is := assert.New(t)
	if err := os.Setenv("TEST_USE_VAR", "kakapu"); err != nil {
		t.Fatal("could not setup env")
	}
	defer os.Unsetenv("TEST_USE_VAR")

	goos := "unix"
	if runtime.GOOS == "windows" {
		goos = runtime.GOOS
	}
	credset, err := Load(fmt.Sprintf("testdata/staging-%s.yaml", goos))
	is.NoError(err)

	version, err := GetDefaultSchemaVersion()
	require.NoError(t, err, "GetDefaultSchemaVersion failed")
	is.Equal(version, credset.SchemaVersion)

	h := &host.SecretStore{}
	results, err := credset.ResolveCredentials(h)
	if err != nil {
		t.Fatal(err)
	}
	count := 4
	is.Len(results, count, "Expected %d credentials", count)

	for _, tt := range []struct {
		name   string
		key    string
		expect string
		path   string
	}{
		{name: "run_program", key: "TEST_RUN_PROGRAM", expect: "wildebeest"},
		{name: "use_var", key: "TEST_USE_VAR", expect: "kakapu"},
		{name: "read_file", key: "TEST_READ_FILE", expect: "serval"},
		{name: "plain_value", key: "TEST_PLAIN_VALUE", expect: "cassowary"},
	} {
		dest, ok := results[tt.name]
		is.True(ok)
		is.Equal(tt.expect, strings.TrimSpace(dest))
	}
}
