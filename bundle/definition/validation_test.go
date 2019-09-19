package definition

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectValidationValid(t *testing.T) {
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
	assert.Len(t, valErrors, 0, "expected no validation errors")
	assert.NoError(t, err)
}

func TestObjectValidationValid_CustomValidator_ContentEncoding_base64(t *testing.T) {
	s := `{
		"type": "object",
		"properties" : {
			"file" : {
				"type": "string",
				"contentEncoding": "base64"
			}
		},
		"required" : ["file"]
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "should have been able to marshal definition")
	assert.Equal(t, "object", definition.Type, "type should have been an object")
	props := definition.Properties
	assert.NotNil(t, props, "should have found properties")
	assert.Equal(t, 1, len(props), "should have had a single property")
	propSchema, ok := props["file"]
	assert.True(t, ok, "should have found file property")
	assert.Equal(t, "string", propSchema.Type, "file type should have been a string")
	assert.Equal(t, "base64", propSchema.ContentEncoding, "file contentEncoding should have been base64")

	val := struct {
		File string `json:"file"`
	}{
		File: "SGVsbG8gV29ybGQhCg==",
	}
	valErrors, err := definition.Validate(val)
	assert.NoError(t, err)
	assert.Len(t, valErrors, 0, "expected no validation errors")

	invalidVal := struct {
		File string `json:"file"`
	}{
		File: "SGVsbG8gV29ybGQhCg===",
	}
	valErrors, err = definition.Validate(invalidVal)
	assert.NoError(t, err)
	assert.Len(t, valErrors, 1, "expected 1 validation error")
	assert.Equal(t, "invalid base64 value: SGVsbG8gV29ybGQhCg===", valErrors[0].Error)
}

func TestObjectValidationValid_CustomValidator_ContentEncoding_InvalidEncoding(t *testing.T) {
	s := `{
		"type": "object",
		"properties" : {
			"file" : {
				"type": "string",
				"contentEncoding": "base65"
			}
		},
		"required" : ["file"]
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "should have been able to marshal definition")

	val := struct {
		File string `json:"file"`
	}{
		File: "SGVsbG8gV29ybGQhCg==",
	}
	valErrors, err := definition.Validate(val)
	assert.NoError(t, err)
	assert.Len(t, valErrors, 1, "expected 1 validation error")
	assert.Equal(t, "unsupported or invalid contentEncoding type of base65", valErrors[0].Error)
}

func TestObjectValidationInValidMinimum(t *testing.T) {
	s := `{
		"type": "object",
		"properties" : {
			"port" : {
				"default": 80,
				"maximum": 10240,
				"minimum": 100,
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
	assert.Nil(t, err, "expected no error")
	assert.Len(t, valErrors, 1, "expected a single validation error")
	valErr := valErrors[0]
	assert.NotNil(t, valErr, "expected the obtain the validation error")
	assert.Equal(t, "/port", valErr.Path, "expected validation error to reference port")
	assert.Equal(t, "must be greater than or equal to 100.000000", valErr.Error, "expected validation error to reference port")
}

func TestObjectValidationPropertyRequired(t *testing.T) {
	s := `{
		"type": "object",
		"properties" : {
			"port" : {
				"default": 80,
				"maximum": 10240,
				"minimum": 10,
				"type": "integer"
			},
			"host" : {
				"type" : "string"
			}
		}, 
		"required" : ["port","host"]
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "should have been able to marshall definition")
	assert.Equal(t, "object", definition.Type, "type should have been an object")
	props := definition.Properties
	assert.NotNil(t, props, "should have found properties")
	assert.Equal(t, 2, len(props), "should have had a two properties")
	propSchema, ok := props["port"]
	assert.True(t, ok, "should have found port property")
	assert.Equal(t, "integer", propSchema.Type, "port type should have been an integer")

	val := struct {
		Port int `json:"port"`
	}{
		Port: 80,
	}
	valErrors, err := definition.Validate(val)
	assert.Len(t, valErrors, 1, "expected a validation error")
	assert.NoError(t, err)
	assert.Equal(t, "\"host\" value is required", valErrors[0].Error)

}

func TestObjectValidationNoAdditionalPropertiesAllowed(t *testing.T) {
	s := `{
		"type": "object",
		"properties" : {
			"port" : {
				"default": 80,
				"maximum": 10240,
				"minimum": 10,
				"type": "integer"
			},
			"host" : {
				"type" : "string"
			}
		},
		"additionalProperties" : false,
		"required" : ["port","host"]
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "should have been able to marshall definition")
	assert.Equal(t, "object", definition.Type, "type should have been an object")
	props := definition.Properties
	assert.NotNil(t, props, "should have found properties")
	assert.Equal(t, 2, len(props), "should have had a two properties")
	propSchema, ok := props["port"]
	assert.True(t, ok, "should have found port property")
	assert.Equal(t, "integer", propSchema.Type, "port type should have been an integer")

	val := struct {
		Port     int    `json:"port"`
		Host     string `json:"host"`
		BadActor bool   `json:"badActor"`
	}{
		Port:     80,
		Host:     "localhost",
		BadActor: true,
	}
	valErrors, err := definition.Validate(val)
	assert.Len(t, valErrors, 1, "expected a validation error")
	assert.NoError(t, err)
	assert.Equal(t, "/badActor", valErrors[0].Path, "expected the error to be on badActor")
	assert.Equal(t, "cannot match schema", valErrors[0].Error)
}

func TestObjectValidationAdditionalPropertiesAreStrings(t *testing.T) {
	s := `{
		"type": "object",
		"properties" : {
			"port" : {
				"default": 80,
				"maximum": 10240,
				"minimum": 10,
				"type": "integer"
			},
			"host" : {
				"type" : "string"
			}
		},
		"additionalProperties" : {
			"type" : "string"
		},
		"required" : ["port"]
	}`
	definition := new(Schema)
	err := json.Unmarshal([]byte(s), definition)
	require.NoError(t, err, "should have been able to marshall definition")
	assert.Equal(t, "object", definition.Type, "type should have been an object")
	props := definition.Properties
	assert.NotNil(t, props, "should have found properties")
	assert.Equal(t, 2, len(props), "should have had a two properties")
	propSchema, ok := props["port"]
	assert.True(t, ok, "should have found port property")
	assert.Equal(t, "integer", propSchema.Type, "port type should have been an integer")

	val := struct {
		Port      int    `json:"port"`
		Host      string `json:"host"`
		GoodActor string `json:"goodActor"`
		BadActor  bool   `json:"badActor"`
	}{
		Port:      80,
		Host:      "localhost",
		GoodActor: "hello",
		BadActor:  false,
	}
	valErrors, err := definition.Validate(val)
	assert.Len(t, valErrors, 1, "expected a validation error")
	assert.NoError(t, err)
	assert.Equal(t, "type should be string", valErrors[0].Error)
}
