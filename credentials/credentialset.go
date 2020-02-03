package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/secrets"
	"gopkg.in/yaml.v2"
)

// Set is an actual set of resolved credentials.
// This is the output of resolving a credentialset file.
type Set map[string]string

// Expand expands the set into env vars and paths per the spec in the bundle.
//
// This matches the credentials required by the bundle to the credentials present
// in the credentialset, and then expands them per the definition in the Bundle.
func (s Set) Expand(b *bundle.Bundle, stateless bool) (env, files map[string]string, err error) {
	env, files = map[string]string{}, map[string]string{}
	for name, val := range b.Credentials {
		src, ok := s[name]
		if !ok {
			if stateless || !val.Required {
				continue
			}
			err = fmt.Errorf("credential %q is missing from the user-supplied credentials", name)
			return
		}
		if val.EnvironmentVariable != "" {
			env[val.EnvironmentVariable] = src
		}
		if val.Path != "" {
			files[val.Path] = src
		}
	}
	return
}

// Merge merges a second Set into the base.
//
// Duplicate credential names are not allow and will result in an
// error, this is the case even if the values are identical.
func (s Set) Merge(s2 Set) error {
	for k, v := range s2 {
		if _, ok := s[k]; ok {
			return fmt.Errorf("ambiguous credential resolution: %q is already present in base credential sets, cannot merge", k)
		}
		s[k] = v
	}
	return nil
}

// CredentialSet represents a collection of credentials
type CredentialSet struct {
	// Name is the name of the credentialset.
	Name string `json:"name" yaml:"name"`
	// Created timestamp of the credentialset.
	Created time.Time `json:"created" yaml:"created"`
	// Modified timestamp of the credentialset.
	Modified time.Time `json:"modified" yaml:"modified"`
	// Credentials is a list of credential specs.
	Credentials []CredentialStrategy `json:"credentials" yaml:"credentials"`
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
//
// It is allowed for spec to specify both an env var and a file. In such case, if
// the given set provides either, it will be considered valid.
func Validate(given Set, spec map[string]bundle.Credential) error {
	for name, cred := range spec {
		if !isValidCred(given, name) && cred.Required {
			return fmt.Errorf("bundle requires credential for %s", name)
		}
	}
	return nil
}

func isValidCred(haystack Set, needle string) bool {
	for name := range haystack {
		if name == needle {
			return true
		}
	}
	return false
}

// Resolve looks up the credentials as described in Source, then copies
// the resulting value into the Value field of each credential strategy.
//
// The typical workflow for working with a credential set is:
//
//	- Load the set
//	- Validate the credentials against a spec
//	- Resolve the credentials
//	- Expand them into bundle values
func (c *CredentialSet) ResolveCredentials(s secrets.Store) (Set, error) {
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

// CredentialStrategy represents a source credential and the destination to which it should be sent.
type CredentialStrategy struct {
	// Name is the name of the credential.
	// Name is used to match a credential strategy to a bundle's credential.
	Name string `json:"name" yaml:"name"`
	// Source is the location of the credential.
	// During resolution, the source will be loaded, and the result temporarily placed
	// into Value.
	Source Source `json:"source,omitempty" yaml:"source,omitempty"`
	// Value holds the credential value.
	// When a credential is loaded, it is loaded into this field. In all
	// other cases, it is empty. This field is omitted during serialization.
	Value string `json:"-" yaml:"-"`
}

// Source represents a strategy for loading a credential from local host.
type Source struct {
	Key   string
	Value string
}

func (s *Source) marshalRaw() interface{} {
	if s.Key == "" {
		return nil
	}
	return map[string]string{s.Key: s.Value}
}

func (s *Source) unmarshalRaw(raw map[string]string) error {
	switch len(raw) {
	case 0:
		s.Key = ""
		s.Value = ""
		return nil
	case 1:
		for k, v := range raw {
			s.Key = k
			s.Value = v
		}
		return nil
	default:
		return errors.New("multiple key/value pairs specified for source but only one may be defined")
	}
}

func (s Source) MarshalJSON() ([]byte, error) {
	raw := s.marshalRaw()
	return json.Marshal(raw)
}

func (s *Source) UnmarshalJSON(data []byte) error {
	var raw map[string]string
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	return s.unmarshalRaw(raw)
}

func (s Source) MarshalYAML() (interface{}, error) {
	// TODO: use https://github.com/ghodss/yaml so that we don't need json and yaml defined
	return s.marshalRaw(), nil
}

func (s *Source) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]string
	err := unmarshal(&raw)
	if err != nil {
		return err
	}
	return s.unmarshalRaw(raw)
}

// Destination represents a strategy for injecting a credential into an image.
type Destination struct {
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}
