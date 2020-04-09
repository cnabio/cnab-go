//go:generate packr2

package schema

import (
	"encoding/json"
	"fmt"

	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"github.com/qri-io/jsonschema"
)

// newSchemaBox returns a *packer.Box with the schema files from the schema sub-directory
func newSchemaBox() *packr.Box {
	return packr.New("github.com/cnabio/cnab-go/schema/schema", "./schema")
}

// ValidateBundle validates the provided bundle bytes against the applicable CNAB-Spec schema
func ValidateBundle(bytes []byte) ([]jsonschema.ValError, error) {
	return Validate("bundle", bytes)
}

// ValidateClaim validates the provided claim bytes against the applicable CNAB-Spec schema
func ValidateClaim(bytes []byte) ([]jsonschema.ValError, error) {
	return Validate("claim", bytes)
}

// Validate validates the provided bytes against the provided CNAB-Spec schemaType
func Validate(schemaType string, bytes []byte) ([]jsonschema.ValError, error) {
	schemaData, err := newSchemaBox().Find(fmt.Sprintf("%s.schema.json", schemaType))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read the schema data for type %q", schemaType)
	}

	rs := &jsonschema.RootSchema{}
	err = json.Unmarshal(schemaData, rs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to json.Unmarshal %q schema", schemaType)
	}

	err = rs.FetchRemoteReferences()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch remote references declared by %s schema", schemaType)
	}

	valErrors, err := rs.ValidateBytes(bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate %s", schemaType)
	}

	return valErrors, nil
}
