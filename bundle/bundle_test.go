package bundle

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTopLevelProperties(t *testing.T) {
	json := `{
		"name": "foo",
		"version": "1.0",
		"images": {},
		"credentials": {},
		"custom": {}
	}`
	bundle, err := Unmarshal([]byte(json))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Name != "foo" {
		t.Errorf("Expected name 'foo', got '%s'", bundle.Name)
	}
	if bundle.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", bundle.Version)
	}
	if len(bundle.Images) != 0 {
		t.Errorf("Expected no images, got %d", len(bundle.Images))
	}
	if len(bundle.Credentials) != 0 {
		t.Errorf("Expected no credentials, got %d", len(bundle.Credentials))
	}
	if len(bundle.Custom) != 0 {
		t.Errorf("Expected no custom extensions, got %d", len(bundle.Custom))
	}
}

func TestReadImageProperties(t *testing.T) {
	data, err := ioutil.ReadFile("../testdata/bundles/foo.json")
	if err != nil {
		t.Errorf("cannot read bundle file: %v", err)
	}

	bundle, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(bundle.Images))
	}
	image1 := bundle.Images["image1"]
	if image1.Description != "image1" {
		t.Errorf("Expected description 'image1', got '%s'", image1.Description)
	}
	if image1.Image != "urn:image1uri" {
		t.Errorf("Expected Image 'urn:image1uri', got '%s'", image1.Image)
	}
	if image1.OriginalImage != "urn:image1originaluri" {
		t.Errorf("Expected Image 'urn:image1originaluri', got '%s'", image1.OriginalImage)
	}
	image2 := bundle.Images["image2"]
	if image2.OriginalImage != "" {
		t.Errorf("Expected Image '', got '%s'", image2.OriginalImage)
	}
}

func TestReadCredentialProperties(t *testing.T) {
	data, err := ioutil.ReadFile("../testdata/bundles/foo.json")
	if err != nil {
		t.Errorf("cannot read bundle file: %v", err)
	}

	bundle, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Credentials) != 3 {
		t.Errorf("Expected 3 credentials, got %d", len(bundle.Credentials))
	}
	f := bundle.Credentials["foo"]
	if f.Path != "pfoo" {
		t.Errorf("Expected path 'pfoo', got '%s'", f.Path)
	}
	if f.EnvironmentVariable != "" {
		t.Errorf("Expected env '', got '%s'", f.EnvironmentVariable)
	}
	b := bundle.Credentials["bar"]
	if b.Path != "" {
		t.Errorf("Expected path '', got '%s'", b.Path)
	}
	if b.EnvironmentVariable != "ebar" {
		t.Errorf("Expected env 'ebar', got '%s'", b.EnvironmentVariable)
	}
	q := bundle.Credentials["quux"]
	if q.Path != "pquux" {
		t.Errorf("Expected path 'pquux', got '%s'", q.Path)
	}
	if q.EnvironmentVariable != "equux" {
		t.Errorf("Expected env 'equux', got '%s'", q.EnvironmentVariable)
	}
}

func TestValuesOrDefaults(t *testing.T) {
	is := assert.New(t)
	vals := map[string]interface{}{
		"port":    8080,
		"host":    "localhost",
		"enabled": true,
	}
	b := &Bundle{
		Parameters: ParametersDefinition{
			Fields: map[string]ParameterDefinition{
				"port": {
					DataType: "int",
					Default:  1234,
				},
				"host": {
					DataType: "string",
					Default:  "localhost.localdomain",
				},
				"enabled": {
					DataType: "bool",
					Default:  false,
				},
				"replicaCount": {
					DataType: "int",
					Default:  3,
				},
			},
		},
	}

	vod, err := ValuesOrDefaults(vals, b)

	is.NoError(err)
	is.True(vod["enabled"].(bool))
	is.Equal(vod["host"].(string), "localhost")
	is.Equal(vod["port"].(int), 8080)
	is.Equal(vod["replicaCount"].(int), 3)

	// This should err out because of type problem
	vals["replicaCount"] = "banana"
	_, err = ValuesOrDefaults(vals, b)
	is.Error(err)
}

func TestValuesOrDefaults_Required(t *testing.T) {
	is := assert.New(t)
	vals := map[string]interface{}{
		"enabled": true,
	}
	b := &Bundle{
		Parameters: ParametersDefinition{
			Fields: map[string]ParameterDefinition{
				"minimum": {
					DataType: "int",
				},
				"enabled": {
					DataType: "bool",
					Default:  false,
				},
			},
			Required: []string{"minimum"},
		},
	}

	_, err := ValuesOrDefaults(vals, b)
	is.Error(err)

	// It is unclear what the outcome should be when the user supplies
	// empty values on purpose. For now, we will assume those meet the
	// minimum definition of "required", and that other rules will
	// correct for empty values.
	//
	// Example: It makes perfect sense for a user to specify --set minimum=0
	// and in so doing meet the requirement that a value be specified.
	vals["minimum"] = 0
	res, err := ValuesOrDefaults(vals, b)
	is.NoError(err)
	is.Equal(0, res["minimum"])
}

func TestValidateVersionTag(t *testing.T) {
	is := assert.New(t)

	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "latest",
		InvocationImages: []InvocationImage{img},
	}

	err := b.Validate()
	is.EqualError(err, "'latest' is not a valid bundle version")
}

func TestValidateBundle_RequiresInvocationImage(t *testing.T) {
	b := Bundle{
		Name:    "bar",
		Version: "0.1.0",
	}

	err := b.Validate()
	if err == nil {
		t.Fatal("Validate should have failed because the bundle has no invocation images")
	}

	b.InvocationImages = append(b.InvocationImages, InvocationImage{})

	err = b.Validate()
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadCustomExtensions(t *testing.T) {
	data, err := ioutil.ReadFile("../testdata/bundles/foo.json")
	if err != nil {
		t.Errorf("cannot read bundle file: %v", err)
	}

	bundle, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(bundle.Custom) != 2 {
		t.Errorf("Expected 2 custom extensions, got %d", len(bundle.Custom))
	}

	duffleExtI, ok := bundle.Custom["com.example.duffle-bag"]
	if !ok {
		t.Fatal("Expected the com.example.duffle-bag extension")
	}
	duffleExt, ok := duffleExtI.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected the com.example.duffle-bag to be of type map[string]interface{} but got %T ", duffleExtI)
	}
	assert.Equal(t, "PNG", duffleExt["iconType"])
	assert.Equal(t, "https://example.com/icon.png", duffleExt["icon"])

	backupExtI, ok := bundle.Custom["com.example.backup-preferences"]
	if !ok {
		t.Fatal("Expected the com.example.backup-preferences extension")
	}
	backupExt, ok := backupExtI.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected the com.example.backup-preferences to be of type map[string]interface{} but got %T ", backupExtI)
	}
	assert.Equal(t, true, backupExt["enabled"])
	assert.Equal(t, "daily", backupExt["frequency"])
}

func TestOutputs_Marshall(t *testing.T) {
	bundleJSON := `
	{
		"outputs": {
			"fields" : {
				"clientCert": {
					"contentEncoding": "base64",
					"contentMediaType": "application/x-x509-user-cert",
					"path": "/cnab/app/outputs/clientCert",
					"sensitive": true,
					"type": "string"
				},
				"hostName": {
					"applyTo": [
					"install"
					],
					"description": "the hostname produced installing the bundle",
					"path": "/cnab/app/outputs/hostname",
					"type": "string"
				},
				"port": {
					"path": "/cnab/app/outputs/port",
					"type": "integer"
				}
			}
		}
	}`

	bundle, err := Unmarshal([]byte(bundleJSON))
	assert.NoError(t, err, "should have unmarshalled the bundle")
	require.NotNil(t, bundle.Outputs, "test must fail, not outputs found")
	assert.Equal(t, 3, len(bundle.Outputs.Fields))
	assert.Equal(t, 0, len(bundle.Outputs.Required))

	clientCert, ok := bundle.Outputs.Fields["clientCert"]
	require.True(t, ok, "expected clientCert to exist as an output")
	assert.Equal(t, "string", clientCert.DataType)
	assert.True(t, clientCert.Sensitive, "expected clientCert to be a sensitive value")
	assert.Equal(t, "/cnab/app/outputs/clientCert", clientCert.Path, "clientCert path was not the expected value")

	hostName, ok := bundle.Outputs.Fields["hostName"]
	require.True(t, ok, "expected hostname to exist as an output")
	assert.Equal(t, "string", hostName.DataType)
	assert.Equal(t, "/cnab/app/outputs/hostname", hostName.Path, "hostName path was not the expected value")

	port, ok := bundle.Outputs.Fields["port"]
	require.True(t, ok, "expected port to exist as an output")
	assert.Equal(t, "integer", port.DataType)
	assert.Equal(t, "/cnab/app/outputs/port", port.Path, "port path was not the expected value")
}
