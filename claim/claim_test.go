package claim

import (
	"encoding/json"
	"io/ioutil"
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
	// Make sure that the default Result has status and action set.
	claim, err := New("my_claim")
	assert.NoError(t, err)

	err = claim.Validate()
	assert.NoError(t, err)

	assert.Equal(t, "my_claim", claim.Installation, "Installation name is set")
	assert.Equal(t, "unknown", claim.Result.Status)
	assert.Equal(t, "unknown", claim.Result.Action)

	assert.Equal(t, map[string]interface{}{}, claim.Outputs)
	assert.Equal(t, map[string]interface{}{}, claim.Parameters)
}

func TestUpdate(t *testing.T) {
	claim, err := New("claim")
	assert.NoError(t, err)
	oldMod := claim.Modified
	oldUlid := claim.Revision

	time.Sleep(1 * time.Millisecond) // Force the Update to happen at a new time. For those of us who remembered to press the Turbo button.

	claim.Update(ActionInstall, StatusSucceeded)

	is := assert.New(t)
	is.NotEqual(oldMod, claim.Modified)
	is.NotEqual(oldUlid, claim.Revision)
	is.Equal("install", claim.Result.Action)
	is.Equal("succeeded", claim.Result.Status)
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
	staticRevision = "revision"
	staticDate     = time.Date(1983, time.April, 18, 1, 2, 3, 4, time.UTC)
	exampleBundle  = bundle.Bundle{
		SchemaVersion:    "schemaVersion",
		Name:             "mybun",
		Version:          "v0.1.0",
		Description:      "this is my bundle",
		InvocationImages: []bundle.InvocationImage{},
	}
)

func TestMarshal_New(t *testing.T) {
	claim, err := New("my_claim")
	assert.NoError(t, err)

	// override dynamic fields for testing
	claim.Revision = staticRevision
	claim.Created = staticDate
	claim.Modified = staticDate

	bytes, err := json.Marshal(claim)
	assert.NoError(t, err, "failed to json.Marshal claim")

	wantClaim, err := ioutil.ReadFile("testdata/claim.default.json")
	assert.NoError(t, err, "failed to read test claim")

	assert.Equal(t, string(wantClaim), strings.TrimSpace(string(bytes)), "marshaled claim does not match expected")
}

var schemaVersion, _ = GetDefaultSchemaVersion()
var exampleClaim = Claim{
	SchemaVersion:   schemaVersion,
	Installation:    "my_claim",
	Revision:        staticRevision,
	Created:         staticDate,
	Modified:        staticDate,
	Bundle:          &exampleBundle,
	BundleReference: "example.com/mybundle@sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
	Result: Result{
		Action:  ActionInstall,
		Message: "result message",
		Status:  StatusPending,
	},
	Parameters: map[string]interface{}{
		"myparam": "myparamvalue",
	},
	Outputs: map[string]interface{}{
		"myoutput": "myoutputvalue",
	},
	Custom: []string{
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

func TestResult_Validate_ValidStatus(t *testing.T) {
	validStatuses := []string{
		StatusCanceled,
		StatusRunning,
		StatusFailed,
		StatusPending,
		StatusSucceeded,
		StatusUnknown,
	}
	for _, status := range validStatuses {
		t.Run(status+" status", func(t *testing.T) {
			result := Result{
				Action: ActionInstall,
				Status: status,
			}
			err := result.Validate()
			assert.NoError(t, err, "%s is a valid claim status", status)
		})
	}
}

func TestValidate_InvalidResult(t *testing.T) {
	claim := exampleClaim

	t.Run("if result is empty, validation should fail", func(t *testing.T) {
		claim.Result = Result{}
		err := claim.Validate()
		assert.EqualError(t, err, "claim validation failed: the action must be provided")
	})

	t.Run("if result has empty action, validation should fail", func(t *testing.T) {
		claim.Result = Result{
			Status: StatusSucceeded,
		}
		err := claim.Validate()
		assert.EqualError(t, err, "claim validation failed: the action must be provided")
	})

	t.Run("if result has invalid status, validation should fail", func(t *testing.T) {
		claim.Result = Result{
			Action: "install",
			Status: "invalidStatus",
		}
		err := claim.Validate()
		assert.EqualError(t, err, "claim validation failed: invalid status: invalidStatus")
	})
}

func TestMarshal_AllFields(t *testing.T) {
	bytes, err := json.Marshal(exampleClaim)
	assert.NoError(t, err, "failed to json.Marshal claim")

	wantClaim, err := ioutil.ReadFile("testdata/claim.allfields.json")
	assert.NoError(t, err, "failed to read test claim")

	assert.Equal(t, strings.TrimSpace(string(wantClaim)), string(bytes), "marshaled claim does not match expected")
}

func TestClaimSchema(t *testing.T) {
	t.Skip("This test is pending non-trivial updates to the Claim and Result objects: https://github.com/cnabio/cnab-go/issues/202")
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
