package claim

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/qri-io/jsonschema"

	"github.com/deislabs/cnab-go/bundle"
)

func TestNew(t *testing.T) {
	// Make sure that the default Result has status and action set.
	claim, err := New("my_claim")
	assert.NoError(t, err)

	assert.Equal(t, "my_claim", claim.Name, "Name is set")
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

	claim.Update(ActionInstall, StatusSuccess)

	is := assert.New(t)
	is.NotEqual(oldMod, claim.Modified)
	is.NotEqual(oldUlid, claim.Revision)
	is.Equal("install", claim.Result.Action)
	is.Equal("success", claim.Result.Status)
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
		SchemaVersion: "schemaVersion",
		Name:          "mybun",
		Version:       "v0.1.0",
		Description:   "this is my bundle",
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

	assert.Equal(t, string(wantClaim), string(bytes), "marshaled claim does not match expected")
}

var exampleClaim = Claim{
	Name:     "my_claim",
	Revision: staticRevision,
	Created:  staticDate,
	Modified: staticDate,
	Bundle:   &exampleBundle,
	Result: Result{
		Action:  ActionInstall,
		Message: "result message",
		Status:  StatusUnderway,
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

func TestMarshal_AllFields(t *testing.T) {
	bytes, err := json.Marshal(exampleClaim)
	assert.NoError(t, err, "failed to json.Marshal claim")

	wantClaim, err := ioutil.ReadFile("testdata/claim.allfields.json")
	assert.NoError(t, err, "failed to read test claim")

	assert.Equal(t, string(wantClaim), string(bytes), "marshaled claim does not match expected")
}

func TestClaimSchema(t *testing.T) {
	t.Skip("this test is currently a work in progress; see issue comment below")

	claimBytes, err := json.Marshal(exampleClaim)
	assert.NoError(t, err, "failed to json.Marshal the claim")

	url := "https://raw.githubusercontent.com/deislabs/cnab-spec/master/schema/claim.schema.json"
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err, "failed to construct GET request for fetching claim schema")
	res, err := http.DefaultClient.Do(req)
	assert.NoError(t, err, "failed to get claim schema")

	defer res.Body.Close()
	schemaData, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err, "failed to read claim schema")

	rs := &jsonschema.RootSchema{}
	// This currently fails; needs https://github.com/deislabs/cnab-spec/pull/243
	err = json.Unmarshal(schemaData, rs)
	assert.NoError(t, err, "failed to json.Unmarshal root claim schema")

	// This currently fails due to https://github.com/deislabs/cnab-spec/issues/241
	// Thus, since the referenced bundle schema can't be fetched, schema validation is impaired
	// Alternatively, we could read the qri-o/jsonschema docs to see how we might 'seed' our Validator
	// with a fetched version of the bundle schema (from GitHub, as above for the claim schema)
	err = rs.FetchRemoteReferences()
	assert.NoError(t, err, "failed to fetch remote references declared by claim schema")

	errors, err := rs.ValidateBytes(claimBytes)
	assert.NoError(t, err, "failed to validate claim")

	if len(errors) > 0 {
		t.Log("claim validation against the JSON schema failed:")
		for _, error := range errors {
			t.Log(error)
		}
		t.Fail()
	}
}
