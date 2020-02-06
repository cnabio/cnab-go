package crud

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The main point of these tests is to catch any case where the interface
// changes. But we also provide a mock for testing.
var _ Store = &MockStore{}

const (
	ConnectCount  = "connect-count"
	CloseCount    = "close-count"
	TestItemType  = "test-items"
	MockStoreType = "mock-store"
)

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
	return &MockStore{data: map[string]map[string][]byte{}}
}

func (s *MockStore) Connect() error {
	_, ok := s.data[MockStoreType]
	if !ok {
		s.data[MockStoreType] = make(map[string][]byte, 1)
	}

	// Keep track of Connect calls for test asserts later
	count, err := s.GetConnectCount()
	if err != nil {
		return err
	}

	s.data[MockStoreType][ConnectCount] = []byte(strconv.Itoa(count + 1))

	return nil
}

func (s *MockStore) Close() error {
	_, ok := s.data[MockStoreType]
	if !ok {
		s.data[MockStoreType] = make(map[string][]byte, 1)
	}

	// Keep track of Close calls for test asserts later
	count, err := s.GetCloseCount()
	if err != nil {
		return err
	}

	s.data[MockStoreType][CloseCount] = []byte(strconv.Itoa(count + 1))

	return nil
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

// GetConnectCount is for tests to safely read the Connect call count
// without accidentally triggering it by using Read.
func (s *MockStore) GetConnectCount() (int, error) {
	countB, ok := s.data[MockStoreType][ConnectCount]
	if !ok {
		countB = []byte("0")
	}

	count, err := strconv.Atoi(string(countB))
	if err != nil {
		return 0, fmt.Errorf("could not convert connect-count %s to int: %v", string(countB), err)
	}

	return count, nil
}

// GetCloseCount is for tests to safely read the Close call count
// without accidentally triggering it by using Read.
func (s *MockStore) GetCloseCount() (int, error) {
	countB, ok := s.data[MockStoreType][CloseCount]
	if !ok {
		countB = []byte("0")
	}

	count, err := strconv.Atoi(string(countB))
	if err != nil {
		return 0, fmt.Errorf("could not convert close-count %s to int: %v", string(countB), err)
	}

	return count, nil
}
