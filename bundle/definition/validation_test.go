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
	err = definition.Validate(val)
	assert.Nil(t, err, "expected no validation errors")
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
	err = definition.Validate(val)
	assert.NotNil(t, err, "expected a validation error")
	assert.EqualError(t,
		err,
		"invalid parameter or output: unable to validate /port, error: must be greater than or equal to 100.000000",
	)
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
	err = definition.Validate(val)
	assert.NotNil(t, err, "expected a validation error")
	assert.EqualError(t,
		err,
		"invalid parameter or output: unable to validate /, error: \"host\" value is required",
	)
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
	err = definition.Validate(val)
	assert.NotNil(t, err, "expected a validation error")
	assert.EqualError(t,
		err,
		"invalid parameter or output: unable to validate /badActor, error: cannot match schema",
	)
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
	err = definition.Validate(val)
	assert.NotNil(t, err, "expected a validation error")
	assert.EqualError(t,
		err,
		"invalid parameter or output: unable to validate /badActor, error: type should be string",
	)
}
