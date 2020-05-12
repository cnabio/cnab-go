package crud

var _ Store = &BackingStore{}

// BackingStore wraps another store that may have Connect/Close methods that
// need to be called.
// - Connect is called before a method when the connection is closed.
// - Close is called after each method when AutoClose is true (default).
type BackingStore struct {

	// AutoClose specifies if the connection should be automatically
	// closed when done accessing the backing store.
	AutoClose bool

	// opened specifies if the backing store's connect has been called
	// and has not been closed yet.
	opened bool

	// connect handler for the backing store, if defined.
	connect func() error

	// close handler for the backing store, if defined.
	close func() error

	// backingStore being wrapped.
	backingStore Store
}

func NewBackingStore(store Store) *BackingStore {
	backingStore := BackingStore{
		AutoClose:    true,
		backingStore: store,
	}

	if connectable, ok := store.(HasConnect); ok {
		backingStore.connect = connectable.Connect
	}

	if closable, ok := store.(HasClose); ok {
		backingStore.close = closable.Close
	}

	return &backingStore
}

func (s *BackingStore) Connect() error {
	if s.opened {
		return nil
	}
	if s.connect != nil {
		s.opened = true
		return s.connect()
	}
	return nil
}

func (s *BackingStore) Close() error {
	if s.close != nil {
		s.opened = false
		return s.close()
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
	if s.shouldAutoConnect() {
		defer s.autoClose()
		if err := s.Connect(); err != nil {
			return nil, err
		}
	}

	return s.backingStore.List(itemType)
}

func (s *BackingStore) Save(itemType string, name string, data []byte) error {
	if s.shouldAutoConnect() {
		defer s.autoClose()
		if err := s.Connect(); err != nil {
			return err
		}
	}

	return s.backingStore.Save(itemType, name, data)
}

func (s *BackingStore) Read(itemType string, name string) ([]byte, error) {
	if s.shouldAutoConnect() {
		defer s.autoClose()
		if err := s.Connect(); err != nil {
			return nil, err
		}
	}

	return s.backingStore.Read(itemType, name)
}

// ReadAll retrieves all the items.
func (s *BackingStore) ReadAll(itemType string) ([][]byte, error) {
	if s.shouldAutoConnect() {
		defer s.autoClose()
		if err := s.Connect(); err != nil {
			return nil, err
		}
	}

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
	if s.shouldAutoConnect() {
		defer s.autoClose()
		if err := s.Connect(); err != nil {
			return err
		}
	}

	return s.backingStore.Delete(itemType, name)
}

func (s *BackingStore) shouldAutoConnect() bool {
	// If the connection is already open, let the upstream
	// caller manage the connection.
	return !s.opened && s.connect != nil
}
