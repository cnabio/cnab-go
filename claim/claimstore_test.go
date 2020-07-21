package claim

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/utils/crud"
)

var _ Provider = Store{}

var b64encode = func(src []byte) ([]byte, error) {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(dst, src)
	return dst, nil
}

var b64decode = func(src []byte) ([]byte, error) {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	n, err := base64.StdEncoding.Decode(dst, src)
	return dst[:n], err
}

// generateClaimData creates test claims, results and outputs
// it returns a claim Provider, and a test cleanup function.
//
// claims/
//   foo/
//     CLAIM_ID_1 (install)
//     CLAIM_ID_2 (upgrade)
//     CLAIM_ID_3 (invoke - test)
//     CLAIM_ID_4 (uninstall)
//   bar/
//     CLAIM_ID_10 (install)
//   baz/
//     CLAIM_ID_20 (install)
//     CLAIM_ID_21 (install)
// results/
//   CLAIM_ID_1/
//     RESULT_ID_1 (success)
//   CLAIM_ID_2/
//     RESULT_ID 2 (success)
//   CLAIM_ID_3/
//     RESULT_ID_3 (failed)
//   CLAIM_ID_4/
//     RESULT_ID_4 (success)
//   CLAIM_ID_10/
//     RESULT_ID_10 (running)
//     RESULT_ID_11 (success)
//   CLAIM_ID_20/
//     RESULT_ID_20 (failed)
//   CLAIM_ID_21/
//     NO RESULT YET
// outputs/
//   RESULT_ID_1/
//     RESULT_ID_1_OUTPUT_1
//   RESULT_ID_2/
//     RESULT_ID_2_OUTPUT_1
//     RESULT_ID_2_OUTPUT_2
func generateClaimData(t *testing.T) (Provider, crud.MockStore) {
	backingStore := crud.NewMockStore()
	cp := NewClaimStore(crud.NewBackingStore(backingStore), nil, nil)

	bun := bundle.Bundle{
		Definitions: map[string]*definition.Schema{
			"output1": {
				Type: "string",
			},
			"output2": {
				Type: "string",
			},
		},
		Outputs: map[string]bundle.Output{
			"output1": {
				Definition: "output1",
			},
			"output2": {
				Definition: "output2",
				ApplyTo:    []string{"upgrade"},
			},
		},
	}
	createClaim := func(installation string, action string) Claim {
		c, err := New(installation, action, bun, nil)
		require.NoError(t, err, "New claim failed")

		err = cp.SaveClaim(c)
		require.NoError(t, err, "SaveClaim failed")

		return c
	}

	createResult := func(c Claim, status string) Result {
		r, err := c.NewResult(status)
		require.NoError(t, err, "NewResult failed")

		err = cp.SaveResult(r)
		require.NoError(t, err, "SaveResult failed")

		return r
	}

	createOutput := func(c Claim, r Result, name string) Output {
		o := NewOutput(c, r, name, []byte(c.Action+" "+name))

		err := cp.SaveOutput(o)
		require.NoError(t, err, "SaveOutput failed")

		return o
	}

	// Create the foo installation data
	const foo = "foo"
	c := createClaim(foo, ActionInstall)
	r := createResult(c, StatusSucceeded)
	createOutput(c, r, "output1")

	c = createClaim(foo, ActionUpgrade)
	r = createResult(c, StatusSucceeded)
	createOutput(c, r, "output1")
	createOutput(c, r, "output2")

	c = createClaim(foo, "test")
	createResult(c, StatusFailed)

	c = createClaim(foo, ActionUninstall)
	createResult(c, StatusSucceeded)

	// Create the bar installation data
	const bar = "bar"
	c = createClaim(bar, ActionInstall)
	createResult(c, StatusRunning)
	createResult(c, StatusSucceeded)

	// Create the baz installation data
	const baz = "baz"
	c = createClaim(baz, ActionInstall)
	createResult(c, StatusFailed)

	createClaim(baz, ActionInstall)

	backingStore.ResetCounts()
	return cp, backingStore
}

func assertSingleConnection(t *testing.T, datastore crud.MockStore) {
	t.Helper()

	connects, err := datastore.GetConnectCount()
	require.NoError(t, err, "GetConnectCount failed")
	assert.Equal(t, 1, connects, "expected a single connect")

	closes, err := datastore.GetCloseCount()
	require.NoError(t, err, "GetCloseCount failed")
	assert.Equal(t, 1, closes, "expected a single close")
}

func TestCanSaveReadAndDelete(t *testing.T) {
	is := assert.New(t)
	must := require.New(t)

	c1, err := New("foo", ActionUnknown, exampleBundle, nil)
	must.NoError(err)
	c1.Bundle = bundle.Bundle{Name: "foobundle", Version: "0.1.2"}

	tempDir, err := ioutil.TempDir("", "cnabtest")
	must.NoError(err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	datastore := crud.NewFileSystemStore(storeDir, NewClaimStoreFileExtensions())
	store := NewClaimStore(crud.NewBackingStore(datastore), nil, nil)

	err = store.SaveClaim(c1)
	must.NoError(err, "SaveClaim failed")
	_, err = datastore.Read(ItemTypeInstallations, c1.Installation)
	must.NoError(err, "A file representing the installation should have been created")

	c2, err := store.ReadLastClaim("foo")
	must.NoError(err, "ReadLastClaim failed")
	is.Equal(c2.Bundle, c1.Bundle, "Expected to read back bundle %s, got %s", c1.Bundle.Name, c2.Bundle.Name)

	installations, err := store.ListInstallations()
	must.NoError(err, "ListInstallations failed")
	is.Len(installations, 1)
	is.Equal(installations[0], c1.Installation)

	must.NoError(store.DeleteInstallation(c2.Installation))

	_, err = store.ReadClaim(c2.ID)
	is.Error(err, "Claims associated with the installation should have been deleted")

	installations, err = store.ListInstallations()
	must.NoError(err, "ListInstallations failed")
	is.Empty(installations, "The installation should have been deleted")

	_, err = datastore.Read(ItemTypeInstallations, c1.Installation)
	must.Error(err, "Installation should have been deleted")
	is.Contains(err.Error(), crud.ErrRecordDoesNotExist.Error(), "Installation should have been deleted")
}

func TestCanUpdate(t *testing.T) {
	is := assert.New(t)
	b := bundle.Bundle{Name: "foobundle", Version: "0.1.2"}
	c1, err := New("foo", ActionUnknown, b, nil)
	is.NoError(err)

	tempDir, err := ioutil.TempDir("", "cnabtest")
	is.NoError(err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	datastore := crud.NewFileSystemStore(storeDir, NewClaimStoreFileExtensions())
	store := NewClaimStore(crud.NewBackingStore(datastore), nil, nil)

	err = store.SaveClaim(c1)
	require.NoError(t, err)

	c2, err := c1.NewClaim(ActionInstall, b, nil)
	require.NoError(t, err, "NewClaim failed")

	err = store.SaveClaim(c2)
	is.NoError(err, "Failed to update")

	c3, err := store.ReadLastClaim("foo")
	is.NoError(err, "Failed to read")

	is.Equal(ActionInstall, c3.Action, "wrong action")
	is.NotEqual(c1.Revision, c3.Revision, "revision did not update")
}

func TestClaimStore_Installations(t *testing.T) {
	cp, datastore := generateClaimData(t)

	t.Run("ListInstallations", func(t *testing.T) {
		datastore.ResetCounts()
		installations, err := cp.ListInstallations()
		require.NoError(t, err, "ListInstallations failed")

		require.Len(t, installations, 3, "Expected 3 installations")
		assert.Equal(t, []string{"bar", "baz", "foo"}, installations)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadAllInstallationStatus", func(t *testing.T) {
		datastore.ResetCounts()
		installations, err := cp.ReadAllInstallationStatus()
		require.NoError(t, err, "ReadAllInstallationStatus failed")

		require.Len(t, installations, 3, "Expected 3 installations")
		bar := installations[0]
		baz := installations[1]
		foo := installations[2]

		// Validate the results were sorted by Name
		assert.Equal(t, "bar", bar.Name)
		assert.Equal(t, "baz", baz.Name)
		assert.Equal(t, "foo", foo.Name)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadInstallationStatus", func(t *testing.T) {
		datastore.ResetCounts()
		foo, err := cp.ReadInstallationStatus("foo")
		require.NoError(t, err, "ReadInstallationStatus failed")

		assert.Equal(t, "foo", foo.Name)

		// Validate enough information was set to render its status
		assert.Equal(t, StatusSucceeded, foo.GetLastStatus())
		lastClaim, err := foo.GetLastClaim()
		require.NoError(t, err, "GetLastClaim failed")
		assert.Equal(t, ActionUninstall, lastClaim.Action)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadInstallationStatus - invalid installation", func(t *testing.T) {
		foo, err := cp.ReadInstallationStatus("missing")
		require.EqualError(t, err, "Installation does not exist")
		assert.Empty(t, foo)
	})

	t.Run("ReadInstallation", func(t *testing.T) {
		datastore.ResetCounts()
		foo, err := cp.ReadInstallation("foo")
		require.NoError(t, err, "ReadInstallation failed")

		assert.Equal(t, "foo", foo.Name)
		require.Len(t, foo.Claims, 4, "Expected 4 claims")
		assert.Equal(t, StatusSucceeded, foo.GetLastStatus(), "expected the status to be loaded on the installation")
		assert.Equal(t, "foo", foo.Claims[0].Installation, "expected the claim to be associated with the installation")
		assert.Equal(t, ActionInstall, foo.Claims[0].Action)
		assert.Equal(t, ActionUpgrade, foo.Claims[1].Action)
		assert.Equal(t, "test", foo.Claims[2].Action)
		assert.Equal(t, ActionUninstall, foo.Claims[3].Action)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadInstallation - invalid installation", func(t *testing.T) {
		foo, err := cp.ReadInstallation("missing")
		require.EqualError(t, err, "Installation does not exist")
		assert.Empty(t, foo)
	})
}

func TestClaimStore_DeleteInstallation(t *testing.T) {
	cp, datastore := generateClaimData(t)

	err := cp.DeleteInstallation("foo")
	require.NoError(t, err, "DeleteInstallation failed")

	assertSingleConnection(t, datastore)

	names, err := cp.ListInstallations()
	require.NoError(t, err, "ListInstallations failed")
	assert.Equal(t, []string{"bar", "baz"}, names, "expected foo to be deleted completely")

	_, err = cp.ReadLastClaim("foo")
	require.EqualError(t, err, "Installation does not exist")
}

func TestClaimStore_Claims(t *testing.T) {
	cp, datastore := generateClaimData(t)

	t.Run("ReadAllClaims", func(t *testing.T) {
		datastore.ResetCounts()
		claims, err := cp.ReadAllClaims("foo")
		require.NoError(t, err, "Failed to read claims: %s", err)

		require.Len(t, claims, 4, "Expected 4 claims")
		assert.Equal(t, ActionInstall, claims[0].Action)
		assert.Equal(t, ActionUpgrade, claims[1].Action)
		assert.Equal(t, "test", claims[2].Action)
		assert.Equal(t, ActionUninstall, claims[3].Action)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadAllClaims - invalid installation", func(t *testing.T) {
		claims, err := cp.ReadAllClaims("missing")
		require.EqualError(t, err, "Installation does not exist")
		assert.Empty(t, claims)
	})

	t.Run("ListClaims", func(t *testing.T) {
		datastore.ResetCounts()
		claims, err := cp.ListClaims("foo")
		require.NoError(t, err, "Failed to read claims: %s", err)

		require.Len(t, claims, 4, "Expected 4 claims")

		assertSingleConnection(t, datastore)
	})

	t.Run("ListClaims - invalid installation", func(t *testing.T) {
		claims, err := cp.ListClaims("missing")
		require.EqualError(t, err, "Installation does not exist")
		assert.Empty(t, claims)
	})

	t.Run("ReadClaim", func(t *testing.T) {
		claims, err := cp.ListClaims("foo")
		require.NoError(t, err, "ListClaims failed")

		assert.NotEmpty(t, claims, "no claims were found")
		claimID := claims[0]

		datastore.ResetCounts()
		c, err := cp.ReadClaim(claimID)
		require.NoError(t, err, "ReadClaim failed")

		assert.Equal(t, "foo", c.Installation)
		assert.Equal(t, ActionInstall, c.Action)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadClaim - invalid claim", func(t *testing.T) {
		_, err := cp.ReadClaim("missing")
		require.EqualError(t, err, "Claim does not exist")
	})

	t.Run("ReadLastClaim", func(t *testing.T) {
		datastore.ResetCounts()
		c, err := cp.ReadLastClaim("bar")
		require.NoError(t, err, "ReadLastClaim failed")

		assert.Equal(t, "bar", c.Installation)
		assert.Equal(t, ActionInstall, c.Action)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadLastClaim - invalid installation", func(t *testing.T) {
		c, err := cp.ReadLastClaim("missing")
		require.EqualError(t, err, "Installation does not exist")
		assert.Empty(t, c)
	})
}

func TestClaimStore_Results(t *testing.T) {
	cp, datastore := generateClaimData(t)

	barClaims, err := cp.ListClaims("bar")
	require.NoError(t, err, "ListClaims failed")
	require.Len(t, barClaims, 1, "expected 1 claim")
	claimID := barClaims[0] // this claim has multiple results

	bazClaims, err := cp.ListClaims("baz")
	require.NoError(t, err, "ListClaims failed")
	require.Len(t, bazClaims, 2, "expected 2 claims")
	unfinishedClaimID := bazClaims[1] // this claim doesn't have any results yet

	t.Run("ListResults", func(t *testing.T) {
		datastore.ResetCounts()

		results, err := cp.ListResults(claimID)
		require.NoError(t, err, "ListResults failed")
		assert.Len(t, results, 2, "expected 2 results")

		assertSingleConnection(t, datastore)
	})

	t.Run("ListResults - unfinished claim", func(t *testing.T) {
		results, err := cp.ListResults(unfinishedClaimID)
		require.NoError(t, err, "listing results for a claim that doesn't have any yet should not result in an error")
		assert.Empty(t, results)
	})

	t.Run("ReadAllResults", func(t *testing.T) {
		datastore.ResetCounts()

		results, err := cp.ReadAllResults(claimID)
		require.NoError(t, err, "ReadAllResults failed")
		assert.Len(t, results, 2, "expected 2 results")

		assert.Equal(t, StatusRunning, results[0].Status)
		assert.Equal(t, StatusSucceeded, results[1].Status)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadAllResults - unfinished claim", func(t *testing.T) {
		results, err := cp.ReadAllResults(unfinishedClaimID)
		require.NoError(t, err, "reading results for a claim that doesn't have any yet should not result in an error")
		assert.Empty(t, results)
	})

	t.Run("ReadLastResult", func(t *testing.T) {
		datastore.ResetCounts()

		r, err := cp.ReadLastResult(claimID)
		require.NoError(t, err, "ReadLastResult failed")

		assert.Equal(t, StatusSucceeded, r.Status)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadLastResult - unfinished claim", func(t *testing.T) {
		results, err := cp.ReadAllResults(unfinishedClaimID)
		require.NoError(t, err, "reading results for a claim that doesn't have any yet should not result in an error")
		assert.Empty(t, results)
	})

	t.Run("ReadResult", func(t *testing.T) {
		results, err := cp.ListResults(claimID)
		require.NoError(t, err, "ListResults failed")

		resultID := results[0]

		datastore.ResetCounts()
		r, err := cp.ReadResult(resultID)
		require.NoError(t, err, "ReadResult failed")

		assert.Equal(t, StatusRunning, r.Status)

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadResult - invalid result", func(t *testing.T) {
		r, err := cp.ReadResult("missing")
		require.EqualError(t, err, "Result does not exist")
		assert.Empty(t, r)
	})
}

func TestClaimStore_Outputs(t *testing.T) {
	cp, datastore := generateClaimData(t)

	fooClaims, err := cp.ReadAllClaims("foo")
	require.NoError(t, err, "ReadAllClaims failed")
	require.NotEmpty(t, fooClaims, "expected foo to have a claim")
	fooClaim := fooClaims[1]
	fooResults, err := cp.ReadAllResults(fooClaim.ID) // Use foo's upgrade claim that has two outputs
	require.NoError(t, err, "ReadAllResults failed")
	require.NotEmpty(t, fooResults, "expected foo to have a result")
	fooResult := fooResults[0]
	resultID := fooResult.ID // this result has an output

	barClaims, err := cp.ReadAllClaims("bar")
	require.NoError(t, err, "ReadAllClaims failed")
	require.Len(t, barClaims, 1, "expected bar to have a claim")
	barClaim := barClaims[0]
	barResults, err := cp.ReadAllResults(barClaim.ID)
	require.NoError(t, err, "ReadAllResults failed")
	require.NotEmpty(t, barResults, "expected bar to have a result")
	barResult := barResults[0]
	resultIDWithoutOutputs := barResult.ID

	t.Run("ListOutputs", func(t *testing.T) {
		datastore.ResetCounts()
		outputs, err := cp.ListOutputs(resultID)
		require.NoError(t, err, "ListResults failed")
		assert.Len(t, outputs, 2, "expected 2 outputs")

		assert.Equal(t, "output1", outputs[0])
		assert.Equal(t, "output2", outputs[1])

		assertSingleConnection(t, datastore)
	})

	t.Run("ListOutputs - no outputs", func(t *testing.T) {
		outputs, err := cp.ListResults(resultIDWithoutOutputs)
		require.NoError(t, err, "listing outputs for a result that doesn't have any should not result in an error")
		assert.Empty(t, outputs)
	})

	t.Run("ReadLastOutputs", func(t *testing.T) {
		datastore.ResetCounts()
		outputs, err := cp.ReadLastOutputs("foo")

		require.NoError(t, err, "GetLastOutputs failed")
		assert.Equal(t, 2, outputs.Len(), "wrong number of outputs identified")

		gotOutput1, hasOutput1 := outputs.GetByName("output1")
		assert.True(t, hasOutput1, "should have found output1")
		assert.Equal(t, "upgrade output1", string(gotOutput1.Value), "did not find the most recent value for output1")

		gotOutput2, hasOutput2 := outputs.GetByName("output2")
		assert.True(t, hasOutput2, "should have found output2")
		assert.Equal(t, "upgrade output2", string(gotOutput2.Value), "did not find the most recent value for output2")

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadLastOutputs - invalid installation", func(t *testing.T) {
		outputs, err := cp.ReadLastOutputs("missing")
		require.EqualError(t, err, "Installation does not exist")
		assert.Empty(t, outputs)
	})

	t.Run("ReadLastOutput", func(t *testing.T) {
		datastore.ResetCounts()
		o, err := cp.ReadLastOutput("foo", "output1")

		require.NoError(t, err, "GetLastOutputs failed")
		assert.Equal(t, "upgrade output1", string(o.Value), "did not find the most recent value for output1")

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadLastOutput - invalid installation", func(t *testing.T) {
		o, err := cp.ReadLastOutput("missing", "output1")
		require.EqualError(t, err, "Installation does not exist")
		assert.Empty(t, o)
	})

	t.Run("ReadOutput", func(t *testing.T) {
		// Read the initial value of output1 from the install action
		installClaim := fooClaims[0]
		installResult, err := cp.ReadLastResult(installClaim.ID)
		require.NoError(t, err, "ReadLastResult failed")

		datastore.ResetCounts()

		o, err := cp.ReadOutput(installClaim, installResult, "output1")
		require.NoError(t, err, "ReadOutput failed")

		assert.Equal(t, "output1", o.Name)
		assert.Equal(t, installResult.ID, o.result.ID, "output.Result is not set")
		assert.Equal(t, installClaim.ID, o.result.claim.ID, "output.Result.Claim is not set")
		assert.Equal(t, "install output1", string(o.Value))

		assertSingleConnection(t, datastore)
	})

	t.Run("ReadOutput - no outputs", func(t *testing.T) {
		o, err := cp.ReadOutput(barClaim, barResult, "output1")
		require.EqualError(t, err, "Output does not exist")
		assert.Empty(t, o)
	})
}

func TestCanUpdateOutputs(t *testing.T) {
	is := assert.New(t)
	must := require.New(t)

	claim, err := New("foo", ActionUnknown, exampleBundle, nil)
	must.NoError(err)

	tempDir, err := ioutil.TempDir("", "cnabgotest")
	must.NoError(err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	fsStore := crud.NewFileSystemStore(storeDir, NewClaimStoreFileExtensions())
	store := NewClaimStore(crud.NewBackingStore(fsStore), nil, nil)

	err = store.SaveClaim(claim)
	must.NoError(err, "Failed to store claim")

	wantOutputs := OutputMetadata{
		"foo-output": true,
		"bar-output": "bar",
	}

	result, err := claim.NewResult(StatusSucceeded)
	require.NoError(t, err, "NewResult failed")

	result.OutputMetadata = OutputMetadata{
		"foo-output": true,
		"bar-output": "bar",
	}

	err = store.SaveResult(result)
	must.NoError(err, "Failed to store result with initial outputs")

	result, err = store.ReadResult(result.ID)
	must.NoError(err, "ReadResult failed")
	is.Equal(wantOutputs, result.OutputMetadata, "Wrong outputs on result")

	result.OutputMetadata["bar-output"] = "baz"

	err = store.SaveResult(result)
	must.NoError(err, "Failed to store result")

	result, err = store.ReadResult(result.ID)
	must.NoError(err, "Failed to read result")

	wantOutputs = OutputMetadata{
		"foo-output": true,
		"bar-output": "baz",
	}
	is.Equal(wantOutputs, result.OutputMetadata, "Wrong outputs on result")
}

func TestStore_EncryptClaims(t *testing.T) {
	s := NewMockStore(b64encode, b64decode)
	backingStore := s.GetBackingStore()

	err := s.SaveClaim(exampleClaim)
	require.NoError(t, err, "SaveClaim failed")

	// Verify that it was encrypted at rest
	encodedClaimB, err := backingStore.Read(ItemTypeClaims, exampleClaim.ID)
	require.NoError(t, err, "could not read raw claim data")
	var gotClaim Claim
	decodedClaimB, err := b64decode(encodedClaimB)
	require.NoError(t, err, "failed to decrypt raw claim data")
	err = json.Unmarshal(decodedClaimB, &gotClaim)
	require.NoError(t, err, "failed to unmarshal decrypted claim")
	assert.Equal(t, exampleClaim, gotClaim, "decoded claim doesn't match the original claim")

	// Verify that the claim is decrypted when read
	gotClaim, err = s.ReadClaim(exampleClaim.ID)
	require.NoError(t, err, "ReadClaim failed")
	assert.Equal(t, exampleClaim, gotClaim, "ReadClaim did not round trip the claim properly")
}

func TestStore_EncryptOutputs(t *testing.T) {
	writeOnly := func(value bool) *bool {
		return &value
	}
	s := NewMockStore(b64encode, b64decode)
	backingStore := s.GetBackingStore()

	b := bundle.Bundle{
		Definitions: map[string]*definition.Schema{
			"password": {
				WriteOnly: writeOnly(true),
			},
			"port": {
				WriteOnly: writeOnly(false),
			},
		},
		Outputs: map[string]bundle.Output{
			"password": {
				Definition: "password",
			},
			"port": {
				Definition: "port",
			},
		},
	}
	c, err := New("wordpress", ActionInstall, b, nil)
	require.NoError(t, err, "New claim failed")

	r, err := c.NewResult(StatusSucceeded)
	require.NoError(t, err, "NewResult failed")

	err = s.SaveClaim(c)
	require.NoError(t, err, "SaveClaim failed")
	err = s.SaveResult(r)
	require.NoError(t, err, "SaveResult failed")

	password := NewOutput(c, r, "password", []byte("mypassword"))
	err = s.SaveOutput(password)
	require.NoError(t, err, "SaveOutput failed")

	// Verify that password was encrypted at rest
	encryptedOutputB, err := backingStore.Read(ItemTypeOutputs, s.outputKey(r.ID, password.Name))
	require.NoError(t, err, "could not read raw output data")
	decryptedOutputB, err := b64decode(encryptedOutputB)
	require.NoError(t, err, "failed to decrypt raw output data")
	assert.Equal(t, string(password.Value), string(decryptedOutputB), "decrypted output doesn't match the original output")

	// Verify the password is decrypted by the claim store automatically
	retrievedPassword, err := s.ReadOutput(c, r, "password")
	require.NoError(t, err, "ReadOutput failed")
	assert.Equal(t, string(password.Value), string(retrievedPassword.Value), "ReadOutput didn't decrypt the output automatically")

	port := NewOutput(c, r, "port", []byte("8080"))
	err = s.SaveOutput(port)
	require.NoError(t, err, "SaveOutput failed")

	// Verify that port was not encrypted at rest because it's not sensitive
	outputB, err := backingStore.Read(ItemTypeOutputs, s.outputKey(r.ID, port.Name))
	require.NoError(t, err, "could not read raw output data")
	assert.Equal(t, string(port.Value), string(outputB), "output doesn't match the original output")

	// Verify that it is read without mangling
	gotPort, err := s.ReadOutput(c, r, "port")
	require.NoError(t, err, "ReadOutput failed")
	assert.Equal(t, string(port.Value), string(gotPort.Value), "output doesn't match the original output")
}
