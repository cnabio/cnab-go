package crud

import (
	"fmt"
	"strconv"
)

// The main point of these tests is to catch any case where the interface
// changes. But we also provide a mock for testing.
var _ Store = MockStore{}

const (
	connectCount  = "connect-count"
	closeCount    = "close-count"
	testItemType  = "test-items"
	testGroup     = "test-group"
	mockStoreType = "mock-store"
)

// MockStore is an in-memory store with optional mocked functionality that is
// intended for use with unit testing.
type MockStore struct {
	// data stores the mocked data
	// itemType -> name -> data
	data map[string]map[string][]byte

	// groups stores the groupings applied to the mocked data
	// itemType -> group -> list of names
	groups map[string]map[string][]string

	// DeleteMock replaces the default Delete implementation with the specified function.
	// This allows for simulating failures.
	DeleteMock func(itemType string, name string) error

	// ListMock replaces the default List implementation with the specified function.
	// This allows for simulating failures.
	ListMock func(itemType string, group string) ([]string, error)

	// ReadMock replaces the default Read implementation with the specified function.
	// This allows for simulating failures.
	ReadMock func(itemType string, name string) ([]byte, error)

	// SaveMock replaces the default Save implementation with the specified function.
	// This allows for simulating failures.
	SaveMock func(itemType string, name string, data []byte) error
}

func NewMockStore() MockStore {
	return MockStore{
		groups: map[string]map[string][]string{},
		data:   map[string]map[string][]byte{},
	}
}

func (s MockStore) Connect() error {
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

func (s MockStore) Close() error {
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

func (s MockStore) List(itemType string, group string) ([]string, error) {
	if s.ListMock != nil {
		return s.ListMock(itemType, group)
	}

	if groups, ok := s.groups[itemType]; ok {
		if names, ok := groups[group]; ok {
			buf := make([]string, len(names))
			i := 0
			for _, name := range names {
				buf[i] = name
				i++
			}
			return buf, nil
		}

		if group == "" {
			// List all the groups, e.g. if we were listing claims, this would list the installation names
			names := make([]string, 0, len(groups))
			for groupName := range groups {
				names = append(names, groupName)
			}
			return names, nil
		}
	}

	return nil, nil
}

func (s MockStore) Save(itemType string, group string, name string, data []byte) error {
	if s.SaveMock != nil {
		return s.SaveMock(itemType, name, data)
	}

	groupNames, ok := s.groups[itemType]
	if !ok {
		groupNames = map[string][]string{
			group: make([]string, 0, 1),
		}
		s.groups[itemType] = groupNames
	}
	groupNames[group] = append(groupNames[group], name)

	itemData, ok := s.data[itemType]
	if !ok {
		itemData = make(map[string][]byte, 1)
		s.data[itemType] = itemData
	}

	itemData[name] = data
	return nil
}

func (s MockStore) Read(itemType string, name string) ([]byte, error) {
	if s.ReadMock != nil {
		return s.ReadMock(itemType, name)
	}

	if itemData, ok := s.data[itemType]; ok {
		if data, ok := itemData[name]; ok {
			return data, nil
		}
	}

	return nil, ErrRecordDoesNotExist
}

func (s MockStore) Delete(itemType string, name string) error {
	if s.DeleteMock != nil {
		return s.DeleteMock(itemType, name)
	}

	if itemData, ok := s.data[itemType]; ok {
		if _, ok := itemData[name]; ok {
			delete(itemData, name)
			return nil
		}
	}

	return ErrRecordDoesNotExist
}

// GetConnectCount is for tests to safely read the Connect call count
// without accidentally triggering it by using Read.
func (s MockStore) GetConnectCount() (int, error) {
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
func (s MockStore) GetCloseCount() (int, error) {
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
