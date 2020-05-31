package valuesource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cnabio/cnab-go/bundle"
)

func TestSet_ExpandCredentials(t *testing.T) {

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
	set := Set{
		"first":  "first",
		"second": "second",
		"third":  "third",
	}

	env, path, err := set.ExpandCredentials(b, false)
	is := assert.New(t)
	is.NoError(err)
	for k, v := range b.Credentials {
		if v.EnvironmentVariable != "" {
			is.Equal(env[v.EnvironmentVariable], set[k])
		}
		if v.Path != "" {
			is.Equal(path[v.Path], set[k])
		}
	}
}

func TestSet_Merge(t *testing.T) {
	set := Set{
		"first":  "first",
		"second": "second",
		"third":  "third",
	}

	is := assert.New(t)

	err := set.Merge(Set{})
	is.NoError(err)
	is.Len(set, 3)
	is.NotContains(set, "fourth")

	err = set.Merge(Set{"fourth": "fourth"})
	is.NoError(err)
	is.Len(set, 4)
	is.Contains(set, "fourth")

	err = set.Merge(Set{"second": "bis"})
	is.EqualError(err, `ambiguous value resolution: "second" is already present in base sets, cannot merge`)

}

func TestSetMissingRequiredCred(t *testing.T) {
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
	set := Set{}
	_, _, err := set.ExpandCredentials(b, false)
	assert.EqualError(t, err, `credential "first" is missing from the user-supplied credentials`)
	_, _, err = set.ExpandCredentials(b, true)
	assert.NoError(t, err)
}

func TestSetMissingOptionalCred(t *testing.T) {
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
	set := Set{}
	_, _, err := set.ExpandCredentials(b, false)
	assert.NoError(t, err)
	_, _, err = set.ExpandCredentials(b, true)
	assert.NoError(t, err)
}
