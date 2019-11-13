package crud

var _ Store = &BackingStore{}

// BackingStore wraps another store that may have Connect/Close methods that
// need to be called.
// - Connect is called before a method when the connection is closed.
// - Close is called after each method when AutoClose is true (default).
type BackingStore struct {
	AutoClose    bool
	closed       bool
	backingStore Store
}

func NewBackingStore(store Store) *BackingStore {
	return &BackingStore{
		AutoClose:    true,
		closed:       true,
		backingStore: store,
	}
}

func (s *BackingStore) Connect() error {
	if !s.closed {
		return nil
	}
	if connectable, ok := s.backingStore.(HasConnect); ok {
		s.closed = false
		return connectable.Connect()
	}
	return nil
}

func (s *BackingStore) Close() error {
	if closable, ok := s.backingStore.(HasClose); ok {
		s.closed = true
		return closable.Close()
	}
	return nil
}

func (s *BackingStore) List() ([]string, error) {
	err := s.Connect()
	if err != nil {
		return nil, err
	}

	if s.AutoClose {
		defer s.Close()
	}

	return s.backingStore.List()
}

func (s *BackingStore) Save(name string, data []byte) error {
	err := s.Connect()
	if err != nil {
		return err
	}

	if s.AutoClose {
		defer s.Close()
	}

	return s.backingStore.Save(name, data)
}

func (s *BackingStore) Read(name string) ([]byte, error) {
	err := s.Connect()
	if err != nil {
		return nil, err
	}

	if s.AutoClose {
		defer s.Close()
	}

	return s.backingStore.Read(name)
}

// ReadAll retrieves all the items.
func (s *BackingStore) ReadAll() ([][]byte, error) {
	if s.AutoClose {
		defer s.Close()
	}

	autoClose := s.AutoClose
	s.AutoClose = false

	results := make([][]byte, 0)
	list, err := s.List()
	if err != nil {
		return results, err
	}

	for _, name := range list {
		result, err := s.Read(name)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	s.AutoClose = autoClose

	return results, nil
}

func (s *BackingStore) Delete(name string) error {
	err := s.Connect()
	if err != nil {
		return err
	}

	if s.AutoClose {
		defer s.Close()
	}

	return s.backingStore.Delete(name)
}
