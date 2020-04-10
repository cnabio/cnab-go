package crud

import (
	"fmt"
	"strconv"
)

// The main point of these tests is to catch any case where the interface
// changes. But we also provide a mock for testing.
var _ Store = &MockStore{}

const (
	connectCount  = "connect-count"
	closeCount    = "close-count"
	testItemType  = "test-items"
	mockStoreType = "mock-store"
)

// MockStore is an in-memory store with optional mocked functionality that is
// intended for use with unit testing.
type MockStore struct {
	data map[string]map[string][]byte

	// DeleteMock replaces the default Delete implementation with the specified function.
	// This allows for simulating failures.
	DeleteMock func(itemType string, name string) error

	// ListMock replaces the default List implementation with the specified function.
	// This allows for simulating failures.
	ListMock func(itemType string) ([]string, error)

	// ReadMock replaces the default Read implementation with the specified function.
	// This allows for simulating failures.
	ReadMock func(itemType string, name string) ([]byte, error)

	// SaveMock replaces the default Save implementation with the specified function.
	// This allows for simulating failures.
	SaveMock func(itemType string, name string, data []byte) error
}

func NewMockStore() *MockStore {
	return &MockStore{data: map[string]map[string][]byte{}}
}

func (s *MockStore) Connect() error {
	_, ok := s.data[mockStoreType]
	if !ok {
		s.data[mockStoreType] = make(map[string][]byte, 1)
	}

	// Keep track of Connect calls for test asserts later
	count, err := s.GetConnectCount()
	if err != nil {
		return err
	}

	s.data[mockStoreType][connectCount] = []byte(strconv.Itoa(count + 1))

	return nil
}

func (s *MockStore) Close() error {
	_, ok := s.data[mockStoreType]
	if !ok {
		s.data[mockStoreType] = make(map[string][]byte, 1)
	}

	// Keep track of Close calls for test asserts later
	count, err := s.GetCloseCount()
	if err != nil {
		return err
	}

	s.data[mockStoreType][closeCount] = []byte(strconv.Itoa(count + 1))

	return nil
}

func (s *MockStore) List(itemType string) ([]string, error) {
	if s.ListMock != nil {
		return s.ListMock(itemType)
	}

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
	if s.SaveMock != nil {
		return s.SaveMock(itemType, name, data)
	}

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
	if s.ReadMock != nil {
		return s.ReadMock(itemType, name)
	}

	if itemData, ok := s.data[itemType]; ok {
		return itemData[name], nil
	}

	return nil, nil
}

func (s *MockStore) Delete(itemType string, name string) error {
	if s.DeleteMock != nil {
		return s.DeleteMock(itemType, name)
	}

	if itemData, ok := s.data[itemType]; ok {
		delete(itemData, name)
		return nil
	}
	return nil
}

// GetConnectCount is for tests to safely read the Connect call count
// without accidentally triggering it by using Read.
func (s *MockStore) GetConnectCount() (int, error) {
	countB, ok := s.data[mockStoreType][connectCount]
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
	countB, ok := s.data[mockStoreType][closeCount]
	if !ok {
		countB = []byte("0")
	}

	count, err := strconv.Atoi(string(countB))
	if err != nil {
		return 0, fmt.Errorf("could not convert close-count %s to int: %v", string(countB), err)
	}

	return count, nil
}
