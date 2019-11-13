package crud

// Store is a simplified interface to a key-blob store supporting CRUD operations.
type Store interface {
	List(itemType string) ([]string, error)
	Save(itemType string, name string, data []byte) error
	Read(itemType string, name string) ([]byte, error)
	Delete(itemType string, name string) error
}
