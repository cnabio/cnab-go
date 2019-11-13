package crud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The main point of these tests is to catch any case where the interface
// changes. But we also provide a mock for testing.
var _ Store = &MockStore{}

const TestItemType = "test-items"

func TestMockStore(t *testing.T) {
	s := NewMockStore()
	is := assert.New(t)
	is.NoError(s.Save(TestItemType, "test", []byte("data")))
	list, err := s.List(TestItemType)
	is.NoError(err)
	is.Len(list, 1)
	data, err := s.Read(TestItemType, "test")
	is.NoError(err)
	is.Equal(data, []byte("data"))

}

type MockStore struct {
	data map[string]map[string][]byte
}

func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string]map[string][]byte),
	}
}

func (s *MockStore) List(itemType string) ([]string, error) {
	if itemData, ok := s.data[itemType]; ok {
		buf := make([]string, len(itemData))
		i := 0
		for k := range itemData {
			buf[i] = k
			i++
		}
		return buf, nil
	}

	return nil, nil
}

func (s *MockStore) Save(itemType string, name string, data []byte) error {
	var itemData map[string][]byte
	itemData, ok := s.data[itemType]
	if !ok {
		itemData = make(map[string][]byte, 1)
		s.data[itemType] = itemData
	}

	itemData[name] = data
	return nil
}

func (s *MockStore) Read(itemType string, name string) ([]byte, error) {
	if itemData, ok := s.data[itemType]; ok {
		return itemData[name], nil
	}

	return nil, nil
}

func (s *MockStore) Delete(itemType string, name string) error {
	if itemData, ok := s.data[itemType]; ok {
		delete(itemData, name)
		return nil
	}

	return nil
}
