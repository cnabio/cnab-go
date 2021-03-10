package crud

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Store = FileSystemStore{}

func TestFilesystemStore(t *testing.T) {
	testcases := []struct {
		name  string
		group string
		ext   string
	}{
		{
			name:  "no group supplied",
			group: "",
			ext:   ".json",
		}, {
			name:  "group supplied",
			group: testGroup,
			ext:   ".json",
		}, {
			name: "empty extension",
			ext:  "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tmdir, err := ioutil.TempDir("", "cnab-test-")
			require.NoError(t, err)
			defer os.RemoveAll(tmdir)

			s := NewFileSystemStore(tmdir, map[string]string{testItemType: tc.ext})
			keys := []string{"test.key1", "test.key2"} // Use periods in name to detect improper file extension checks
			val := []byte("testval")

			// Save some records
			for _, key := range keys {
				require.NoError(t, s.Save(testItemType, tc.group, key, val))
			}

			// List the records
			list, err := s.List(testItemType, tc.group)
			require.NoError(t, err)
			require.Equal(t, []string{"test.key1", "test.key2"}, list)

			// Read each record
			for _, key := range keys {
				d, err := s.Read(testItemType, key)
				require.NoError(t, err)
				require.Equal(t, []byte("testval"), d)
			}

			// Delete a record
			require.NoError(t, s.Delete(testItemType, keys[0]))

			// Verify list count
			list, err = s.List(testItemType, tc.group)
			require.NoError(t, err, "expected no error when listing directly from the item type directory")
			require.Len(t, list, len(keys)-1)

			// Verify that the group/parent dir remains
			groupDir, err := os.Stat(filepath.Join(tmdir, testItemType, tc.group))
			require.NoError(t, err, "expected the group/parent directory to exist")
			require.True(t, groupDir.IsDir())

			// Delete last record
			require.NoError(t, s.Delete(testItemType, keys[1]))

			// Verify group/parent dir removed
			if tc.group != "" {
				_, err := os.Stat(filepath.Join(tmdir, testItemType, tc.group))
				require.True(t, os.IsNotExist(err),
					"expected the parent group directory to be removed")
			}

			// Verify group/parent dir removed and error received
			// or list is empty
			list, err = s.List(testItemType, tc.group)
			if tc.group != "" {
				require.Equal(t, ErrRecordDoesNotExist, err,
					"expected an error when listing from a removed group directory")
			} else {
				require.NoError(t, err, "expected no error when listing directly from the item type directory")
				require.Len(t, list, 0)
			}

			// Verify the item type dir still exists
			itemTypeDir, err := os.Stat(filepath.Join(tmdir, testItemType))
			require.NoError(t, err, "expected the item type directory to exist")
			require.True(t, itemTypeDir.IsDir())
		})
	}
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
