package claim

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cnabio/cnab-go/utils/crud"
)

// ItemType is the location in the backing store where claims are persisted.
const ItemType = "claims"

// ErrClaimNotFound represents a claim not found in claim storage
var ErrClaimNotFound = errors.New("Claim does not exist")

// Store is a persistent store for claims.
type Store struct {
	backingStore *crud.BackingStore
}

// NewClaimStore creates a persistent store for claims using the specified
// backing key-blob store.
func NewClaimStore(store crud.Store) Store {
	return Store{
		backingStore: crud.NewBackingStore(store),
	}
}

// List lists the names of the stored claims.
func (s Store) List() ([]string, error) {
	return s.backingStore.List(ItemType)
}

// Save a claim. Any previous version of the claim (that is, with the same
// name) is overwritten.
func (s Store) Save(claim Claim) error {
	bytes, err := json.MarshalIndent(claim, "", "  ")
	if err != nil {
		return err
	}
	return s.backingStore.Save(ItemType, claim.Installation, bytes)
}

// Read loads the claim with the given name from the store.
func (s Store) Read(name string) (Claim, error) {
	bytes, err := s.backingStore.Read(ItemType, name)
	if err != nil {
		if err == crud.ErrRecordDoesNotExist {
			return Claim{}, ErrClaimNotFound
		}
		return Claim{}, err
	}
	claim := Claim{}
	err = json.Unmarshal(bytes, &claim)
	return claim, err
}

// ReadAll retrieves all of the claims.
func (s Store) ReadAll() ([]Claim, error) {
	results, err := s.backingStore.ReadAll(ItemType)
	if err != nil {
		return nil, err
	}

	claims := make([]Claim, len(results))
	for i, bytes := range results {
		var claim Claim
		err = json.Unmarshal(bytes, &claim)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling claim: %v", err)
		}
		claims[i] = claim
	}

	return claims, nil
}

// Delete deletes a claim from the store.
func (s Store) Delete(name string) error {
	return s.backingStore.Delete(ItemType, name)
}
