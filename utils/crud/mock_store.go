package crud

import (
	"fmt"
	"path"
	"strconv"
)

// The main point of these tests is to catch any case where the interface
// changes. But we also provide a mock for testing.
var _ Store = MockStore{}

type item struct {
	itemType, group, name string
	data                  []byte
}

type itemGroup struct {
	itemType, group string
	items           map[string]struct{}
}

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
	data map[string]*item

	// groups stores the groupings applied to the mocked data
	// itemType -> group -> list of keys
	groups map[string]*itemGroup

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
		groups: map[string]*itemGroup{},
		data:   map[string]*item{},
	}
}

func (s MockStore) Connect() error {
	// Keep track of Connect calls for test asserts later
	count, err := s.GetConnectCount()
	if err != nil {
		return err
	}
	s.setCount(connectCount, count+1)

	return nil
}

func (s MockStore) Close() error {
	// Keep track of Close calls for test asserts later
	count, err := s.GetCloseCount()
	if err != nil {
		return err
	}
	s.setCount(closeCount, count+1)

	return nil
}

func (s MockStore) key(itemType string, id string) string {
	return path.Join(itemType, id)
}

func (s MockStore) Count(itemType string, group string) (int, error) {
	names, err := s.List(itemType, group)
	return len(names), err
}

func (s MockStore) List(itemType string, group string) ([]string, error) {
	if s.ListMock != nil {
		return s.ListMock(itemType, group)
	}

	// List all items in a group, e.g. claims in an installation
	if g, ok := s.groups[s.key(itemType, group)]; ok {
		names := make([]string, 0, len(g.items))
		for name := range g.items {
			names = append(names, name)
		}
		return names, nil
	}

	return nil, nil
}

func (s MockStore) Save(itemType string, group string, name string, data []byte) error {
	if s.SaveMock != nil {
		return s.SaveMock(itemType, name, data)
	}

	g, ok := s.groups[s.key(itemType, group)]
	if !ok {
		g = &itemGroup{
			group:    group,
			itemType: itemType,
			items:    make(map[string]struct{}, 1),
		}
		s.groups[s.key(itemType, group)] = g
	}
	g.items[name] = struct{}{}

	i := &item{
		itemType: itemType,
		group:    group,
		name:     name,
		data:     data,
	}
	s.data[s.key(itemType, name)] = i

	return nil
}

func (s MockStore) Read(itemType string, name string) ([]byte, error) {
	if s.ReadMock != nil {
		return s.ReadMock(itemType, name)
	}

	if i, ok := s.data[s.key(itemType, name)]; ok {
		return i.data, nil
	}

	return nil, ErrRecordDoesNotExist
}

func (s MockStore) Delete(itemType string, name string) error {
	if s.DeleteMock != nil {
		return s.DeleteMock(itemType, name)
	}

	if i, ok := s.data[s.key(itemType, name)]; ok {
		delete(s.data, s.key(itemType, name))

		if g, ok := s.groups[s.key(itemType, i.group)]; ok {
			delete(g.items, i.name)
			if len(g.items) == 0 {
				delete(s.groups, s.key(itemType, i.group))
			}
		}
		return nil
	}

	return ErrRecordDoesNotExist
}

// GetConnectCount is for tests to safely read the Connect call count
// without accidentally triggering it by using Read.
func (s MockStore) GetConnectCount() (int, error) {
	countB, ok := s.data[s.key(mockStoreType, connectCount)]
	if !ok {
		return 0, nil
	}

	count, err := strconv.Atoi(string(countB.data))
	if err != nil {
		return 0, fmt.Errorf("could not convert connect-count %s to int: %v", string(countB.data), err)
	}

	return count, nil
}

// GetCloseCount is for tests to safely read the Close call count
// without accidentally triggering it by using Read.
func (s MockStore) GetCloseCount() (int, error) {
	countB, ok := s.data[s.key(mockStoreType, closeCount)]
	if !ok {
		return 0, nil
	}

	count, err := strconv.Atoi(string(countB.data))
	if err != nil {
		return 0, fmt.Errorf("could not convert close-count %s to int: %v", string(countB.data), err)
	}

	return count, nil
}

func (s MockStore) ResetCounts() {
	s.setCount(connectCount, 0)
	s.setCount(closeCount, 0)
}

func (s MockStore) setCount(count string, value int) {
	s.data[path.Join(mockStoreType, count)] = &item{
		itemType: mockStoreType,
		name:     count,
		data:     []byte(strconv.Itoa(value)),
	}
}
