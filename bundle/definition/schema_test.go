package definition

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleUnMarshallDefinition(t *testing.T) {
	def := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "array",
		"items": {
			"type": "object",
			"required": ["description", "schema", "tests"],
			"properties": {
				"description": {"type": "string"},
				"schema": {},
				"tests": {
					"type": "array",
					"items": {
						"type": "object",
						"required": ["description", "data", "valid"],
						"properties": {
							"description": {"type": "string"},
							"data": {},
							"valid": {"type": "boolean"}
						},
						"additionalProperties": false
					},
					"minItems": 1
				}
			},
			"additionalProperties": false,
			"minItems": 1
		}	
	}`

	definition := new(Schema)
	err := json.Unmarshal([]byte(def), definition)
	require.NoError(t, err, "should have been able to marshall definition")
	assert.Equal(t, "array", definition.Type, "type should have been an array")
}

func TestSimpleSchema(t *testing.T) {
	s := `
	{
		"default": 80,
		"maximum": 10240,
		"minimum": 10,
		"type": "integer"
	}
	`

	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "should have been able to marshall definition")
	assert.Equal(t, "integer", definition.Type, "type should have been an integer")

	maxVal := definition.Maximum
	require.NotNil(t, maxVal, "maximum should have been loaded")
	assert.Equal(t, 10240, int(*maxVal), "max should have been 10240")

	minVal := definition.Minimum
	require.NotNil(t, minVal, "minimum should have been loaded")
	assert.Equal(t, 10, int(*minVal), "min should have been 10")

	def, ok := definition.Default.(float64)
	require.True(t, ok, "default should h ave been float64")
	assert.Equal(t, 80, int(def), "default should have been 80")

}

func TestUnknownSchemaType(t *testing.T) {
	s := `
	{
		"type": "cnab"
	}
	`

	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	assert.Error(t, err, "should not have been able to marshall definition")
	assert.EqualError(t, err, "error unmarshaling type from json: \"cnab\" is not a valid type")
}

func TestSingleSchemaType(t *testing.T) {
	s := `
	{
		"type": "number" 
	}
	`

	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	assert.NoError(t, err, "should have been able to marshall definition")

	singleType := definition.SingleType()
	assert.True(t, singleType, "this should have had a single type")
	typeString, ok, err := definition.GetType()
	assert.NoError(t, err, "should have gotten back no error on fetch of types")
	assert.True(t, ok, "types should have been a slice of strings")
	assert.Equal(t, "number", typeString, "should have had a number type")
}

func TestMultipleSchemaTypes(t *testing.T) {
	s := `
	{
		"type": ["number", "string"] 
	}
	`

	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	assert.NoError(t, err, "should have been able to marshall definition")
	singleType := definition.SingleType()
	assert.False(t, singleType, "this should have had multiple types")

	types, ok, err := definition.GetTypes()
	assert.NoError(t, err, "should have gotten back no error on fetch of types")
	assert.True(t, ok, "types should have been a slice of strings")
	assert.Equal(t, 2, len(types), "should have had two types")
}

func TestObjectDefinitionType(t *testing.T) {
	s := `{
		"type": "object",
		"properties" : {
			"port" : {
				"default": 80,
				"maximum": 10240,
				"minimum": 10,
				"type": "integer"
			}
		}, 
		"required" : ["port"]
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "should have been able to marshall definition")
	assert.Equal(t, "object", definition.Type, "type should have been an object")
	props := definition.Properties
	assert.NotNil(t, props, "should have found properties")
	assert.Equal(t, 1, len(props), "should have had a single property")
	propSchema, ok := props["port"]
	assert.True(t, ok, "should have found port property")
	assert.Equal(t, "integer", propSchema.Type, "port type should have been an integer")

	val := struct {
		Port int `json:"port"`
	}{
		Port: 80,
	}
	err = definition.Validate(val)
	assert.Nil(t, err, "expected no validation errors")
}

func TestBooleanTypeValidation(t *testing.T) {
	boolValue := "true"
	s := valueTestJSON("boolean", boolValue, boolValue)
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "schema should be valid for bool test")
	thingToCheck := true
	err = definition.Validate(thingToCheck)
	assert.NoError(t, err, "check of true should have resulted in no validation errors")
	err = definition.Validate(!thingToCheck)
	assert.Error(t, err, "check of false should have resulted in a validation error")
	assert.EqualError(t, err, "invalid parameter or output: unable to validate /, error: should be one of [true]")

	boolValue2 := "true, false"
	s2 := valueTestJSON("boolean", boolValue, boolValue2)
	definition2 := new(Schema)
	err = json.Unmarshal([]byte(s2), definition2)
	err = definition2.Validate(thingToCheck)
	assert.NoError(t, err, "check of true should have resulted in no validation errors with both allowed")
	err = definition2.Validate(!thingToCheck)
	assert.NoError(t, err, "check of false should have resulted in no validation errors with both allowed")
}

func TestStringTypeValidationEnum(t *testing.T) {
	defaultVal := "\"dog\""
	s := valueTestJSON("string", defaultVal, defaultVal)
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "schema should be valid for string test")
	err = definition.Validate("dog")
	assert.NoError(t, err, "check of `dog` should have resulted in no validation errors")
	err = definition.Validate("cat")
	assert.Error(t, err, "check of 'cat' should have resulted in a validation error")
	assert.EqualError(t, err, "invalid parameter or output: unable to validate /, error: should be one of [\"dog\"]")

	anotherSchema := `{
		"type" : "string",
		"enum" : ["chicken", "duck"]
	}`

	definition2 := new(Schema)
	err = json.Unmarshal([]byte(anotherSchema), definition2)
	require.NoError(t, err, "should have been a valid schema")

	err = definition2.Validate("pig")
	assert.EqualError(t, err, "invalid parameter or output: unable to validate /, error: should be one of [\"chicken\", \"duck\"]")
}

func TestStringMinLengthValidator(t *testing.T) {
	aSchema := `{
		"type" : "string",
		"minLength" : 10
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(aSchema), definition)
	require.NoError(t, err, "should have been a valid schema")

	err = definition.Validate("four")
	assert.Error(t, err, "expected the validation to fail with four characters")
	assert.EqualError(t, err, "invalid parameter or output: unable to validate /, error: min length of 10 characters required: four")

	err = definition.Validate("abcdefghijklmnopqrstuvwxyz")
	assert.NoError(t, err, "expected the validation to not fail with more than 10 characters")

	err = definition.Validate("qwertyuiop")
	assert.NoError(t, err, "expected the validation to not fail with exactly 10 characters")
}

func TestStringMaxLengthValidator(t *testing.T) {
	aSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
    	"$id": "http://json-schema.org/draft-07/schema#",
    	"title": "a string validator",
		"type" : "string",
		"maxLength" : 10
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(aSchema), definition)
	require.NoError(t, err, "should have been a valid schema")

	err = definition.Validate("four")
	assert.NoError(t, err, "expected the validation to not fail with four characters")

	err = definition.Validate("abcdefghijklmnopqrstuvwxyz")
	assert.Error(t, err, "expected the validation to fail with more than 10 characters")
	assert.EqualError(t, err, "invalid parameter or output: unable to validate /, error: max length of 10 characters exceeded: abcdefghijklmnopqrstuvwxyz")

	err = definition.Validate("qwertyuiop")
	assert.NoError(t, err, "expected the validation to not fail with exactly 10 characters")
}

func valueTestJSON(kind, def, enum string) []byte {
	return []byte(fmt.Sprintf(`{
		"type" : "%s",
		"default": %s,
		"enum": [ %s ]
	}`, kind, def, enum))
}
