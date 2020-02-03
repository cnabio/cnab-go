package claim

import (
	"encoding/json"
	"errors"

	"github.com/cnabio/cnab-go/utils/crud"
)

const ItemType = "claims"

// ErrClaimNotFound represents a claim not found in claim storage
var ErrClaimNotFound = errors.New("Claim does not exist")

// Store is a persistent store for claims.
type Store struct {
	backingStore crud.Store
}

// NewClaimStore creates a persistent store for claims using the specified
// backing key-blob store.
func NewClaimStore(backingStore crud.Store) Store {
	return Store{
		backingStore: backingStore,
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
	return s.backingStore.Save(ItemType, claim.Name, bytes)
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

// ReadAll retrieves all the claims
func (s Store) ReadAll() ([]Claim, error) {
	claims := make([]Claim, 0)

	list, err := s.backingStore.List(ItemType)
	if err != nil {
		return claims, err
	}

	for _, c := range list {
		cl, err := s.Read(c)
		if err != nil {
			return claims, err
		}
		claims = append(claims, cl)
	}
	return claims, nil
}

// Delete deletes a claim from the store.
func (s Store) Delete(name string) error {
	return s.backingStore.Delete(ItemType, name)
}
