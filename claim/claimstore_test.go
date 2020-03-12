package claim

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/utils/crud"
)

func TestCanSaveReadAndDelete(t *testing.T) {
	is := assert.New(t)
	claim, err := New("foo")
	is.NoError(err)
	claim.Bundle = &bundle.Bundle{Name: "foobundle", Version: "0.1.2"}

	tempDir, err := ioutil.TempDir("", "duffletest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, "json"))

	err = store.Save(*claim)
	is.NoError(err, "Failed to store: %s", err)

	c, err := store.Read("foo")
	is.NoError(err, "Failed to read: %s", err)
	is.Equal(c.Bundle, claim.Bundle, "Expected to read back bundle %s, got %s", claim.Bundle.Name, c.Bundle.Name)

	claims, err := store.List()
	is.NoError(err, "Failed to list: %s", err)
	is.Len(claims, 1)
	is.Equal(claims[0], claim.Installation)

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

	err = store.Save(*claim)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	claim.Update(ActionInstall, StatusSucceeded)

	err = store.Save(*claim)
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

	is.NoError(store.Save(*claim), "Failed to store: %s", err)

	claim2, err := New("bar")
	is.NoError(err)
	claim2.Bundle = &bundle.Bundle{Name: "barbundle", Version: "0.1.0"}

	is.NoError(store.Save(*claim2), "Failed to store: %s", err)

	claim3, err := New("baz")
	is.NoError(err)
	claim3.Bundle = &bundle.Bundle{Name: "bazbundle", Version: "0.1.0"}

	is.NoError(store.Save(*claim3), "Failed to store: %s", err)

	claims, err := store.ReadAll()
	is.NoError(err, "Failed to read claims: %s", err)

	is.Len(claims, 3)
	is.Equal("foo", claim.Installation)
	is.Equal("bar", claim2.Installation)
	is.Equal("baz", claim3.Installation)
}

func TestCanUpdateOutputs(t *testing.T) {
	is := assert.New(t)
	claim, err := New("foo")
	is.NoError(err)
	is.Equal(map[string]interface{}{}, claim.Outputs)

	tempDir, err := ioutil.TempDir("", "cnabgotest")
	is.NoError(err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	storeDir := filepath.Join(tempDir, "claimstore")
	store := NewClaimStore(crud.NewFileSystemStore(storeDir, "json"))

	claim.Outputs = map[string]interface{}{
		"foo-output": true,
		"bar-output": "bar",
	}

	err = store.Save(*claim)
	is.NoError(err, "Failed to store claim")

	c, err := store.Read("foo")
	is.NoError(err, "Failed to read claim")

	want := map[string]interface{}{
		"foo-output": true,
		"bar-output": "bar",
	}
	is.Equal(want, c.Outputs, "Wrong outputs on claim")

	claim.Outputs["bar-output"] = "baz"

	err = store.Save(*claim)
	is.NoError(err, "Failed to store claim")

	c, err = store.Read("foo")
	is.NoError(err, "Failed to read claim")

	want = map[string]interface{}{
		"foo-output": true,
		"bar-output": "baz",
	}
	is.Equal(want, c.Outputs, "Wrong outputs on claim")
}
