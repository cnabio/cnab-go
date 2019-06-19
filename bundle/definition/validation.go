package definition

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/qri-io/jsonschema"
)

// Validate applies JSON Schema validation to the data passed as a paraemeter.
// If validation errors occur, they will be returned in as an error.
func (s *Schema) Validate(data interface{}) error {

	b, err := json.Marshal(s)
	if err != nil {
		return errors.Wrap(err, "unable to load schema")
	}
	def := new(jsonschema.RootSchema)
	err = json.Unmarshal([]byte(b), def)
	if err != nil {
		return errors.Wrap(err, "unable to build schema")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "unable to process data")
	}
	valErrs, err := def.ValidateBytes(payload)
	if err != nil {
		return errors.Wrap(err, "unable to perform validation")
	}
	if len(valErrs) > 0 {
		var anError error
		for _, err := range valErrs {
			if anError == nil {
				anError = errors.New(fmt.Sprintf("unable to validate %s, error: %s", err.PropertyPath, err.Message))
			} else {
				anError = errors.Wrap(anError, fmt.Sprintf("unable to validate %s, error: %s", err.PropertyPath, err.Message))
			}
		}
		return errors.Wrap(anError, "invalid parameter or output")
	}
	return nil
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
