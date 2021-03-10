package claim

import (
	"encoding/json"
	"io/ioutil"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/schema"
)

func TestNew(t *testing.T) {
	claim, err := New("my_claim", ActionInstall, exampleBundle, nil)
	assert.NoError(t, err)

	assert.NotEmpty(t, claim.ID)
	assert.NotEmpty(t, claim.Revision)
	assert.NotEmpty(t, claim.Created)
	assert.Equal(t, "my_claim", claim.Installation, "Installation name is set")
	assert.Equal(t, "install", claim.Action)
	assert.Equal(t, exampleBundle, claim.Bundle)
	assert.Nil(t, claim.Parameters)
}

func TestClaim_Validate(t *testing.T) {
	t.Run("builtin action", func(t *testing.T) {
		c, err := New("test", ActionInstall, exampleBundle, nil)
		require.NoError(t, err, "New failed")
		err = c.Validate()
		require.NoError(t, err, "Validate failed")
	})

	t.Run("custom action", func(t *testing.T) {
		c, err := New("test", "logs", exampleBundle, nil)
		require.NoError(t, err, "New failed")
		err = c.Validate()
		require.NoError(t, err, "Validate failed")
	})

	t.Run("missing id", func(t *testing.T) {
		c, err := New("test", ActionInstall, exampleBundle, nil)
		require.NoError(t, err, "New failed")

		c.ID = ""

		err = c.Validate()
		require.EqualError(t, err, "the claim id must be set")
	})

	t.Run("missing revision", func(t *testing.T) {
		c, err := New("test", ActionInstall, exampleBundle, nil)
		require.NoError(t, err, "New failed")

		c.Revision = ""

		err = c.Validate()
		require.EqualError(t, err, "the revision must be set")
	})

	t.Run("missing installation", func(t *testing.T) {
		c, err := New("test", ActionInstall, exampleBundle, nil)
		require.NoError(t, err, "New failed")

		c.Installation = ""

		err = c.Validate()
		require.EqualError(t, err, "the installation must be set")
	})

	t.Run("missing action", func(t *testing.T) {
		c, err := New("test", "", exampleBundle, nil)
		require.NoError(t, err, "New failed")

		err = c.Validate()
		require.EqualError(t, err, "the action must be set")
	})

	t.Run("invalid action", func(t *testing.T) {
		c, err := New("test", "missing", exampleBundle, nil)
		require.NoError(t, err, "New failed")

		err = c.Validate()
		require.EqualError(t, err, `action "missing" is not defined in the bundle`)
	})
}

func TestClaim_NewClaim(t *testing.T) {
	existingClaim, err := New("claim", ActionUnknown, exampleBundle, nil)
	assert.NoError(t, err)

	t.Run("modifying action", func(t *testing.T) {
		updatedClaim, err := existingClaim.NewClaim("test", exampleBundle, nil)
		require.NoError(t, err, "NewClaim failed")

		is := assert.New(t)
		is.NotEqual(existingClaim.ID, updatedClaim.ID)
		is.NotEqual(existingClaim.Revision, updatedClaim.Revision)
		is.Equal("test", updatedClaim.Action)
	})

	t.Run("non-modifying action", func(t *testing.T) {
		updatedClaim, err := existingClaim.NewClaim("logs", exampleBundle, nil)
		require.NoError(t, err, "NewClaim failed")

		is := assert.New(t)
		is.NotEqual(existingClaim.ID, updatedClaim.ID)
		is.Equal(existingClaim.Revision, updatedClaim.Revision)
		is.Equal("logs", updatedClaim.Action)
	})

}

func TestValidName(t *testing.T) {
	for name, expect := range map[string]bool{
		"M4cb3th":               true,
		"Lady MacBeth":          false, // spaces illegal
		"3_Witches":             true,
		"Banquø":                false, // We could probably loosen this one up
		"King-Duncan":           true,
		"MacDuff@geocities.com": false,
		"hecate":                true, // I wouldn't dare cross Hecate.
		"foo bar baz":           false,
		"foo.bar.baz":           true,
		"foo-bar-baz":           true,
		"foo_bar_baz":           true,
		"":                      false,
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, expect, ValidName.MatchString(name), "expected '%s' to be %t", name, expect)
		})
	}
}

var (
	staticID       = "id"
	staticRevision = "revision"
	staticDate     = time.Date(1983, time.April, 18, 1, 2, 3, 4, time.UTC)
	exampleBundle  = bundle.Bundle{
		SchemaVersion:    "schemaVersion",
		Name:             "mybun",
		Version:          "v0.1.0",
		Description:      "this is my bundle",
		InvocationImages: []bundle.InvocationImage{},
		Actions: map[string]bundle.Action{
			"test": {Modifies: true},
			"logs": {Modifies: false},
		},
	}
)

func TestMarshal_New(t *testing.T) {
	claim, err := New("my_claim", ActionUnknown, bundle.Bundle{}, nil)
	assert.NoError(t, err)

	// override dynamic fields for testing
	claim.ID = staticID
	claim.Revision = staticRevision
	claim.Created = staticDate

	bytes, err := json.Marshal(claim)
	assert.NoError(t, err, "failed to json.Marshal claim")

	wantClaim, err := ioutil.ReadFile("testdata/claim.default.json")
	assert.NoError(t, err, "failed to read test claim")

	gotClaim := strings.TrimSpace(string(bytes))
	assert.Equal(t, string(wantClaim), gotClaim, "marshaled claim does not match expected")
}

var schemaVersion, _ = GetDefaultSchemaVersion()
var exampleClaim = Claim{
	SchemaVersion:   schemaVersion,
	ID:              staticID,
	Installation:    "my_claim",
	Revision:        staticRevision,
	Created:         staticDate,
	Bundle:          exampleBundle,
	BundleReference: "example.com/mybundle@sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
	Action:          ActionInstall,
	Parameters: map[string]interface{}{
		"myparam": "myparamvalue",
	},
	Custom: []interface{}{
		"anything goes",
	},
}

func TestValidateExampleClaim(t *testing.T) {
	claim := exampleClaim

	err := claim.Validate()
	assert.NoError(t, err)

	// change the SchemaVersion to an invalid value
	claim.SchemaVersion = "not-semver"
	err = claim.Validate()
	assert.EqualError(t, err,
		`claim validation failed: invalid schema version "not-semver": Invalid Semantic Version`)
}

func TestMarshal_AllFields(t *testing.T) {
	bytes, err := json.Marshal(exampleClaim)
	assert.NoError(t, err, "failed to json.Marshal claim")

	wantClaim, err := ioutil.ReadFile("testdata/claim.allfields.json")
	assert.NoError(t, err, "failed to read test claim")

	gotClaim := string(bytes)
	assert.Equal(t, strings.TrimSpace(string(wantClaim)), gotClaim, "marshaled claim does not match expected")
}

func TestClaimSchema(t *testing.T) {
	claimBytes, err := json.Marshal(exampleClaim)
	assert.NoError(t, err, "failed to json.Marshal the claim")

	valErrors, err := schema.ValidateClaim(claimBytes)
	assert.NoError(t, err, "failed to validate claim schema")

	if len(valErrors) > 0 {
		t.Log("claim validation against the JSON schema failed:")
		for _, error := range valErrors {
			t.Log(error)
		}
		t.Fail()
	}
}

func TestClaim_GetLastResult(t *testing.T) {
	succeeded := Result{
		ID:     "2",
		Status: StatusSucceeded,
	}
	running := Result{
		ID:     "1",
		Status: StatusRunning,
	}

	t.Run("result exists", func(t *testing.T) {
		c := Claim{
			results: &Results{
				succeeded,
				running,
			},
		}

		r, err := c.GetLastResult()

		require.NoError(t, err, "GetLastResult failed")
		assert.Equal(t, succeeded, r, "GetLastResult did not return the expected result")
		assert.Equal(t, StatusSucceeded, c.GetStatus(), "GetStatus did not return the status of the last result")
	})

	t.Run("no results loaded", func(t *testing.T) {
		c := Claim{
			results: nil,
		}

		r, err := c.GetLastResult()

		require.EqualError(t, err, "the claim does not have results loaded")
		assert.Equal(t, Result{}, r, "should return an empty result when one cannot be found")
		assert.Equal(t, StatusUnknown, c.GetStatus(), "GetStatus should return unknown when there are no results")
	})

	t.Run("no results", func(t *testing.T) {
		c := Claim{
			results: &Results{},
		}

		r, err := c.GetLastResult()

		require.EqualError(t, err, "the claim has no results")
		assert.Equal(t, Result{}, r, "should return an empty result when one cannot be found")
		assert.Equal(t, StatusUnknown, c.GetStatus(), "GetStatus should return unknown when there are no results")
	})

}

func TestClaims_Sort(t *testing.T) {
	c := Claims{
		{ID: "2"},
		{ID: "1"},
		{ID: "3"},
	}

	sort.Sort(c)

	assert.Equal(t, "1", c[0].ID, "Claims did not sort 1 to the first slot")
	assert.Equal(t, "2", c[1].ID, "Claims did not sort 2 to the second slot")
	assert.Equal(t, "3", c[2].ID, "Claims did not sort 3 to the third slot")
}

func TestNewULID_ThreadSafe(t *testing.T) {
	// Validate that the ULID function is thread-safe and generates
	// monotonically increasing values

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var last string
			for j := 0; j < 1000; j++ {
				next, err := NewULID()
				if err != nil {
					t.Fatal(err)
				}

				if strings.Compare(next, last) != 1 {
					t.Fatal("generated a ULID that was not monotonically increasing")
				}
				last = next
			}
		}()
	}
	wg.Wait()
}

func TestMustNewULID_ReturnsError(t *testing.T) {
	originalEntropy := ulidEntropy
	defer func() {
		ulidEntropy = originalEntropy
	}()
	ulidEntropy = strings.NewReader("")

	result, err := NewULID()
	require.EqualError(t, err, "could not generate a new ULID: EOF")
	assert.Equal(t, "", result, "no ULID should be returned when an error occurs")
}

func TestMustNewULID(t *testing.T) {
	result1 := MustNewULID()
	_, err := ulid.Parse(result1)
	require.NoError(t, err, "MustNewULID did not generate a properly encoded ULID")

	result2 := MustNewULID()
	_, err = ulid.Parse(result2)
	require.NoError(t, err, "MustNewULID did not generate a properly encoded ULID")

	assert.Greater(t, result2, result1, "expected increasing ULID values with each call to MustNewULID")
}

func TestMustNewULID_Panics(t *testing.T) {
	originalEntropy := ulidEntropy
	defer func() {
		ulidEntropy = originalEntropy
	}()
	ulidEntropy = strings.NewReader("")

	defer func() {
		recovered := recover()
		err := recovered.(error)
		require.EqualError(t, err, "could not generate a new ULID: EOF")
	}()

	MustNewULID()
	require.Fail(t, "expected MustNewULID to panic")
}

func TestClaim_IsModifyingAction(t *testing.T) {
	testcases := []struct {
		name         string
		action       string
		wantModifies bool
		wantError    string
	}{
		{name: "install", action: ActionInstall, wantModifies: true},
		{name: "upgrade", action: ActionInstall, wantModifies: true},
		{name: "uninstall", action: ActionInstall, wantModifies: true},
		{name: "modifying action", action: "test", wantModifies: true},
		{name: "non-modifying action", action: "logs", wantModifies: false},
		{name: "invalid action", action: "missing", wantError: `custom action not defined "missing"`},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := Claim{Action: tc.action, Bundle: exampleBundle}
			modifies, err := c.IsModifyingAction()

			if tc.wantError != "" {
				require.EqualError(t, err, tc.wantError)
			} else {
				require.NoError(t, err, "IsModifyingAction failed")
				assert.Equal(t, tc.wantModifies, modifies, "invalid modifies")
			}
		})
	}
}

func TestClaim_HasLogs(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		c := Claim{}

		hasLogs, ok := c.HasLogs()
		assert.False(t, ok, "Expected ok to be false")
		assert.False(t, hasLogs, "Expected hasLogs to be false")
	})

	t.Run("true", func(t *testing.T) {
		r := Result{}
		r.OutputMetadata.SetGeneratedByBundle(OutputInvocationImageLogs, false)
		c := Claim{results: &Results{r}}

		hasLogs, ok := c.HasLogs()
		assert.True(t, ok, "Expected ok to be true")
		assert.True(t, hasLogs, "Expected hasLogs to be true")
	})

	t.Run("false", func(t *testing.T) {
		r := Result{}
		c := Claim{results: &Results{r}}

		hasLogs, ok := c.HasLogs()
		assert.True(t, ok, "Expected ok to be true")
		assert.False(t, hasLogs, "Expected hasLogs to be false")
	})
}
