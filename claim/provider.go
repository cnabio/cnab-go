package claim

// Provider interface for claim data.
type Provider interface {
	// ListInstallations returns Installation names sorted in ascending order.
	ListInstallations() ([]string, error)

	// ListClaims returns Claim IDs associated with an Installation sorted in ascending order.
	ListClaims(installation string) ([]string, error)

	// ListResults returns Result IDs associated with a Claim, sorted in ascending order.
	ListResults(claimID string) ([]string, error)

	// ListOutputs returns the names of outputs associated with a result that
	// have been persisted. It is possible for results to have outputs that were
	// generated but not persisted see
	// https://github.com/cnabio/cnab-spec/blob/cnab-claim-1.0.0-DRAFT+b5ed2f3/400-claims.md#outputs
	// for more information.
	ListOutputs(resultID string) ([]string, error)

	// ReadInstallation returns the specified Installation with all Claims and their Results loaded.
	ReadInstallation(installation string) (Installation, error)

	// ReadInstallationStatus returns the specified Installation with the last Claim and its last Result loaded.
	ReadInstallationStatus(installation string) (Installation, error)

	// ReadAllInstallationStatus returns all Installations with the last Claim and its last Result loaded.
	ReadAllInstallationStatus() ([]Installation, error)

	// ReadClaim returns the specified Claim.
	ReadClaim(claimID string) (Claim, error)

	// ReadAllClaims returns all claims associated with an Installation, sorted in ascending order.
	ReadAllClaims(installation string) ([]Claim, error)

	// ReadLastClaim returns the last claim associated with an Installation.
	ReadLastClaim(installation string) (Claim, error)

	// ReadResult returns the specified Result.
	ReadResult(resultID string) (Result, error)

	// ReadAllResult returns all results associated with a Claim, sorted in ascending order.
	ReadAllResults(claimID string) ([]Result, error)

	// ReadLastResult returns the last result associated with a Claim.
	ReadLastResult(claimID string) (Result, error)

	// ReadAllOutputs returns the most recent (last) value of each Output associated
	// with the installation.
	ReadLastOutputs(installation string) (Outputs, error)

	// ReadLastOutput returns the most recent value (last) of the specified Output associated
	// with the installation.
	ReadLastOutput(installation string, name string) (Output, error)

	// ReadOutput returns the contents of the named output associated with the specified Result.
	ReadOutput(claim Claim, result Result, outputName string) (Output, error)

	// SaveClaim persists the specified claim.
	// Associated results, Claim.Results, must be persisted separately with SaveResult.
	SaveClaim(claim Claim) error

	// SaveResult persists the specified result.
	SaveResult(result Result) error

	// SaveOutput persists the output, encrypting the value if defined as
	// sensitive (write-only) in the bundle.
	SaveOutput(output Output) error

	// DeleteInstallation removes all data associated with the specified installation.
	DeleteInstallation(installation string) error

	// DeleteClaim removes all data associated with the specified claim.
	DeleteClaim(claimID string) error

	// DeleteResult removes all data associated with the specified result.
	DeleteResult(resultID string) error

	// DeleteOutput removes an output persisted with the specified result.
	DeleteOutput(resultID string, outputName string) error
}
