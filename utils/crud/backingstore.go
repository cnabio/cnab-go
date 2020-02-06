package crud

var _ Store = &BackingStore{}

// BackingStore wraps another store that may have Connect/Close methods that
// need to be called.
// - Connect is called before a method when the connection is closed.
// - Close is called after each method when AutoClose is true (default).
type BackingStore struct {
	AutoClose    bool
	opened       bool
	backingStore Store
}

func NewBackingStore(store Store) *BackingStore {
	return &BackingStore{
		AutoClose:    true,
		backingStore: store,
	}
}

func (s *BackingStore) Connect() error {
	if s.opened {
		return nil
	}
	if connectable, ok := s.backingStore.(HasConnect); ok {
		s.opened = true
		return connectable.Connect()
	}
	return nil
}

func (s *BackingStore) Close() error {
	if closable, ok := s.backingStore.(HasClose); ok {
		s.opened = false
		return closable.Close()
	}
	return nil
}

func (s *BackingStore) autoClose() error {
	if s.opened && s.AutoClose {
		return s.Close()
	}
	return nil
}

func (s *BackingStore) List(itemType string) ([]string, error) {
	err := s.Connect()
	if err != nil {
		return nil, err
	}

	defer s.autoClose()

	return s.backingStore.List(itemType)
}

func (s *BackingStore) Save(itemType string, name string, data []byte) error {
	err := s.Connect()
	if err != nil {
		return err
	}

	defer s.autoClose()

	return s.backingStore.Save(itemType, name, data)
}

func (s *BackingStore) Read(itemType string, name string) ([]byte, error) {
	err := s.Connect()
	if err != nil {
		return nil, err
	}

	defer s.autoClose()

	return s.backingStore.Read(itemType, name)
}

// ReadAll retrieves all the items.
func (s *BackingStore) ReadAll(itemType string) ([][]byte, error) {
	defer s.autoClose()

	autoClose := s.AutoClose
	s.AutoClose = false
	defer func() { s.AutoClose = autoClose }()

	results := make([][]byte, 0)
	list, err := s.List(itemType)
	if err != nil {
		return results, err
	}

	for _, name := range list {
		result, err := s.Read(itemType, name)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}

func (s *BackingStore) Delete(itemType string, name string) error {
	err := s.Connect()
	if err != nil {
		return err
	}

	defer s.autoClose()

	return s.backingStore.Delete(itemType, name)
}
