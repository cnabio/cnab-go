package credentials

import (
	"github.com/cnabio/cnab-go/utils/crud"
)

// NewMockStore creates a mock credentials store for unit testing.
func NewMockStore() Store {
	return NewCredentialStore(crud.NewBackingStore(crud.NewMockStore()))
}
