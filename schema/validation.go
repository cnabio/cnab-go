package schema

import (
	"embed"
	"fmt"

	"github.com/pkg/errors"

	// Using this library as qri-io/jsonschema doesn't appear to have
	// first-class support for adding auxiliary/ref'd sub-schemas,
	// apart from fetching remote references over the network
	// (which doesn't support airgapped scenarios)
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schema
var schemas embed.FS

// ValidationError is a validation error as defined by this package
// As of now, it simply equates to a stock Golang error
type ValidationError error

// ValidateBundle validates the provided bundle bytes against the applicable CNAB-Spec schema
func ValidateBundle(bytes []byte) ([]ValidationError, error) {
	return Validate("bundle", bytes)
}

// ValidateClaim validates the provided claim bytes against the applicable CNAB-Spec schema
func ValidateClaim(bytes []byte) ([]ValidationError, error) {
	return Validate("claim", bytes)
}

// Validate validates the provided bytes against the provided CNAB-Spec schemaType
func Validate(schemaType string, bytes []byte) ([]ValidationError, error) {
	valErrs := []ValidationError{}

	// Retrieve main schema bytes
	schemaData, err := schemas.ReadFile(fmt.Sprintf("schema/%s.schema.json", schemaType))
	if err != nil {
		return valErrs, errors.Wrapf(err, "failed to read the schema data for type %q", schemaType)
	}

	// Build schema validator
	sl := gojsonschema.NewSchemaLoader()

	// Now add main schema and compile
	schemaLoader := gojsonschema.NewBytesLoader(schemaData)
	schema, err := sl.Compile(schemaLoader)
	if err != nil {
		return valErrs, errors.Wrapf(err, "unable to compile schema validator for schema\n%s", string(schemaData))
	}

	// Validate the provided bytes via the compiled schema validator
	bytesLoader := gojsonschema.NewBytesLoader(bytes)
	result, err := schema.Validate(bytesLoader)
	if err != nil {
		return valErrs, errors.Wrap(err, "unable to validate provided data")
	}

	// Collect validation errors
	for _, desc := range result.Errors() {
		valErrs = append(valErrs, errors.New(desc.String()))
	}

	return valErrs, nil
}
