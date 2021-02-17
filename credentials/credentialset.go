package credentials

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/schema"
	"github.com/cnabio/cnab-go/secrets"
	"github.com/cnabio/cnab-go/valuesource"
)

const (
	// DefaultSchemaVersion is the default SchemaVersion value
	// set on new CredentialSet instances, and is the semver portion
	// of CNABSpecVersion.
	DefaultSchemaVersion = schema.Version("1.0.0-DRAFT+b6c701f")

	// CNABSpecVersion represents the CNAB Spec version of the Credentials
	// that this library implements
	// This value is prefixed with e.g. `cnab-credentials-` so isn't itself valid semver.
	CNABSpecVersion string = "cnab-credentialsets-" + string(DefaultSchemaVersion)
)

// CredentialSet represents a collection of credentials
type CredentialSet struct {
	// SchemaVersion is the version of the credential-set schema.
	SchemaVersion schema.Version `json:"schemaVersion" yaml:"schemaVersion"`
	// Name is the name of the credentialset.
	Name string `json:"name" yaml:"name"`
	// Created timestamp of the credentialset.
	Created time.Time `json:"created" yaml:"created"`
	// Modified timestamp of the credentialset.
	Modified time.Time `json:"modified" yaml:"modified"`
	// Credentials is a list of credential specs.
	Credentials []valuesource.Strategy `json:"credentials" yaml:"credentials"`
}

// NewCredentialSet creates a new CredentialSet with the required fields initialized.
func NewCredentialSet(name string, creds ...valuesource.Strategy) CredentialSet {
	now := time.Now()
	cs := CredentialSet{
		SchemaVersion: DefaultSchemaVersion,
		Name:          name,
		Created:       now,
		Modified:      now,
		Credentials:   creds,
	}

	return cs
}

// Load a CredentialSet from a file at a given path.
//
// It does not load the individual credentials.
func Load(path string) (*CredentialSet, error) {
	cset := &CredentialSet{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return cset, err
	}
	return cset, yaml.Unmarshal(data, cset)
}

// Validate compares the given credentials with the spec.
//
// This will result in an error only when the following conditions are true:
// - a credential in the spec is not present in the given set
// - the credential is required
// - the credential applies to the specified action
//
// It is allowed for spec to specify both an env var and a file. In such case, if
// the given set provides either, it will be considered valid.
func Validate(given valuesource.Set, spec map[string]bundle.Credential, action string) error {
	for name, cred := range spec {
		if !cred.AppliesTo(action) {
			continue
		}

		if !valuesource.IsValid(given, name) && cred.Required {
			return fmt.Errorf("bundle requires credential for %s", name)
		}
	}
	return nil
}

// ResolveCredentials looks up the credentials as described in Source, then copies
// the resulting value into the Value field of each credential strategy.
//
// The typical workflow for working with a credential set is:
//
//	- Load the set
//	- Validate the credentials against a spec
//	- Resolve the credentials
//	- Expand them into bundle values
func (c *CredentialSet) ResolveCredentials(s secrets.Store) (valuesource.Set, error) {
	l := len(c.Credentials)
	res := make(map[string]string, l)
	for i := 0; i < l; i++ {
		cred := c.Credentials[i]
		val, err := s.Resolve(cred.Source.Key, cred.Source.Value)
		if err != nil {
			return nil, fmt.Errorf("credential %q: %v", c.Credentials[i].Name, err)
		}
		cred.Value = val
		res[c.Credentials[i].Name] = cred.Value
	}
	return res, nil
}
