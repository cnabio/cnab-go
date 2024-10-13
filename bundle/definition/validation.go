package definition

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidationError error represents a validation error
// against the JSON Schema. The type includes the path
// in the given object and the error message
type ValidationError struct {
	Path  string
	Error string
}

// ValidateSchema validates that the Schema is valid JSON Schema.
// If no errors occur, the validated jsonschema.Schema is returned.
func (s *Schema) ValidateSchema() (*jsonschema.Schema, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load schema")
	}
	schema, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	c := NewCompiler()
	err = c.AddResource("schema.json", schema)
	if err != nil {
		return nil, err
	}
	rs, err := c.Compile("schema.json")
	if err != nil {
		return nil, err
	}
	return rs, nil
}

// Validate applies JSON Schema validation to the data passed as a parameter.
// If validation errors occur, they will be returned in as a slice of ValidationError
// structs. If any other error occurs, it will be returned as a separate error
func (s *Schema) Validate(data interface{}) ([]ValidationError, error) {
	def, err := s.ValidateSchema()
	if err != nil {
		return nil, err
	}

	jsonPayload, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "unable to process data")
	}

	payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, errors.Wrap(err, "unable to process data")
	}

	err = def.Validate(payload)
	if err == nil {
		return nil, nil
	}

	if verr, ok := err.(*jsonschema.ValidationError); ok {
		valErrors := make([]ValidationError, 0, len(verr.Causes))
		for _, e := range verr.Causes {
			path := strings.Join(e.InstanceLocation, "/")
			path = "/" + path
			valError := ValidationError{
				Path:  path,
				Error: e.Error(),
			}
			valErrors = append(valErrors, valError)
		}
		return valErrors, nil
	}
	return nil, err
}

// CoerceValue can be used to turn float and other numeric types into integers. When
// unmarshaled, often integer values are not represented as an integer. This is a
// convenience method.
func (s *Schema) CoerceValue(value interface{}) interface{} {
	if s.Type == "int" || s.Type == "integer" {
		f, ok := value.(float64)
		if ok {
			i, ok := asInt(f)
			if !ok {
				return f
			}
			return i
		}
	}
	return value
}

func asInt(f float64) (int, bool) {
	i := int(f)
	if float64(i) != f {
		return 0, false
	}
	return i, true
}
