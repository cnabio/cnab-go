package claim

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/cnabio/cnab-go/utils/crud"
)

// ItemType is the location in the backing store where claims are persisted.
const (
	// Deprecated: ItemType has been replaced by ItemTypeClaims.
	ItemType = "claims"

	// ItemTypeClaims
	ItemTypeClaims  = "claims"
	ItemTypeResults = "results"
	ItemTypeOutputs = "outputs"
)

var (
	// ErrInstallationNotFound represents an installation not found in claim storage
	ErrInstallationNotFound = errors.New("Installation does not exist")

	// ErrClaimNotFound represents a claim not found in claim storage
	ErrClaimNotFound = errors.New("Claim does not exist")

	// ErrResultNotFound represents a result not found in claim storage
	ErrResultNotFound = errors.New("Result does not exist")

	// ErrOutputNotFound represents an output not found in claim storage
	ErrOutputNotFound = errors.New("Output does not exist")
)

// Store is a persistent store for claims.
type Store struct {
	backingStore *crud.BackingStore
	encrypt      EncryptionHandler
	decrypt      EncryptionHandler
}

// NewClaimStore creates a persistent store for claims using the specified
// backing key-blob store.
func NewClaimStore(store crud.Store, encrypt EncryptionHandler, decrypt EncryptionHandler) Store {
	if encrypt == nil {
		encrypt = noOpEncryptionHandler
	}

	if decrypt == nil {
		decrypt = noOpEncryptionHandler
	}

	return Store{
		backingStore: crud.NewBackingStore(store),
		encrypt:      encrypt,
		decrypt:      decrypt,
	}
}

// NewClaimStoreFileExtensions builds a FileExtensions map suitable for use
// with a crud.FileSystem for a ClaimStore.
func NewClaimStoreFileExtensions() map[string]string {
	const json = ".json"
	return map[string]string{
		ItemTypeClaims:  json,
		ItemTypeResults: json,
		ItemTypeOutputs: "",
	}
}

// EncryptionHandler is a function that transforms data by encrypting or decrypting it.
type EncryptionHandler func([]byte) ([]byte, error)

// noOpEncryptHandler is used when no handler is specified.
var noOpEncryptionHandler = func(data []byte) ([]byte, error) {
	return data, nil
}

func (s Store) ListInstallations() ([]string, error) {
	return s.backingStore.List(ItemTypeClaims, "")
}

func (s Store) ListClaims(installation string) ([]string, error) {
	return s.backingStore.List(ItemTypeClaims, installation)
}

func (s Store) ListResults(claimID string) ([]string, error) {
	return s.backingStore.List(ItemTypeResults, claimID)
}

func (s Store) ListOutputs(resultID string) ([]string, error) {
	outputNames, err := s.backingStore.List(ItemTypeOutputs, resultID)
	if err != nil {
		return nil, err
	}

	// outputs are keyed with the result, like RESULTID-OUTPUTNAME to make them unique
	// Strip off RESULTID- and return just OUTPUTNAME
	for i, fullName := range outputNames {
		outputNames[i] = strings.TrimLeft(fullName, resultID+"-")
	}

	return outputNames, nil
}

func (s Store) ReadInstallation(installation string) (Installation, error) {
	claims, err := s.ReadAllClaims(installation)
	if err != nil {
		return Installation{}, err
	}

	i := Installation{
		Name:   installation,
		Claims: claims,
	}
	return i, nil
}

func (s Store) ReadInstallationStatus(installation string) (Installation, error) {
	i := Installation{
		Name: installation,
	}

	claimIds, err := s.ListClaims(installation)
	if err != nil {
		return Installation{}, err
	}

	if len(claimIds) == 0 {
		return i, nil
	}

	sort.Strings(claimIds)
	lastClaimID := claimIds[len(claimIds)-1]
	c, err := s.ReadClaim(lastClaimID)
	if err != nil {
		return Installation{}, err
	}
	i.Claims = Claims{c}

	resultIDs, err := s.ListResults(lastClaimID)
	if err != nil {
		return Installation{}, err
	}

	if len(resultIDs) == 0 {
		return i, nil
	}

	sort.Strings(resultIDs)
	lastResultID := resultIDs[len(resultIDs)-1]
	r, err := s.ReadResult(lastResultID)
	if err != nil {
		return Installation{}, err
	}
	i.Claims[0].Results = Results{r}

	return i, nil
}

func (s Store) ReadAllInstallationStatus() ([]Installation, error) {
	names, err := s.ListInstallations()
	if err != nil {
		return nil, err
	}

	installations := make([]Installation, 0, len(names))
	for _, name := range names {
		installation, err := s.ReadInstallationStatus(name)
		if err != nil {
			// TODO: (carolynvs) for any of these ranges, return some results instead of nothing when one fails
			return nil, err
		}
		installations = append(installations, installation)
	}

	return installations, nil
}

func (s Store) ReadClaim(claimID string) (Claim, error) {
	bytes, err := s.backingStore.Read(ItemTypeClaims, claimID)
	if err != nil {
		if strings.Contains(err.Error(), crud.ErrRecordDoesNotExist.Error()) {
			return Claim{}, ErrClaimNotFound
		}
		return Claim{}, err
	}

	bytes, err = s.decrypt(bytes)
	if err != nil {
		return Claim{}, errors.Wrap(err, "error decrypting claim")
	}

	claim := Claim{}
	err = json.Unmarshal(bytes, &claim)
	return claim, err
}

func (s Store) ReadAllClaims(installation string) ([]Claim, error) {
	items, err := s.backingStore.ReadAll(ItemTypeClaims, installation)
	if err != nil {
		// TODO: handle installation not found
		return nil, err
	}

	claims := make(Claims, len(items))
	for i, bytes := range items {
		bytes, err = s.decrypt(bytes)
		if err != nil {
			return nil, errors.Wrap(err, "error decrypting claim")
		}

		var claim Claim
		err = json.Unmarshal(bytes, &claim)
		if err != nil {
			return nil, errors.Wrap(err, "error unmarshaling claim")
		}
		claims[i] = claim
	}

	sort.Sort(claims)
	return claims, nil
}

func (s Store) ReadLastClaim(installation string) (Claim, error) {
	claimIds, err := s.backingStore.List(ItemTypeClaims, installation)
	if err != nil {
		// TODO: handle installation not found
		return Claim{}, err
	}

	if len(claimIds) == 0 {
		return Claim{}, ErrClaimNotFound
	}

	sort.Strings(claimIds)
	lastClaimID := claimIds[len(claimIds)-1]

	return s.ReadClaim(lastClaimID)
}

func (s Store) ReadResult(resultID string) (Result, error) {
	bytes, err := s.backingStore.Read(ItemTypeResults, resultID)
	if err != nil {
		// TODO: handle installation/ claim / result not found
		if strings.Contains(err.Error(), crud.ErrRecordDoesNotExist.Error()) {
			return Result{}, ErrResultNotFound
		}
		return Result{}, err
	}
	result := Result{}
	err = json.Unmarshal(bytes, &result)
	return result, err
}

func (s Store) ReadAllResults(claimID string) ([]Result, error) {
	items, err := s.backingStore.ReadAll(ItemTypeResults, claimID)
	if err != nil {
		// TODO: handle claim not found
		return nil, err
	}

	results := make(Results, len(items))
	for i, bytes := range items {
		var result Result
		err = json.Unmarshal(bytes, &result)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling result: %v", err)
		}
		results[i] = result
	}

	sort.Sort(results)
	return results, nil
}

func (s Store) ReadLastResult(claimID string) (Result, error) {
	resultIDs, err := s.backingStore.List(ItemTypeResults, claimID)
	if err != nil {
		// TODO: handle installation/claim not found
		return Result{}, err
	}

	if len(resultIDs) == 0 {
		return Result{}, fmt.Errorf("claim %s has no results", claimID)
	}

	sort.Strings(resultIDs)
	lastResultID := resultIDs[len(resultIDs)-1]

	return s.ReadResult(lastResultID)
}

func (s Store) ReadOutput(claim Claim, result Result, outputName string) (Output, error) {
	bytes, err := s.backingStore.Read(ItemTypeOutputs, s.outputKey(result.ID, outputName))
	if err != nil {
		if strings.Contains(err.Error(), crud.ErrRecordDoesNotExist.Error()) {
			return Output{}, ErrOutputNotFound
		}
		return Output{}, err
	}

	sensitive, err := claim.Bundle.IsOutputSensitive(outputName)
	if err != nil {
		return Output{}, errors.Wrapf(err, "could not determine if the output %q is sensitive", outputName)
	}

	if sensitive {
		bytes, err = s.decrypt(bytes)
		if err != nil {
			return Output{}, errors.Wrap(err, "error decrypting output")
		}
	}

	return Output{
		Claim:  claim,
		Result: result,
		Name:   outputName,
		Value:  bytes,
	}, nil
}

func (s Store) SaveClaim(claim Claim) error {
	bytes, err := json.MarshalIndent(claim, "", "  ")
	if err != nil {
		return err
	}

	bytes, err = s.encrypt(bytes)
	if err != nil {
		return errors.Wrapf(err, "error encrypting claim %s of installation %s", claim.ID, claim.Installation)
	}

	return s.backingStore.Save(ItemTypeClaims, claim.Installation, claim.ID, bytes)
}

func (s Store) SaveResult(result Result) error {
	bytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return s.backingStore.Save(ItemTypeResults, result.ClaimID, result.ID, bytes)
}

func (s Store) SaveOutput(output Output) error {
	sensitive, err := output.Claim.Bundle.IsOutputSensitive(output.Name)
	if err != nil {
		return errors.Wrapf(err, "could not determine if the output %q is sensitive", output.Name)
	}

	data := output.Value
	if sensitive {
		data, err = s.encrypt(output.Value)
		if err != nil {
			// TODO (carolynvs) make all of the error messages provide context like this one
			return errors.Wrapf(err, "error encrypting output %s for result %s of installation %s", output.Name, output.Result.ID, output.Claim.Installation)
		}
	}

	return s.backingStore.Save(ItemTypeOutputs, output.Result.ID, s.outputKey(output.Result.ID, output.Name), data)
}

func (s Store) DeleteInstallation(installation string) error {
	claimIds, err := s.ListClaims(installation)
	if err != nil {
		return err
	}

	// Todo: go routines
	for _, claimID := range claimIds {
		err := s.DeleteClaim(claimID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s Store) DeleteClaim(claimID string) error {
	resultIds, err := s.ListResults(claimID)
	if err != nil {
		return err
	}

	// Todo: go routines
	for _, resultID := range resultIds {
		err := s.DeleteResult(resultID)
		if err != nil {
			return err
		}
	}

	return s.backingStore.Delete(ItemTypeClaims, claimID)
}

func (s Store) DeleteResult(resultID string) error {
	outputNames, err := s.ListOutputs(resultID)
	if err != nil {
		return err
	}

	// Todo: go routines
	for _, output := range outputNames {
		err := s.DeleteOutput(resultID, output)
		if err != nil {
			return err
		}
	}

	return s.backingStore.Delete(ItemTypeResults, resultID)
}

func (s Store) DeleteOutput(resultID string, outputName string) error {
	return s.backingStore.Delete(ItemTypeOutputs, s.outputKey(resultID, outputName))
}

// outputKey returns the full name of an Output suitable for storage.
// ResultId is used to create a unique name because output names are
// not unique across bundle executions.
func (s Store) outputKey(resultID string, output string) string {
	return resultID + "-" + output
}
