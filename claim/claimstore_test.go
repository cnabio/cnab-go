package claim

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
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

func TestCanSaveReadAndDelete(t *testing.T) {
	is := assert.New(t)
	must := require.New(t)

	c1, err := New("foo", ActionUnknown, exampleBundle, nil)
	must.NoError(err)
	c1.Bundle = bundle.Bundle{Name: "foobundle", Version: "0.1.2"}

	tempDir, err := ioutil.TempDir("", "duffletest")
	must.NoError(err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, NewClaimStoreFileExtensions()), nil, nil)

	err = store.SaveClaim(c1)
	must.NoError(err, "Failed to store: %s", err)

	c2, err := store.ReadLastClaim("foo")
	must.NoError(err, "Failed to read: %s", err)
	is.Equal(c2.Bundle, c1.Bundle, "Expected to read back bundle %s, got %s", c1.Bundle.Name, c2.Bundle.Name)

	installations, err := store.ListInstallations()
	must.NoError(err, "Failed to list: %s", err)
	is.Len(installations, 1)
	is.Equal(installations[0], c1.Installation)

	must.NoError(store.DeleteClaim(c2.ID))

	_, err = store.ReadClaim(c2.ID)
	is.Error(err, "Should have had error reading after deletion but did not")
}

func TestCanUpdate(t *testing.T) {
	is := assert.New(t)
	b := bundle.Bundle{Name: "foobundle", Version: "0.1.2"}
	c1, err := New("foo", ActionUnknown, b, nil)
	is.NoError(err)

	tempDir, err := ioutil.TempDir("", "duffletest")
	is.NoError(err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, NewClaimStoreFileExtensions()), nil, nil)

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

func TestListInstallations(t *testing.T) {
	is := assert.New(t)

	tempDir, err := ioutil.TempDir("", "duffletest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, NewClaimStoreFileExtensions()), nil, nil)

	b1 := bundle.Bundle{Name: "foobundle", Version: "0.1.0"}
	c1, err := New("foo", ActionUnknown, b1, nil)
	is.NoError(err)

	is.NoError(store.SaveClaim(c1), "Failed to store: %s", err)

	b2 := bundle.Bundle{Name: "barbundle", Version: "0.1.0"}
	c2, err := New("bar", ActionUnknown, b2, nil)
	is.NoError(err)

	is.NoError(store.SaveClaim(c2), "Failed to store: %s", err)

	b3 := bundle.Bundle{Name: "bazbundle", Version: "0.1.0"}
	c3, err := New("baz", ActionUnknown, b3, nil)
	is.NoError(err)

	is.NoError(store.SaveClaim(c3), "Failed to store: %s", err)

	installations, err := store.ListInstallations()
	is.NoError(err, "Failed to read claims: %s", err)

	is.Len(installations, 3)
	is.Equal("bar", installations[0])
	is.Equal("baz", installations[1])
	is.Equal("foo", installations[2])
}

func TestReadAll(t *testing.T) {
	is := assert.New(t)
	must := assert.New(t)

	tempDir, err := ioutil.TempDir("", "duffletest")
	must.NoError(err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, NewClaimStoreFileExtensions()), nil, nil)

	b := bundle.Bundle{Name: "foobundle", Version: "0.1.0"}
	c1, err := New("foo", ActionInstall, b, nil)
	must.NoError(err)
	t.Log(c1.ID)
	must.NoError(store.SaveClaim(c1), "Failed to store: %s", err)

	c2, err := c1.NewClaim(ActionUpgrade, b, nil)
	must.NoError(err, "NewClaim failed")
	must.NoError(store.SaveClaim(c2), "Failed to store: %s", err)

	c3, err := c1.NewClaim(ActionUninstall, b, nil)
	must.NoError(err, "NewClaim failed")
	must.NoError(store.SaveClaim(c3), "Failed to store: %s", err)

	claims, err := store.ReadAllClaims(c1.Installation)
	must.NoError(err, "Failed to read claims: %s", err)

	must.Len(claims, 3)
	t.Log(claims[0].ID, claims[1].ID, claims[2].ID)
	is.Equal(ActionInstall, claims[0].Action)
	is.Equal(ActionUpgrade, claims[1].Action)
	is.Equal(ActionUninstall, claims[2].Action)
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
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, NewClaimStoreFileExtensions()), nil, nil)

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

func TestClaimStore_HandlesNotFoundError(t *testing.T) {
	mockStore := crud.NewMockStore()
	mockStore.ReadMock = func(itemType string, name string) (bytes []byte, err error) {
		// Change the default error message to test that we are checking
		// inside the error message and not matching it exactly
		return nil, errors.New("wrapping error message: " + crud.ErrRecordDoesNotExist.Error())
	}
	cs := NewClaimStore(mockStore, nil, nil)

	_, err := cs.ReadClaim("missing claim")
	assert.EqualError(t, err, ErrClaimNotFound.Error())
}

func TestStore_EncryptClaims(t *testing.T) {
	backingStore := crud.NewMockStore()
	s := NewClaimStore(backingStore, b64encode, b64decode)

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
	backingStore := crud.NewMockStore()
	s := NewClaimStore(backingStore, b64encode, b64decode)

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

	password := Output{
		Claim:  c,
		Result: r,
		Name:   "password",
		Value:  []byte("mypassword"),
	}
	err = s.SaveOutput(password)
	require.NoError(t, err, "SaveOutput failed")

	// Verify that password was encrypted at rest
	encryptedOutputB, err := backingStore.Read(ItemTypeOutputs, s.outputKey(r.ID, password.Name))
	require.NoError(t, err, "could not read raw output data")
	decryptedOutputB, err := b64decode(encryptedOutputB)
	require.NoError(t, err, "failed to decrypt raw output data")
	assert.Equal(t, string(password.Value), string(decryptedOutputB), "decrypted output doesn't match the original output")

	port := Output{
		Claim:  c,
		Result: r,
		Name:   "port",
		Value:  []byte("8080"),
	}
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
