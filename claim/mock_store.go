package claim

import (
	"github.com/cnabio/cnab-go/utils/crud"
)

// NewMockStore creates a mock claim store for unit testing.
func NewMockStore(encrypt EncryptionHandler, decrypt EncryptionHandler) Store {
	return NewClaimStore(crud.NewBackingStore(crud.NewMockStore()), encrypt, decrypt)
}
