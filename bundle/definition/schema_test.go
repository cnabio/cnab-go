package definition

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	yaml "gopkg.in/yaml.v2"
)

func TestSimpleUnMarshalDefinition(t *testing.T) {
	def := `{
		"$comment": "schema comment",
		"$id": "schema id",
		"$ref": "schema ref",
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "array",
		"items": [
			{
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
		],
		"additionalItems": {
			"type": "string"
		}
	}`

	definition := new(Schema)
	err := json.Unmarshal([]byte(def), definition)
	require.NoError(t, err, "should have been able to json.Marshal definition")
	assert.Equal(t, "array", definition.Type, "type should have been an array")

	err = yaml.UnmarshalStrict([]byte(def), definition)
	require.NoError(t, err, "should have been able to yaml.Marshal definition")
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
	valErrors, err := definition.Validate(val)
	assert.Empty(t, valErrors, "expected no validation errors")
	assert.NoError(t, err, "expected not to encounter an error in the validation")
}

func TestBooleanTypeValidation(t *testing.T) {
	boolValue := "true"
	s := valueTestJSON("boolean", boolValue, boolValue)
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "schema should be valid for bool test")
	thingToCheck := true
	valErrors, err := definition.Validate(thingToCheck)
	assert.Empty(t, valErrors, "check of true should have resulted in no validation errors")
	assert.NoError(t, err, "should not have encountered an error in the validator")
	valErrors, err = definition.Validate(!thingToCheck)
	assert.NoError(t, err, "this check should not have resulted in an error")
	assert.Len(t, valErrors, 1, "expected a validation error")
	valErr := valErrors[0]
	assert.Equal(t, "/", valErr.Path, "expected validation to fail at the root")
	assert.Equal(t, "should be one of [true]", valErr.Error)

	boolValue2 := "true, false"
	s2 := valueTestJSON("boolean", boolValue, boolValue2)
	definition2 := new(Schema)
	err = json.Unmarshal([]byte(s2), definition2)
	require.NoError(t, err, "test requires unmarshaled bundled")
	valErrors, err = definition2.Validate(thingToCheck)
	assert.Len(t, valErrors, 0, "check of true should have resulted in no validation errors with both allowed")
	assert.NoError(t, err, "should not have encountered an error")
	valErrors, err = definition2.Validate(!thingToCheck)
	assert.Len(t, valErrors, 0, "check of false should have resulted in no validation errors with both allowed")
	assert.NoError(t, err, "should not have encountered an error")
}

func TestStringTypeValidationEnum(t *testing.T) {
	defaultVal := "\"dog\""
	s := valueTestJSON("string", defaultVal, defaultVal)
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "schema should be valid for string test")
	valErrors, err := definition.Validate("dog")
	assert.Len(t, valErrors, 0, "check of `dog` should have resulted in no validation errors")
	assert.NoError(t, err, "should not have encountered an error")
	valErrors, err = definition.Validate("cat")
	assert.Len(t, valErrors, 1, "check of 'cat' should have resulted in a validation error")
	assert.NoError(t, err)
	valErr := valErrors[0]
	assert.Equal(t, "/", valErr.Path, "expected validation to fail at the root")
	assert.Equal(t, "should be one of [\"dog\"]", valErr.Error)

	anotherSchema := `{
		"type" : "string",
		"enum" : ["chicken", "duck"]
	}`

	definition2 := new(Schema)
	err = json.Unmarshal([]byte(anotherSchema), definition2)
	require.NoError(t, err, "should have been a valid schema")

	valErrors, err = definition2.Validate("pig")
	assert.NoError(t, err, "shouldn't have gotten an actual error")
	assert.Len(t, valErrors, 1, "expected validation failure for pig")
	assert.Equal(t, "should be one of [\"chicken\", \"duck\"]", valErrors[0].Error)
}

func TestStringMinLengthValidator(t *testing.T) {
	aSchema := `{
		"type" : "string",
		"minLength" : 10
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(aSchema), definition)
	require.NoError(t, err, "should have been a valid schema")

	valErrors, err := definition.Validate("four")
	assert.Len(t, valErrors, 1, "expected the validation to fail with four characters")
	assert.Equal(t, "min length of 10 characters required: four", valErrors[0].Error)
	assert.NoError(t, err)

	valErrors, err = definition.Validate("abcdefghijklmnopqrstuvwxyz")
	assert.Len(t, valErrors, 0, "expected the validation to not fail with more than 10 characters")
	assert.NoError(t, err)

	valErrors, err = definition.Validate("qwertyuiop")
	assert.Len(t, valErrors, 0, "expected the validation to not fail with exactly 10 characters")
	assert.NoError(t, err)
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

	valErrors, err := definition.Validate("four")
	assert.Len(t, valErrors, 0, "expected the validation to not fail with four characters")
	assert.NoError(t, err)

	valErrors, err = definition.Validate("abcdefghijklmnopqrstuvwxyz")
	assert.Len(t, valErrors, 1, "expected the validation to fail with more than 10 characters")
	assert.NoError(t, err)

	valErrors, err = definition.Validate("qwertyuiop")
	assert.Len(t, valErrors, 0, "expected the validation to not fail with exactly 10 characters")
	assert.NoError(t, err)
}

func valueTestJSON(kind, def, enum string) []byte {
	return []byte(fmt.Sprintf(`{
		"type" : "%s",
		"default": %s,
		"enum": [ %s ]
	}`, kind, def, enum))
}

func TestConvertValue(t *testing.T) {
	pd := Schema{
		Type: "boolean",
	}
	is := assert.New(t)

	out, _ := pd.ConvertValue("true")
	is.True(out.(bool))
	out, _ = pd.ConvertValue("false")
	is.False(out.(bool))
	out, _ = pd.ConvertValue("barbeque")
	is.False(out.(bool))

	pd.Type = "string"
	out, err := pd.ConvertValue("hello")
	is.NoError(err)
	is.Equal("hello", out.(string))

	pd.Type = "integer"
	out, err = pd.ConvertValue("123")
	is.NoError(err)
	is.Equal(123, out.(int))

	_, err = pd.ConvertValue("onetwothree")
	is.Error(err)

	pd.Type = "number"
	_, err = pd.ConvertValue("123")
	is.Error(err)
	is.Contains(err.Error(), "invalid definition")

	_, err = pd.ConvertValue("5.5")
	is.Error(err)
	is.Contains(err.Error(), "invalid definition")

	_, err = pd.ConvertValue("nope")
	is.Error(err)

	pd.Type = "array"
	_, err = pd.ConvertValue("nope")
	is.Error(err)

	_, err = pd.ConvertValue("123")
	is.Error(err)

	_, err = pd.ConvertValue("true")
	is.Error(err)

	_, err = pd.ConvertValue("123.5")
	is.Error(err)

	pd.Type = "object"
	out, err = pd.ConvertValue(`{"object": true}`)
	is.NoError(err)
	is.Equal(map[string]interface{}{"object": true}, out)

	out, err = pd.ConvertValue(`{"object" true}`)
	is.Error(err)
	is.Contains(err.Error(), "could not unmarshal")
	is.Equal(nil, out)
}
