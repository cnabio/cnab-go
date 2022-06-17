package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/claim"
)

// CNABSchemaURLPrefix is the URL prefix to fetch schemas from
const CNABSchemaURLPrefix = "https://cdn.cnab.io/schema"

// CNABSchemaDestPrefix is the filepath prefix to write schemas to
const CNABSchemaDestPrefix = "./schema/schema"

func main() {
	schemas := map[string]string{
		"bundle": bundle.CNABSpecVersion,
		"claim":  claim.CNABSpecVersion,
	}

	for schema, version := range schemas {
		bytes, err := fetchSchema(schema, version)
		if err != nil {
			fmt.Printf("unable to fetch %s schema with version %s: %s\n", schema, version, err.Error())
			// if cdn.cnab.io is not reachable, we stick with default schema
			continue
		}

		err = writeSchema(schema, bytes)
		if err != nil {
			fmt.Printf("unable to write %s schema: %s\n", schema, err.Error())
		}
	}
}

func fetchSchema(schemaType, schemaVersion string) ([]byte, error) {
	schemaURL := fmt.Sprintf("%s/%s/%s.schema.json", CNABSchemaURLPrefix, schemaVersion, schemaType)
	fmt.Println("Retrieving schema", schemaURL)
	resp, err := http.Get(schemaURL)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch schema from %q", schemaURL)
	}

	data, err := ioutil.ReadAll(resp.Body)
	return data, errors.Wrap(err, "unable to read response body")
}

func writeSchema(schemaType string, data []byte) error {
	dest := fmt.Sprintf("%s/%s.schema.json", CNABSchemaDestPrefix, schemaType)
	err := ioutil.WriteFile(dest, data, 0644)
	return errors.Wrapf(err, "unable to write file to %q", dest)
}
