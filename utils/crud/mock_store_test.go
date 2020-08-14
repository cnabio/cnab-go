package crud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockStoreWithGroups(t *testing.T) {
	s := NewMockStore()
	is := assert.New(t)
	is.NoError(s.Save(testItemType, testGroup, "test", []byte("data")))

	list, err := s.List(testItemType, testGroup)
	is.NoError(err)
	is.Len(list, 1)

	data, err := s.Read(testItemType, "test")
	is.NoError(err)
	is.Equal([]byte("data"), data)

	data, err = s.Read(testItemType, "not-exist")
	is.EqualError(err, ErrRecordDoesNotExist.Error())
	is.Empty(data)
}

func TestMockStoreWithoutGroups(t *testing.T) {
	// This is the structure used with credentials
	// credentials/
	// - NAME.json

	s := NewMockStore()
	is := assert.New(t)
	is.NoError(s.Save(testItemType, "", "test", []byte("data")))

	list, err := s.List(testItemType, "")
	is.NoError(err)
	is.Len(list, 1)

	data, err := s.Read(testItemType, "test")
	is.NoError(err)
	is.Equal(data, []byte("data"))

	data, err = s.Read(testItemType, "not-exist")
	is.EqualError(err, ErrRecordDoesNotExist.Error())
	is.Empty(data)

	groups, err := s.List(testItemType, "")
	is.NoError(err)
	is.Equal(groups, []string{"test"})
}

func TestMockStore_Count(t *testing.T) {
	s := NewMockStore()

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
