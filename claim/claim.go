package claim

import (
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"github.com/oklog/ulid"
	"github.com/pkg/errors"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/utils/schemaversion"
)

// CNABSpecVersion represents the CNAB Spec version of the Claim
// Schema injected at build-time.
// This value is prefixed with e.g. `cnab-claim-` so isn't itself valid semver.
var CNABSpecVersion string

// CNABSchemaVersion represents the semver CNAB schema version of the Claim
// that this library implements
var CNABSchemaVersion = schemaversion.GetSemverSchemaVersion(CNABSpecVersion)

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

// Claim is an installation claim receipt.
//
// Claims represent information about a particular installation, and
// provide the necessary data to upgrade and uninstall
// a CNAB package.
type Claim struct {
	SchemaVersion   schemaversion.SchemaVersion `json:"schemaVersion"`
	Installation    string                      `json:"installation"`
	Revision        string                      `json:"revision"`
	Created         time.Time                   `json:"created"`
	Modified        time.Time                   `json:"modified"`
	Bundle          *bundle.Bundle              `json:"bundle"`
	BundleReference string                      `json:"bundleReference,omitempty"`
	Result          Result                      `json:"result,omitempty"`
	Parameters      map[string]interface{}      `json:"parameters,omitempty"`
	Outputs         map[string]interface{}      `json:"outputs,omitempty"`
	Custom          interface{}                 `json:"custom,omitempty"`
}

// ValidName is a regular expression that indicates whether a name is a valid claim name.
var ValidName = regexp.MustCompile("^[a-zA-Z0-9._-]+$")

// New creates a new Claim initialized for an installation operation.
func New(name string) (*Claim, error) {

	if !ValidName.MatchString(name) {
		return nil, fmt.Errorf("invalid installation name %q. Names must be [a-zA-Z0-9-_]+", name)
	}

	now := time.Now()
	return &Claim{
		SchemaVersion: CNABSchemaVersion,
		Installation:  name,
		Revision:      ULID(),
		Created:       now,
		Modified:      now,
		Result: Result{
			Action: ActionUnknown,
			Status: StatusUnknown,
		},
		Parameters: map[string]interface{}{},
		Outputs:    map[string]interface{}{},
	}, nil
}

// Update is a convenience for modifying the necessary fields on a Claim.
//
// Per spec, when a claim is updated, the action, status, revision, and modified fields all change.
// All but status and action can be computed.
func (c *Claim) Update(action, status string) {
	c.Result.Action = action
	c.Result.Status = status
	c.Modified = time.Now()
	c.Revision = ULID()
}

// Result tracks the result of an operation on a CNAB installation
type Result struct {
	Message string `json:"message,omitempty"`
	Action  string `json:"action"`
	Status  string `json:"status"`
}

// Validate the Result
func (r Result) Validate() error {
	if r.Action == "" {
		return errors.New("the action must be provided")
	}

	switch r.Status {
	case StatusCanceled, StatusFailed, StatusPending, StatusRunning, StatusSucceeded, StatusUnknown:
		return nil
	}
	return fmt.Errorf("invalid status: %s", r.Status)
}

// ULID generates a string representation of a ULID.
func ULID() string {
	now := time.Now()
	entropy := rand.New(rand.NewSource(now.UnixNano()))
	return ulid.MustNew(ulid.Timestamp(now), entropy).String()
}

// Validate the Claim
func (c Claim) Validate() error {
	// validate the schemaVersion
	err := c.SchemaVersion.Validate()
	if err != nil {
		return errors.Wrap(err, "claim validation failed")
	}

	// validate the Result
	return errors.Wrap(c.Result.Validate(), "claim validation failed")
}
