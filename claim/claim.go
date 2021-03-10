package claim

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/oklog/ulid"
	"github.com/pkg/errors"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/schema"
)

// CNABSpecVersion represents the CNAB Spec version of the Claim
// that this library implements
// This value is prefixed with e.g. `cnab-claim-` so isn't itself valid semver.
var CNABSpecVersion string = "cnab-claim-1.0.0-DRAFT+b5ed2f3"

// Status constants define the CNAB status fields on a Result.
const (
	StatusSucceeded = "succeeded"
	StatusCanceled  = "canceled"
	StatusFailed    = "failed"
	StatusRunning   = "running"
	StatusPending   = "pending"
	StatusUnknown   = "unknown"

	// Deprecated: StatusSuccess has been replaced by StatusSucceeded.
	StatusSuccess = StatusSucceeded

	// Deprecated: StatusFailure has been replaced by StatusFailed.
	StatusFailure = StatusFailed
)

// Action constants define the CNAB action to be taken
const (
	ActionInstall   = "install"
	ActionUpgrade   = "upgrade"
	ActionUninstall = "uninstall"
	ActionUnknown   = "unknown"
)

// Output constants define metadata about outputs that may be stored on a claim
// Result.
const (
	// OutputContentDigest is the output metadata key for the output's content digest.
	OutputContentDigest = "contentDigest"

	// OutputGeneratedByBundle is the output metadata key for if the output was
	// defined by the bundle and the value was set by the invocation image. Some
	// outputs are created by the executing driver or CNAB tool.
	OutputGeneratedByBundle = "generatedByBundle"

	// OutputInvocationImageLogs is a well-known output name used to store the logs from the invocation image.
	OutputInvocationImageLogs = "io.cnab.outputs.invocationImageLogs"
)

var (
	builtinActions = map[string]struct{}{"install": {}, "uninstall": {}, "upgrade": {}}
)

// Claim is an installation claim receipt.
//
// Claims represent information about a particular installation, and
// provide the necessary data to upgrade and uninstall
// a CNAB package.
type Claim struct {
	// SchemaVersion is the version of the claim schema.
	SchemaVersion schema.Version `json:"schemaVersion"`

	// Id of the claim.
	ID string `json:"id"`

	// Installation name.
	Installation string `json:"installation"`

	// Revision of the installation.
	Revision string `json:"revision"`

	// Created timestamp of the claim.
	Created time.Time `json:"created"`

	// Action executed against the installation.
	Action string `json:"action"`

	// Bundle is the definition of the bundle.
	Bundle bundle.Bundle `json:"bundle"`

	// BundleReference is the canonical reference to the bundle used in the action.
	BundleReference string `json:"bundleReference,omitempty"`

	// Parameters are the key/value pairs that were passed in during the operation.
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// Custom extension data applicable to a given runtime.
	Custom interface{} `json:"custom,omitempty"`

	// Results of executing the Claim's operation.
	// These are not stored in the Claim document but can be loaded onto the
	// the Claim to build an in-memory hierarchy.
	results *Results
}

// GetDefaultSchemaVersion returns the default semver CNAB schema version of the Claim
// that this library implements
func GetDefaultSchemaVersion() (schema.Version, error) {
	ver, err := schema.GetSemver(CNABSpecVersion)
	if err != nil {
		return "", err
	}
	return ver, nil
}

// ValidName is a regular expression that indicates whether a name is a valid claim name.
var ValidName = regexp.MustCompile("^[a-zA-Z0-9._-]+$")

// New creates a new Claim initialized for an operation.
func New(installation string, action string, bun bundle.Bundle, parameters map[string]interface{}) (Claim, error) {
	if !ValidName.MatchString(installation) {
		return Claim{}, fmt.Errorf("invalid installation name %q. Names must be [a-zA-Z0-9-_]+", installation)
	}

	schemaVersion, err := GetDefaultSchemaVersion()
	if err != nil {
		return Claim{}, err
	}

	now := time.Now()
	id, err := NewULID()
	if err != nil {
		return Claim{}, err
	}
	revision, err := NewULID()
	if err != nil {
		return Claim{}, err
	}

	return Claim{
		SchemaVersion: schemaVersion,
		ID:            id,
		Installation:  installation,
		Revision:      revision,
		Created:       now,
		Action:        action,
		Bundle:        bun,
		Parameters:    parameters,
	}, nil
}

// NewClaim is a convenience for creating a new claim from an existing claim.
func (c Claim) NewClaim(action string, bun bundle.Bundle, parameters map[string]interface{}) (Claim, error) {
	updatedClaim := c
	updatedClaim.Bundle = bun
	updatedClaim.Action = action
	updatedClaim.Parameters = parameters
	updatedClaim.Created = time.Now()

	id, err := NewULID()
	if err != nil {
		return Claim{}, err
	}
	updatedClaim.ID = id

	modifies, err := updatedClaim.IsModifyingAction()
	if err != nil {
		return Claim{}, err
	}

	if modifies {
		rev, err := NewULID()
		if err != nil {
			return Claim{}, err
		}
		updatedClaim.Revision = rev
	}

	return updatedClaim, nil
}

// IsModifyingAction determines if the Claim's action modifies the bundle.
// Non-modifying actions are not required to be persisted by the Claims spec.
func (c Claim) IsModifyingAction() (bool, error) {
	switch c.Action {
	case ActionInstall, ActionUpgrade, ActionUninstall:
		return true, nil
	default:
		actionDef, ok := c.Bundle.Actions[c.Action]
		if !ok {
			return false, fmt.Errorf("custom action not defined %q", c.Action)
		}

		return actionDef.Modifies, nil
	}
}

// NewResult is a convenience for creating a result with the necessary fields
// set on a Result.
func (c Claim) NewResult(status string) (Result, error) {
	return NewResult(c, status)
}

// Validate the Claim
func (c Claim) Validate() error {
	// validate the schemaVersion
	err := c.SchemaVersion.Validate()
	if err != nil {
		return errors.Wrap(err, "claim validation failed")
	}

	if c.ID == "" {
		return errors.New("the claim id must be set")
	}

	if c.Revision == "" {
		return errors.New("the revision must be set")
	}

	if c.Installation == "" {
		return errors.New("the installation must be set")
	}

	if c.Action == "" {
		return errors.New("the action must be set")
	}

	// Check the action is built-in or defined as a custom action
	if _, isBuiltInAction := builtinActions[c.Action]; !isBuiltInAction {
		_, isCustomAction := c.Bundle.Actions[c.Action]
		if !isCustomAction {
			return fmt.Errorf("action %q is not defined in the bundle", c.Action)
		}
	}

	return nil
}

// GetLastResult returns the most recent (last) result associated with the
// claim.
func (c Claim) GetLastResult() (Result, error) {
	if c.results == nil {
		return Result{}, errors.New("the claim does not have results loaded")
	}

	results := *c.results
	if len(results) == 0 {
		return Result{}, errors.New("the claim has no results")
	}

	sort.Sort(results)
	return results[len(results)-1], nil
}

// GetStatus returns the status of the claim using the last result.
func (c Claim) GetStatus() string {
	result, err := c.GetLastResult()
	if err != nil {
		return StatusUnknown
	}

	return result.Status
}

// HasLogs indicates if logs were persisted for the bundle action.
// When ok is false, this indicates that the result is indeterminate
// because results are not loaded on the claim.
func (c Claim) HasLogs() (hasLogs bool, ok bool) {
	if c.results == nil {
		return false, false
	}

	for _, r := range *c.results {
		if r.HasLogs() {
			return true, true
		}
	}

	return false, true
}

type Claims []Claim

func (c Claims) Len() int {
	return len(c)
}

func (c Claims) Less(i, j int) bool {
	return c[i].ID < c[j].ID
}

func (c Claims) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// ulidMutex guards the generation of ULIDs, because the use of rand
// is not thread-safe.
var ulidMutex sync.Mutex

// ulidEntropy must be set once and reused when generating ULIDs, to guarantee
// that each ULID is monotonically increasing.
var ulidEntropy = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

// MustNewULID generates a string representation of a ULID and panics on failure
// instead of returning an error.
func MustNewULID() string {
	result, err := NewULID()
	if err != nil {
		panic(err)
	}
	return result
}

// NewULID generates a string representation of a ULID.
func NewULID() (string, error) {
	ulidMutex.Lock()
	defer ulidMutex.Unlock()
	result, err := ulid.New(ulid.Timestamp(time.Now()), ulidEntropy)
	if err != nil {
		return "", errors.Wrap(err, "could not generate a new ULID")
	}
	return result.String(), nil
}
