package schemavalidation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/qri-io/jsonschema"
)

// Validate validates the provided objectBytes against the provided CNAB-Spec schemaType
// The schemaType must be a valid schema currently hosted at https://cnab.io/v1/*.schema.json
func Validate(schemaType string, objectBytes []byte) ([]jsonschema.ValError, error) {
	url := fmt.Sprintf("https://cnab.io/v1/%s.schema.json", schemaType)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to construct GET request for fetching %s schema", schemaType)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s schema", schemaType)
	}

	defer res.Body.Close()
	schemaData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s schema", schemaType)
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
