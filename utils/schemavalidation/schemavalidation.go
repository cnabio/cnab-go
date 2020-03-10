//go:generate packr2

package schemavalidation

import (
	"encoding/json"
	"fmt"

	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"github.com/qri-io/jsonschema"
)

// NewSchemaBox returns a *packer.Box with the schema files from the schema sub-directory
func NewSchemaBox() *packr.Box {
	return packr.New("github.com/cnabio/cnab-go/utils/schemavalidation/schema", "./schema")
}

// Validate validates the provided objectBytes against the provided CNAB-Spec schemaType
func Validate(schemaType string, objectBytes []byte) ([]jsonschema.ValError, error) {
	schemaData, err := NewSchemaBox().Find(fmt.Sprintf("%s.schema.json", schemaType))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read the schema data for type %q", schemaType)
	}

	rs := &jsonschema.RootSchema{}
	err = json.Unmarshal(schemaData, rs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to json.Unmarshal %s schema", schemaType)
	}

	err = rs.FetchRemoteReferences()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch remote references declared by %s schema", schemaType)
	}

	valErrors, err := rs.ValidateBytes(objectBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate %s", schemaType)
	}

	return valErrors, nil
}
