package credentials

import (
	"errors"
	"testing"

	"github.com/cnabio/cnab-go/utils/crud"
	"github.com/stretchr/testify/assert"
)

func TestCredentialStore_HandlesNotFoundError(t *testing.T) {
	mockStore := crud.NewMockStore()
	mockStore.ReadMock = func(itemType string, name string) (bytes []byte, err error) {
		// Change the default error message to test that we are checking
		// inside the error message and not matching it exactly
		return nil, errors.New("wrapping error message: " + crud.ErrRecordDoesNotExist.Error())
	}
	cs := NewCredentialStore(mockStore)

	_, err := cs.Read("missing cred set")
	assert.EqualError(t, err, ErrNotFound.Error())
}
