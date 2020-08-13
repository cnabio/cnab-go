package crud

// Store is a simplified interface to a key-blob store supporting CRUD operations.
type Store interface {
	List(itemType string, group string) ([]string, error)
	Save(itemType string, group string, name string, data []byte) error
	Read(itemType string, name string) ([]byte, error)
	Delete(itemType string, name string) error
}

// ManagedStore is a wrapped crud.Store with a managed connection lifecycle.
type ManagedStore interface {
	// Store is the underlying datastore.
	Store

	// ReadAll retrieves all the items with the specified prefix.
	ReadAll(itemType string, group string) ([][]byte, error)

	// GetDataStore returns the datastore managed by this instance.
	GetDataStore() Store

	// HandleConnect connects if necessary, returning a function to close the
	// connection. This close function may be a no-op when connection was
	// already established and this call to Connect isn't managing the
	// connection.
	HandleConnect() (func() error, error)
}

// HasConnect indicates that a struct must be initialized using the Connect
// method before the interface's methods are called.
type HasConnect interface {
	Connect() error
}

// HasClose indicates that a struct must be cleaned up using the Close
// method before the interface's methods are called.
type HasClose interface {
	Close() error
}
