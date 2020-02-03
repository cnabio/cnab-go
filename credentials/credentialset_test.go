package credentials

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/secrets/host"
	"github.com/stretchr/testify/assert"
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

func TestCredentialSet_Expand(t *testing.T) {

	b := &bundle.Bundle{
		Name: "knapsack",
		Credentials: map[string]bundle.Credential{
			"first": {
				Location: bundle.Location{
					EnvironmentVariable: "FIRST_VAR",
				},
			},
			"second": {
				Location: bundle.Location{
					Path: "/second/path",
				},
			},
			"third": {
				Location: bundle.Location{
					EnvironmentVariable: "/THIRD_VAR",
					Path:                "/third/path",
				},
			},
		},
	}
	cs := Set{
		"first":  "first",
		"second": "second",
		"third":  "third",
	}

	env, path, err := cs.Expand(b, false)
	is := assert.New(t)
	is.NoError(err)
	for k, v := range b.Credentials {
		if v.EnvironmentVariable != "" {
			is.Equal(env[v.EnvironmentVariable], cs[k])
		}
		if v.Path != "" {
			is.Equal(path[v.Path], cs[k])
		}
	}
}

func TestCredentialSet_Merge(t *testing.T) {
	cs := Set{
		"first":  "first",
		"second": "second",
		"third":  "third",
	}

	is := assert.New(t)

	err := cs.Merge(Set{})
	is.NoError(err)
	is.Len(cs, 3)
	is.NotContains(cs, "fourth")

	err = cs.Merge(Set{"fourth": "fourth"})
	is.NoError(err)
	is.Len(cs, 4)
	is.Contains(cs, "fourth")

	err = cs.Merge(Set{"second": "bis"})
	is.EqualError(err, `ambiguous credential resolution: "second" is already present in base credential sets, cannot merge`)

}

func TestCredentialSetMissingRequiredCred(t *testing.T) {
	b := &bundle.Bundle{
		Name: "knapsack",
		Credentials: map[string]bundle.Credential{
			"first": {
				Location: bundle.Location{
					EnvironmentVariable: "FIRST_VAR",
				},
				Required: true,
			},
		},
	}
	cs := Set{}
	_, _, err := cs.Expand(b, false)
	assert.EqualError(t, err, `credential "first" is missing from the user-supplied credentials`)
	_, _, err = cs.Expand(b, true)
	assert.NoError(t, err)
}

func TestCredentialSetMissingOptionalCred(t *testing.T) {
	b := &bundle.Bundle{
		Name: "knapsack",
		Credentials: map[string]bundle.Credential{
			"first": {
				Location: bundle.Location{
					EnvironmentVariable: "FIRST_VAR",
				},
			},
		},
	}
	cs := Set{}
	_, _, err := cs.Expand(b, false)
	assert.NoError(t, err)
	_, _, err = cs.Expand(b, true)
	assert.NoError(t, err)
}
