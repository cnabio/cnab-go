package definition

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/qri-io/jsonpointer"
	"github.com/qri-io/jsonschema"
)

// ContentEncoding represents a "custom" Schema property
type ContentEncoding string

// NewContentEncoding allocates a new ContentEncoding validator
func NewContentEncoding() jsonschema.Keyword {
	return new(ContentEncoding)
}

func (c ContentEncoding) Validate(propPath string, data interface{}, errs *[]jsonschema.KeyError) {}

func (c ContentEncoding) ValidateKeyword(ctx context.Context, currentState *jsonschema.ValidationState, data interface{}) {
	if obj, ok := data.(string); ok {
		switch c {
		case "base64":
			_, err := base64.StdEncoding.DecodeString(obj)
			if err != nil {
				currentState.AddError(data, fmt.Sprintf("invalid %s value: %s", c, obj))
			}
		// Add validation support for other encodings as needed
		// See https://json-schema.org/latest/json-schema-validation.html#rfc.section.8.3
		default:
			currentState.AddError(data, fmt.Sprintf("unsupported or invalid contentEncoding type of %s", c))
		}
	}
}

func (c ContentEncoding) Register(uri string, registry *jsonschema.SchemaRegistry) {}

func (c ContentEncoding) Resolve(pointer jsonpointer.Pointer, uri string) *jsonschema.Schema {
	return nil
}

// NewRootSchema returns a jsonschema.RootSchema with any needed custom
// jsonschema.Validators pre-registered
func NewRootSchema() *jsonschema.Schema {
	// Register custom validators here
	// Note: as of writing, jsonschema doesn't have a stock validator for instances of type `contentEncoding`
	// There may be others missing in the library that exist in http://json-schema.org/draft-07/schema#
	// and thus, we'd need to create/register them here (if not included upstream)
	jsonschema.RegisterKeyword("contentEncoding", NewContentEncoding)
	jsonschema.LoadDraft2019_09()
	return &jsonschema.Schema{}
}
