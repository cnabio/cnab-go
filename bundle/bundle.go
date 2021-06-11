package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	cjson "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	pkgErrors "github.com/pkg/errors"

	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/schema"
)

// CNABSpecVersion represents the CNAB Spec version of the Bundle
// that this library implements
// This value is prefixed with e.g. `cnab-core-` so isn't itself valid semver.
var CNABSpecVersion string = "cnab-core-1.2.0"

// Bundle is a CNAB metadata document
type Bundle struct {
	SchemaVersion      schema.Version         `json:"schemaVersion" yaml:"schemaVersion"`
	Name               string                 `json:"name" yaml:"name"`
	Version            string                 `json:"version" yaml:"version"`
	Description        string                 `json:"description" yaml:"description"`
	Keywords           []string               `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Maintainers        []Maintainer           `json:"maintainers,omitempty" yaml:"maintainers,omitempty"`
	InvocationImages   []InvocationImage      `json:"invocationImages" yaml:"invocationImages"`
	Images             map[string]Image       `json:"images,omitempty" yaml:"images,omitempty"`
	Actions            map[string]Action      `json:"actions,omitempty" yaml:"actions,omitempty"`
	Parameters         map[string]Parameter   `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Credentials        map[string]Credential  `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	Outputs            map[string]Output      `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	Definitions        definition.Definitions `json:"definitions,omitempty" yaml:"definitions,omitempty"`
	License            string                 `json:"license,omitempty" yaml:"license,omitempty"`
	RequiredExtensions []string               `json:"requiredExtensions,omitempty" yaml:"requiredExtensions,omitempty"`

	// Custom extension metadata is a named collection of auxiliary data whose
	// meaning is defined outside of the CNAB specification.
	Custom map[string]interface{} `json:"custom,omitempty" yaml:"custom,omitempty"`
}

// GetDefaultSchemaVersion returns the default semver CNAB schema version of the Bundle
// that this library implements
func GetDefaultSchemaVersion() (schema.Version, error) {
	ver, err := schema.GetSemver(CNABSpecVersion)
	if err != nil {
		return "", err
	}
	return ver, nil
}

// Marshal the bundle to canonical json.
func (b Bundle) Marshal() ([]byte, error) {
	// First marshal to json, then convert that to canonical json
	d, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	return cjson.Transform(d)
}

// Unmarshal a Bundle from json.
func Unmarshal(data []byte) (*Bundle, error) {
	b := &Bundle{}
	return b, json.Unmarshal(data, b)
}

// ParseReader reads CNAB metadata from a JSON string
func ParseReader(r io.Reader) (Bundle, error) {
	b := Bundle{}
	err := json.NewDecoder(r).Decode(&b)
	return b, err
}

// WriteFile serializes the bundle and writes it to a file as JSON.
func (b Bundle) WriteFile(dest string, mode os.FileMode) error {
	d, err := b.Marshal()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, d, mode)
}

// WriteTo writes unsigned JSON to an io.Writer using the standard formatting.
func (b Bundle) WriteTo(w io.Writer) (int64, error) {
	d, err := b.Marshal()
	if err != nil {
		return 0, err
	}
	l, err := w.Write(d)
	return int64(l), err
}

// LocationRef specifies a location within the invocation package
type LocationRef struct {
	Path      string `json:"path" yaml:"path"`
	Field     string `json:"field" yaml:"field"`
	MediaType string `json:"mediaType" yaml:"mediaType"`
}

// BaseImage contains fields shared across image types
type BaseImage struct {
	ImageType string            `json:"imageType" yaml:"imageType"`
	Image     string            `json:"image" yaml:"image"`
	Digest    string            `json:"contentDigest,omitempty" yaml:"contentDigest,omitempty"`
	Size      uint64            `json:"size,omitempty" yaml:"size,omitempty"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	MediaType string            `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
}

func (i *BaseImage) DeepCopy() *BaseImage {
	i2 := *i
	i2.Labels = make(map[string]string, len(i.Labels))
	for key, value := range i.Labels {
		i2.Labels[key] = value
	}
	return &i2
}

// Image describes a container image in the bundle
type Image struct {
	BaseImage   `yaml:",inline"`
	Description string `json:"description" yaml:"description"` //TODO: change? see where it's being used? change to description?
}

func (i *Image) DeepCopy() *Image {
	i2 := *i
	i2.BaseImage = *i.BaseImage.DeepCopy()
	return &i2
}

// InvocationImage contains the image type and location for the installation of a bundle
type InvocationImage struct {
	BaseImage `yaml:",inline"`
}

func (img *InvocationImage) DeepCopy() *InvocationImage {
	img2 := *img
	img2.BaseImage = *img.BaseImage.DeepCopy()
	return &img2
}

// Validate the image contents.
func (img InvocationImage) Validate() error {
	switch img.ImageType {
	case "docker", "oci":
		return validateDockerish(img.Image)
	default:
		return nil
	}
}

// Location provides the location where a value should be written in
// the invocation image.
//
// A location may be either a file (by path) or an environment variable.
type Location struct {
	Path                string `json:"path,omitempty" yaml:"path,omitempty"`
	EnvironmentVariable string `json:"env,omitempty" yaml:"env,omitempty"`
}

// Validate the Location
func (l Location) Validate() error {
	forbiddenPath := "/cnab/app/outputs"
	if strings.HasPrefix(l.Path, forbiddenPath) {
		return fmt.Errorf("Path %q must not be a subpath of %q", l.Path, forbiddenPath)
	}
	return nil
}

// Maintainer describes a code maintainer of a bundle
type Maintainer struct {
	// Name is a user name or organization name
	Name string `json:"name" yaml:"name"`
	// Email is an optional email address to contact the named maintainer
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
	// Url is an optional URL to an address for the named maintainer
	URL string `json:"url,omitempty" yaml:"url,omitempty"`
}

// Action describes a custom (non-core) action.
type Action struct {
	// Modifies indicates whether this action modifies the release.
	//
	// If it is possible that an action modify a release, this must be set to true.
	Modifies bool `json:"modifies,omitempty" yaml:"modifies,omitempty"`
	// Stateless indicates that the action is purely informational, that credentials are not required, and that the runtime should not keep track of its invocation
	Stateless bool `json:"stateless,omitempty" yaml:"stateless,omitempty"`
	// Description describes the action as a user-readable string
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// ValuesOrDefaults returns parameter values or the default parameter values. An error is returned when the parameter value does not pass
// the schema validation or a required parameter is missing, assuming the parameter applies to the provided action.
func ValuesOrDefaults(vals map[string]interface{}, b *Bundle, action string) (map[string]interface{}, error) {
	res := map[string]interface{}{}

	for name, param := range b.Parameters {
		// If the parameter doesn't apply to the provided action,
		// skip validation and do not attempt to include in the returned list
		if !param.AppliesTo(action) {
			continue
		}
		s, ok := b.Definitions[param.Definition]
		if !ok {
			return res, fmt.Errorf("unable to find definition for %s", name)
		}

		// Set to the corresponding val if it exists in the supplied overrides,
		// else error out if required or set to the default defined on the parameter
		var uncoerced interface{}
		if val, ok := vals[name]; ok {
			uncoerced = val
		} else if param.Required {
			return res, fmt.Errorf("parameter %q is required", name)
		} else {
			uncoerced = s.Default
		}

		// Only collect defaults and specified parameters. Unspecified optional parameters without defaults should not be validated.
		if param.Required || uncoerced != nil {
			// Validate the selection
			valErrs, err := s.Validate(uncoerced)
			if err != nil {
				return res, pkgErrors.Wrapf(err, "encountered an error validating parameter %s", name)
			}
			// This interface returns a single error. Validation can have multiple errors. For now return the first
			// We should update this later.
			if len(valErrs) > 0 {
				valErr := valErrs[0]
				return res, fmt.Errorf("cannot use value: %v as parameter %s: %s", uncoerced, name, valErr.Error)
			}
		}

		// Per the spec: "If no value is provided and default is unset, the runtime MUST set the value to an empty string (""), regardless of type",
		// that is why this line is called for every parameter.
		res[name] = s.CoerceValue(uncoerced)
	}
	return res, nil
}

// Validate the bundle contents.
func (b Bundle) Validate() error {
	err := b.SchemaVersion.Validate()
	if err != nil {
		return pkgErrors.Wrapf(err, "bundle validation failed")
	}

	if len(b.InvocationImages) == 0 {
		return errors.New("at least one invocation image must be defined in the bundle")
	}

	if b.Version == "latest" {
		return errors.New("'latest' is not a valid bundle version")
	}

	reqExt := make(map[string]bool, len(b.RequiredExtensions))
	for _, requiredExtension := range b.RequiredExtensions {
		// Verify the custom extension declared as required exists
		if _, exists := b.Custom[requiredExtension]; !exists {
			return fmt.Errorf("required extension '%s' is not defined in the Custom section of the bundle", requiredExtension)
		}

		// Check for duplicate entries
		if _, exists := reqExt[requiredExtension]; exists {
			return fmt.Errorf("required extension '%s' is already declared", requiredExtension)
		}

		// Populate map with required extension, for duplicate check above
		reqExt[requiredExtension] = true
	}

	// Validate the invocation images
	for _, img := range b.InvocationImages {
		err := img.Validate()
		if err != nil {
			return err
		}
	}

	// Validate the parameters
	for name, param := range b.Parameters {
		err := param.Validate(name, b)
		if err != nil {
			return pkgErrors.Wrapf(err, "validation failed for parameter %q", name)
		}
	}

	// Validate the credentials
	for name, cred := range b.Credentials {
		err := cred.Validate()
		if err != nil {
			return pkgErrors.Wrapf(err, "validation failed for credential %q", name)

		}
	}

	// Validate the outputs
	for name, output := range b.Outputs {
		err := output.Validate(name, b)
		if err != nil {
			return pkgErrors.Wrapf(err, "validation failed for output %q", name)
		}
	}

	return nil
}

func validateDockerish(s string) error {
	if !strings.Contains(s, ":") {
		return errors.New("tag is required")
	}
	return nil
}
