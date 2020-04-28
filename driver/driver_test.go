package driver

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cnabio/cnab-go/bundle"
)

func TestOperation_Unmarshall(t *testing.T) {
	expectedOp := Operation{
		Action:       "install",
		Installation: "test",
		Parameters: map[string]interface{}{
			"param1": "value1",
			"param2": "value2",
		},
		Image: bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image:     "cnab/helloworld:latest",
				ImageType: "docker",
				Digest:    "sha256:55f83710272990efab4e076f9281453e136980becfd879640b06552ead751284",
			},
		},
		Revision: "01DDY0MT808KX0GGZ6SMXN4TW",
		Environment: map[string]string{
			"ENV1": "value1",
			"ENV2": "value2",
		},
		Files: map[string]string{
			"/cnab/app/image-map.json": "{}",
		},
	}
	var op Operation
	is := assert.New(t)
	bytes, err := ioutil.ReadFile("../testdata/operations/valid-operation.json")
	is.NoError(err, "Error reading from testdata/operations/valid-operation.json")
	is.NoError(json.Unmarshal(bytes, &op), "Error unmarshalling operation")
	is.NotNil(op, "Expected Operation not to be nil")
	is.Equal(expectedOp, op, "Validating value of unmarshalled operation failed")
}

func TestOperation_Marshall(t *testing.T) {
	actualOp := Operation{
		Action:       "install",
		Installation: "test",
		Parameters: map[string]interface{}{
			"param1": "value1",
			"param2": "value2",
		},
		Image: bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image:     "cnab/helloworld:latest",
				ImageType: "docker",
				Digest:    "sha256:55f83710272990efab4e076f9281453e136980becfd879640b06552ead751284",
			},
		},
		Revision: "01DDY0MT808KX0GGZ6SMXN4TW",
		Environment: map[string]string{
			"ENV1": "value1",
			"ENV2": "value2",
		},
		Files: map[string]string{
			"/cnab/app/image-map.json": "{}",
		},
		Out: os.Stdout,
	}
	is := assert.New(t)
	bytes, err := json.Marshal(actualOp)
	is.NoError(err, "Error Marshalling actual operation to json")
	is.NotNil(bytes, "Expected marshalled json not to be nil")
	actualJSON := string(bytes)
	var expectedOp Operation
	bytes, err = ioutil.ReadFile("../testdata/operations/valid-operation.json")
	is.NoError(err, "Error reading from testdata/operations/valid-operation.json")
	is.NoError(json.Unmarshal(bytes, &expectedOp), "Error unmarshalling expected operation")
	bytes, err = json.Marshal(expectedOp)
	is.NoError(err, "Error Marshalling expected operation to json")
	is.NotNil(bytes, "Expected marshalled json not to be nil")
	expectedJSON := string(bytes)
	is.Equal(expectedJSON, actualJSON)
}
