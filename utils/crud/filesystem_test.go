package crud

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Store = FileSystemStore{}

func TestFilesystemStore(t *testing.T) {
	is := assert.New(t)
	tmdir, err := ioutil.TempDir("", "cnab-test-")
	is.NoError(err)
	defer os.RemoveAll(tmdir)
	s := NewFileSystemStore(tmdir, map[string]string{testItemType: ".json"})
	key := "testkey"
	val := []byte("testval")
	is.NoError(s.Save(testItemType, testGroup, key, val))
	list, err := s.List(testItemType, testGroup)
	is.NoError(err)
	is.Len(list, 1)
	d, err := s.Read(testItemType, "testkey")
	is.NoError(err)
	is.Equal([]byte("testval"), d)
	is.NoError(s.Delete(testItemType, key))
	list, err = s.List(testItemType, testGroup)
	is.NoError(err)
	is.Len(list, 0)
}

func TestFileSystemStore_Count(t *testing.T) {
	tmdir, err := ioutil.TempDir("", "cnab-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmdir)
	s := NewFileSystemStore(tmdir, map[string]string{testItemType: ".json"})

	count, err := s.Count(testItemType, "")
	require.NoError(t, err, "Count failed")
	assert.Equal(t, 0, count, "Count should be 0 for an empty datastore")

	err = s.Save(testItemType, "", "key1", []byte("value1"))
	require.NoError(t, err, "Save failed")

	count, err = s.Count(testItemType, "")
	require.NoError(t, err, "Count failed")
	assert.Equal(t, 1, count, "Count should be 1 after adding an item")

	err = s.Delete(testItemType, "key1")
	require.NoError(t, err, "Delete failed")

	count, err = s.Count(testItemType, "")
	require.NoError(t, err, "Count failed")
	assert.Equal(t, 0, count, "Count should be 0 after deleting the item")
}
