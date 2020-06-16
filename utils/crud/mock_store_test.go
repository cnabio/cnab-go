package crud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockStoreWithGroups(t *testing.T) {
	// This is the structure used with claims
	// claims/
	// - INSTALLATION/
	//   - CLAIMID.json

	s := NewMockStore()
	is := assert.New(t)
	is.NoError(s.Save(testItemType, testGroup, "test", []byte("data")))

	list, err := s.List(testItemType, testGroup)
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
	is.Equal(groups, []string{testGroup})
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
