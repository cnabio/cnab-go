package claim

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/utils/crud"
)

func TestCanSaveReadAndDelete(t *testing.T) {
	is := assert.New(t)
	claim, err := New("foo")
	is.NoError(err)
	claim.Bundle = &bundle.Bundle{Name: "foobundle", Version: "0.1.2"}
	claim.RelocationMap = bundle.ImageRelocationMap{
		"some.registry/image1": "some.other.registry/image1",
	}

	tempDir, err := ioutil.TempDir("", "duffletest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, "json"))

	is.NoError(store.Store(*claim), "Failed to store: %s", err)

	c, err := store.Read("foo")
	is.NoError(err, "Failed to read: %s", err)
	is.Equal(c.Bundle, claim.Bundle, "Expected to read back bundle %s, got %s", claim.Bundle.Name, c.Bundle.Name)
	is.Equal("some.other.registry/image1", c.RelocationMap["some.registry/image1"])

	claims, err := store.List()
	is.NoError(err, "Failed to list: %s", err)
	is.Len(claims, 1)
	is.Equal(claims[0], claim.Name)

	is.NoError(store.Delete("foo"))

	_, err = store.Read("foo")
	is.Error(err, "Should have had error reading after deletion but did not")
}

func TestCanUpdate(t *testing.T) {
	is := assert.New(t)
	claim, err := New("foo")
	is.NoError(err)
	claim.Bundle = &bundle.Bundle{Name: "foobundle", Version: "0.1.2"}
	rev := claim.Revision

	tempDir, err := ioutil.TempDir("", "duffletest")
	is.NoError(err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, "json"))

	err = store.Store(*claim)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	claim.Update(ActionInstall, StatusSuccess)

	err = store.Store(*claim)
	is.NoError(err, "Failed to update")

	c, err := store.Read("foo")
	is.NoError(err, "Failed to read")

	is.Equal(ActionInstall, c.Result.Action, "wrong action")
	is.NotEqual(rev, c.Revision, "revision did not update")
}

func TestReadAll(t *testing.T) {
	is := assert.New(t)

	tempDir, err := ioutil.TempDir("", "duffletest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, "json"))

	claim, err := New("foo")
	is.NoError(err)
	claim.Bundle = &bundle.Bundle{Name: "foobundle", Version: "0.1.0"}

	is.NoError(store.Store(*claim), "Failed to store: %s", err)

	claim2, err := New("bar")
	is.NoError(err)
	claim2.Bundle = &bundle.Bundle{Name: "barbundle", Version: "0.1.0"}

	is.NoError(store.Store(*claim2), "Failed to store: %s", err)

	claim3, err := New("baz")
	is.NoError(err)
	claim3.Bundle = &bundle.Bundle{Name: "bazbundle", Version: "0.1.0"}

	is.NoError(store.Store(*claim3), "Failed to store: %s", err)

	claims, err := store.ReadAll()
	is.NoError(err, "Failed to read claims: %s", err)

	is.Len(claims, 3)
	is.Equal("foo", claim.Name)
	is.Equal("bar", claim2.Name)
	is.Equal("baz", claim3.Name)
}
