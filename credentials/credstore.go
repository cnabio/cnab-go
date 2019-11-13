package credentials

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cnabio/cnab-go/utils/crud"
)

const ItemType = "credentials"

// ErrNotFound represents a credential set not found in storage
var ErrNotFound = errors.New("Credential set does not exist")

// Store is a persistent store for
type Store struct {
	backingStore *crud.BackingStore
}

// NewCredentialStore creates a persistent store for credential sets using the specified
// backing key-blob store.
func NewCredentialStore(store crud.Store) Store {
	return Store{
		backingStore: crud.NewBackingStore(store),
	}
}

// List lists the names of the stored credential sets.
func (s Store) List() ([]string, error) {
	return s.backingStore.List()
}

// Save a credential set. Any previous version of the credential set is overwritten.
func (s Store) Save(cred CredentialSet) error {
	bytes, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	return s.backingStore.Save(cred.Name, bytes)
}

// Read loads the credential set with the given name from the store.
func (s Store) Read(name string) (CredentialSet, error) {
	bytes, err := s.backingStore.Read(name)
	if err != nil {
		if err == crud.ErrRecordDoesNotExist {
			return CredentialSet{}, ErrNotFound
		}
		return CredentialSet{}, err
	}
	credset := CredentialSet{}
	err = json.Unmarshal(bytes, &credset)
	return credset, err
}

// ReadAll retrieves all the credential sets.
func (s Store) ReadAll() ([]CredentialSet, error) {
	results, err := s.backingStore.ReadAll()
	if err != nil {
		return nil, err
	}

	claims := make([]CredentialSet, len(results))
	for i := range claims {
		cs := CredentialSet{}
		bytes := results[i]
		err = json.Unmarshal(bytes, &cs)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling credential set: %v", err)
		}
		claims[i] = cs
	}

	return claims, nil
}

// Delete deletes a credential set from the store.
func (s Store) Delete(name string) error {
	return s.backingStore.Delete(name)
}
