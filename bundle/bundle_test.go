package bundle

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/schema"
)

func TestReadTopLevelProperties(t *testing.T) {
	json := `{
		"schemaVersion": "1.0.1",
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
	assert.Equal(t, "1.0.1", string(bundle.SchemaVersion))
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
		Definitions: map[string]*definition.Schema{
			"portType": {
				Type:    "integer",
				Default: 1234,
			},
			"hostType": {
				Type:    "string",
				Default: "locahost.localdomain",
			},
			"replicaCountType": {
				Type:    "integer",
				Default: 3,
			},
			"enabledType": {
				Type:    "boolean",
				Default: false,
			},
			"msg": {
				Type: "string",
				// msg has no default, but it is optional
				// Make sure that we are not validating optional unspecified parameters
			},
		},
		Parameters: map[string]Parameter{
			"port": {
				Definition: "portType",
			},
			"host": {
				Definition: "hostType",
			},
			"enabled": {
				Definition: "enabledType",
			},
			"replicaCount": {
				Definition: "replicaCountType",
			},
			"msg": {
				Definition: "msg",
			},
		},
	}

	vod, err := ValuesOrDefaults(vals, b, "install")

	is.NoError(err)
	is.True(vod["enabled"].(bool))
	is.Equal(vod["host"].(string), "localhost")
	is.Equal(vod["port"].(int), 8080)
	is.Equal(vod["replicaCount"].(int), 3)
	is.Equal(nil, vod["msg"], "msg", "msg should be passed even though it had no default and wasn't set because the spec requires it")

	// This should err out because of type problem
	vals["replicaCount"] = "banana"
	_, err = ValuesOrDefaults(vals, b, "install")
	is.Error(err)

	// Check for panic when zero value Bundle is passed
	_, err = ValuesOrDefaults(vals, &Bundle{}, "install")
	is.NoError(err)
}

func TestValuesOrDefaults_NoParameter(t *testing.T) {
	is := assert.New(t)
	vals := map[string]interface{}{}
	b := &Bundle{}
	vod, err := ValuesOrDefaults(vals, b, "install")
	is.NoError(err)
	is.Len(vod, 0)
}

func TestValuesOrDefaults_Required(t *testing.T) {
	is := assert.New(t)
	vals := map[string]interface{}{
		"enabled": true,
	}
	b := &Bundle{
		Definitions: map[string]*definition.Schema{
			"minType": {
				Type: "integer",
			},
			"enabledType": {
				Type:    "boolean",
				Default: false,
			},
		},
		Parameters: map[string]Parameter{
			"minimum": {
				Definition: "minType",
				Required:   true,
			},
			"enabled": {
				Definition: "enabledType",
			},
		},
	}

	_, err := ValuesOrDefaults(vals, b, "install")
	is.Error(err)

	// It is unclear what the outcome should be when the user supplies
	// empty values on purpose. For now, we will assume those meet the
	// minimum definition of "required", and that other rules will
	// correct for empty values.
	//
	// Example: It makes perfect sense for a user to specify --set minimum=0
	// and in so doing meet the requirement that a value be specified.
	vals["minimum"] = 0
	res, err := ValuesOrDefaults(vals, b, "install")
	is.NoError(err)
	is.Equal(0, res["minimum"])
}

func TestValuesOrDefaults_NotApplicableToAction(t *testing.T) {
	// vals represent user-supplied parameter values
	vals := map[string]interface{}{
		"param-with-default-and-override": true,
	}

	b := &Bundle{
		Definitions: map[string]*definition.Schema{
			"param-with-default-not-applicable": {
				Type:    "string",
				Default: "foo",
			},
			"required-param-not-applicable": {
				Type: "string",
			},
			"param-with-default": {
				Type:    "boolean",
				Default: false,
			},
			"param-with-default-and-override": {
				Type:    "boolean",
				Default: false,
			},
		},
		Parameters: map[string]Parameter{
			"param-with-default-not-applicable": {
				Definition: "param-with-default-not-applicable",
				ApplyTo: []string{
					"uninstall",
				},
			},
			"required-param-not-applicable": {
				Definition: "required-param-not-applicable",
				Required:   true,
				ApplyTo: []string{
					"uninstall",
				},
			},
			"param-with-default": {
				Definition: "param-with-default",
			},
			"param-with-default-and-override": {
				Definition: "param-with-default-and-override",
			},
		},
	}

	res, err := ValuesOrDefaults(vals, b, "install")
	require.NoError(t, err)

	expected := map[string]interface{}{
		"param-with-default":              false,
		"param-with-default-and-override": true,
	}
	require.Equal(t, expected, res)
}

func TestValuesOrDefaults_DefaultFailsValidation(t *testing.T) {
	is := assert.New(t)

	b := &Bundle{
		Definitions: map[string]*definition.Schema{
			"param": {
				Type:    "boolean",
				Default: "notaboolean",
			},
		},
		Parameters: map[string]Parameter{
			"param": {
				Definition: "param",
			},
		},
	}

	_, err := ValuesOrDefaults(map[string]interface{}{}, b, "install")
	is.Error(err)
	is.Contains(err.Error(), "cannot use value")
	is.Contains(err.Error(), "type should be boolean")
}

func TestValidateVersionTag(t *testing.T) {
	is := assert.New(t)

	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "latest",
		SchemaVersion:    "99.98",
		InvocationImages: []InvocationImage{img},
	}

	err := b.Validate()
	is.EqualError(err, "'latest' is not a valid bundle version")
}

func TestValidateSchemaVersion(t *testing.T) {
	is := assert.New(t)

	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "0.1.0",
		SchemaVersion:    "99.98",
		InvocationImages: []InvocationImage{img},
	}

	err := b.Validate()
	is.Nil(err, "valid bundle schema failed to validate")
}

func TestValidateSchemaVersionWithPrefix(t *testing.T) {
	is := assert.New(t)

	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "0.1.0",
		SchemaVersion:    "v99.98",
		InvocationImages: []InvocationImage{img},
	}

	err := b.Validate()
	is.Nil(err, "valid bundle schema failed to validate")
}

func TestValidateMissingSchemaVersion(t *testing.T) {
	is := assert.New(t)

	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "0.1.0",
		InvocationImages: []InvocationImage{img},
	}

	err := b.Validate()
	is.EqualError(err, "bundle validation failed: invalid schema version \"\": Invalid Semantic Version")
}

func TestValidateInvalidSchemaVersion(t *testing.T) {
	is := assert.New(t)

	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "0.1.0",
		SchemaVersion:    ".1",
		InvocationImages: []InvocationImage{img},
	}

	err := b.Validate()
	is.EqualError(err, "bundle validation failed: invalid schema version \".1\": Invalid Semantic Version")
}

func TestValidateBundle_RequiresInvocationImage(t *testing.T) {
	b := Bundle{
		Name:          "bar",
		SchemaVersion: "99.98",
		Version:       "0.1.0",
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

func TestValidateRequiredExtensions(t *testing.T) {
	is := assert.New(t)

	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "0.1.0",
		SchemaVersion:    "99.98",
		InvocationImages: []InvocationImage{img},
		RequiredExtensions: []string{
			"my.custom.extension",
		},
	}

	// Verify the error when a required extension is not present in custom
	err := b.Validate()
	is.EqualError(err, "required extension 'my.custom.extension' is not defined in the Custom section of the bundle")

	// Add corresponding entry in custom
	b.Custom = map[string]interface{}{
		"my.custom.extension": true,
	}

	err = b.Validate()
	is.NoError(err)

	// Add duplicate required extension
	b.RequiredExtensions = append(b.RequiredExtensions, "my.custom.extension")

	err = b.Validate()
	is.EqualError(err, "required extension 'my.custom.extension' is already declared")
}

func TestValidateParameters(t *testing.T) {
	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "0.1.0",
		SchemaVersion:    "99.98",
		InvocationImages: []InvocationImage{img},
		Definitions: definition.Definitions{
			"param": &definition.Schema{},
		},
	}

	t.Run("bad parameter fails", func(t *testing.T) {
		b.Parameters = map[string]Parameter{
			"badParam": {},
		}

		err := b.Validate()
		assert.EqualError(t, err, `validation failed for parameter "badParam": parameter definition must be provided`)
	})

	t.Run("successful validation", func(t *testing.T) {
		b.Parameters = map[string]Parameter{
			"param": {
				Definition:  "param",
				Destination: &Location{Path: "/path/to/param"},
			},
		}

		err := b.Validate()
		require.NoError(t, err, "bundle parameter validation should succeed")
	})
}

func TestValidateCredentials(t *testing.T) {
	img := InvocationImage{BaseImage{}}
	b := Bundle{
		Version:          "0.1.0",
		SchemaVersion:    "99.98",
		InvocationImages: []InvocationImage{img},
	}

	t.Run("bad credential fails", func(t *testing.T) {
		b.Credentials = map[string]Credential{
			"badCred": {},
		}

		err := b.Validate()
		assert.EqualError(t, err, `validation failed for credential "badCred": credential env or path must be supplied`)
	})

	t.Run("successful validation", func(t *testing.T) {
		b.Credentials = map[string]Credential{
			"cred": {
				Location: Location{Path: "/path/to/cred"},
			},
		}

		err := b.Validate()
		require.NoError(t, err, "bundle credential validation should succeed")
	})
}

func TestReadCustomAndRequiredExtensions(t *testing.T) {
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

	if len(bundle.RequiredExtensions) != 1 {
		t.Errorf("Expected 1 required extension, got %d", len(bundle.RequiredExtensions))
	}
	assert.Equal(t, "com.example.duffle-bag", bundle.RequiredExtensions[0])
}

func TestOutputs_Marshall(t *testing.T) {
	bundleJSON := `
	{ 
		"outputs":{ 
		   "clientCert":{ 
			  "contentEncoding":"base64",
			  "contentMediaType":"application/x-x509-user-cert",
			  "path":"/cnab/app/outputs/clientCert",
			  "definition":"clientCert"
		   },
		   "hostName":{ 
			  "applyTo":[ 
				 "install"
			  ],
			  "description":"the hostname produced installing the bundle",
			  "path":"/cnab/app/outputs/hostname",
			  "definition":"hostType"
		   },
		   "port":{ 
			  "path":"/cnab/app/outputs/port",
			  "definition":"portType"
		   }
		}
	 }`

	bundle, err := Unmarshal([]byte(bundleJSON))
	assert.NoError(t, err, "should have unmarshalled the bundle")
	require.NotNil(t, bundle.Outputs, "test must fail, not outputs found")
	assert.Equal(t, 3, len(bundle.Outputs))

	clientCert, ok := bundle.Outputs["clientCert"]
	require.True(t, ok, "expected clientCert to exist as an output")
	assert.Equal(t, "clientCert", clientCert.Definition)
	assert.Equal(t, "/cnab/app/outputs/clientCert", clientCert.Path, "clientCert path was not the expected value")

	hostName, ok := bundle.Outputs["hostName"]
	require.True(t, ok, "expected hostname to exist as an output")
	assert.Equal(t, "hostType", hostName.Definition)
	assert.Equal(t, "/cnab/app/outputs/hostname", hostName.Path, "hostName path was not the expected value")

	port, ok := bundle.Outputs["port"]
	require.True(t, ok, "expected port to exist as an output")
	assert.Equal(t, "portType", port.Definition)
	assert.Equal(t, "/cnab/app/outputs/port", port.Path, "port path was not the expected value")
}

var exampleCred = Credential{
	Description: "a password",
	Location: Location{
		EnvironmentVariable: "PASSWORD",
		Path:                "/cnab/app/path",
	},
}

var exampleBundle *Bundle = &Bundle{
	SchemaVersion: "1.0.1",
	Name:          "testBundle",
	Description:   "something",
	Version:       "1.0",
	License:       "MIT License",
	Credentials: map[string]Credential{
		"password": exampleCred,
	},
	Images: map[string]Image{
		"server": {
			BaseImage: BaseImage{
				Image:     "nginx:1.0",
				ImageType: "docker",
			},
			Description: "complicated",
		},
	},
	InvocationImages: []InvocationImage{
		{
			BaseImage: BaseImage{
				Image:     "cnabio/invocation-image:1.0",
				ImageType: "docker",
				Labels: map[string]string{
					"os": "Linux",
				},
			},
		},
	},
	Definitions: map[string]*definition.Schema{
		"portType": {
			Type:    "integer",
			Default: 1234,
		},
		"hostType": {
			Type:    "string",
			Default: "locahost.localdomain",
		},
		"replicaCountType": {
			Type:    "integer",
			Default: 3,
		},
		"enabledType": {
			Type:    "boolean",
			Default: false,
		},
		"clientCert": {
			Type:            "string",
			ContentEncoding: "base64",
		},
		"productKeyType": {
			Type: "string",
		},
		"sizeType": {
			Type:    "number",
			Default: 5.5,
			Minimum: float64Ptr(3.5),
			Maximum: float64Ptr(10.1),
		},
	},
	Parameters: map[string]Parameter{
		"port": {
			Definition: "portType",
			Destination: &Location{
				EnvironmentVariable: "PORT",
				Path:                "/path/to/port",
			},
			Required: true,
		},
		"host": {
			Definition: "hostType",
			Destination: &Location{
				EnvironmentVariable: "HOST",
			},
			Required: true,
		},
		"enabled": {
			Definition: "enabledType",
			Destination: &Location{
				EnvironmentVariable: "ENABLED",
			},
		},
		"replicaCount": {
			Definition: "replicaCountType",
			Destination: &Location{
				EnvironmentVariable: "REPLICA_COUNT",
			},
		},
		"productKey": {
			Definition: "productKeyType",
			Destination: &Location{
				EnvironmentVariable: "PRODUCT_KEY",
			},
		},
		"size": {
			Definition: "sizeType",
			Destination: &Location{
				EnvironmentVariable: "SIZE",
			},
		},
	},
	Outputs: map[string]Output{
		"clientCert": {
			Path:       "/cnab/app/outputs/blah",
			Definition: "clientCert",
		},
	},
}

func float64Ptr(value float64) *float64 {
	return &value
}

func TestValidateExampleBundle(t *testing.T) {
	err := exampleBundle.Validate()
	require.NoError(t, err, "example bundle validation should succeed")
}

func TestBundleMarshallAllThings(t *testing.T) {
	expectedJSON, err := ioutil.ReadFile("../testdata/bundles/canonical-bundle.json")
	require.NoError(t, err, "couldn't read test data")

	var buf bytes.Buffer

	_, err = exampleBundle.WriteTo(&buf)
	require.NoError(t, err, "test requires output")
	assert.Equal(t, string(expectedJSON), buf.String(), "output should match expected canonical json")
}

func TestBundleYamlRoundtrip(t *testing.T) {
	bytes, err := yaml.Marshal(exampleBundle)
	require.NoError(t, err, "should have been able to yaml.Marshal bundle")

	expectedYAML, err := ioutil.ReadFile("../testdata/bundles/bundle.yaml")
	require.NoError(t, err, "couldn't read test data")

	assert.Equal(t, string(expectedYAML), string(bytes), "marshaled bytes should match expected yaml representation")

	var roundTripBun Bundle
	err = yaml.UnmarshalStrict(bytes, &roundTripBun)
	require.NoError(t, err, "should have been able to yaml.UnmarshalStrict bundle")

	assert.Equal(t, exampleBundle, &roundTripBun, "after a roundtrip yaml marshal/unmarshal, the bundle does not match expected")
}

func TestValidateABundleAndParams(t *testing.T) {

	bun, err := ioutil.ReadFile("../testdata/bundles/foo.json")
	require.NoError(t, err, "couldn't read test bundle")

	bundle, err := Unmarshal(bun)
	require.NoError(t, err, "the bundle should have been valid")

	def, ok := bundle.Definitions["complexThing"]
	require.True(t, ok, "test failed because definition not found")

	testData := struct {
		Port int    `json:"port"`
		Host string `json:"hostName"`
	}{
		Host: "validhost",
		Port: 8080,
	}
	valErrors, err := def.Validate(testData)
	assert.NoError(t, err, "validation should not have resulted in an error")
	assert.Empty(t, valErrors, "validation should have been successful")

	testData2 := struct {
		Host string `json:"hostName"`
	}{
		Host: "validhost",
	}
	valErrors, err = def.Validate(testData2)
	assert.NoError(t, err, "validation should not have encountered an error")
	assert.NotEmpty(t, valErrors, "validation should not have been successful")

	testData3 := struct {
		Port int    `json:"port"`
		Host string `json:"hostName"`
	}{
		Host: "validhost",
		Port: 80,
	}
	valErrors, err = def.Validate(testData3)
	assert.NoError(t, err, "should not have encountered an error with the validator")
	assert.NotEmpty(t, valErrors, "validation should not have been successful")
}

func TestBundle_RoundTrip(t *testing.T) {
	testCases := []struct {
		name     string
		testFile string
	}{
		{name: "EmptyJson", testFile: "testdata/empty.json"},
		{name: "MinimalJson", testFile: "testdata/minimal.json"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wantData, err := ioutil.ReadFile(tc.testFile)
			if err != nil {
				t.Fatal(err)
			}

			bun, err := Unmarshal(wantData)
			if err != nil {
				t.Fatal(err)
			}

			output := &bytes.Buffer{}
			_, err = bun.WriteTo(output)
			require.NoError(t, err, "writing the bundle to json failed")

			gotData := output.String()
			assert.Equal(t, string(wantData), gotData)
		})
	}
}

func TestDigestPresent(t *testing.T) {
	bun, err := ioutil.ReadFile("../testdata/bundles/digest.json")
	require.NoError(t, err, "couldn't read test bundle")

	bundle, err := Unmarshal(bun)
	require.NoError(t, err, "the bundle should have been valid")

	require.Equal(t, 1, len(bundle.InvocationImages), "there should be one invocation image in the bundle")
	assert.Equal(t,
		"sha256:decafbad71b4175951f29eb96035604c8cc372c99affa2e6d05cde6e8e20cc9a",
		bundle.InvocationImages[0].Digest,
	)

	image, ok := bundle.Images["my-microservice"]
	require.True(t, ok, "there should been an image named my-microservice in the bundle")
	assert.Equal(
		t,
		"sha256:beefcacef6c04336a17761db2004813982abe0e87ab727a376c291e09391ea61",
		image.Digest,
	)
}

func TestImageDeepCopy(t *testing.T) {
	origImg := Image{
		Description: "my image",
		BaseImage: BaseImage{
			Image:     "alpine",
			ImageType: "docker",
			Labels: map[string]string{
				"origLabel": "origLabelValue",
			},
			Digest: "abc1234",
			Size:   2,
		},
	}

	newImg := origImg.DeepCopy()

	newImg.Description = "my new image"
	newImg.Image = "debian"
	newImg.Labels["origLabel"] = "newLabelValue"
	newImg.Digest = "123abcd"

	assert.Equal(t, "my image", origImg.Description)
	assert.Equal(t, "alpine", origImg.Image)
	assert.Equal(t, map[string]string{"origLabel": "origLabelValue"}, origImg.Labels)
	assert.Equal(t, "abc1234", origImg.Digest)

	assert.Equal(t, "my new image", newImg.Description)
	assert.Equal(t, "debian", newImg.Image)
	assert.Equal(t, map[string]string{"origLabel": "newLabelValue"}, newImg.Labels)
	assert.Equal(t, "123abcd", newImg.Digest)
}

func TestValidateLocation(t *testing.T) {
	testCases := []struct {
		name     string
		location Location
		err      string
	}{{
		name:     "no path",
		location: Location{},
	}, {
		name:     "ok path",
		location: Location{Path: "/path/to/thing"},
	}, {
		name:     "error path",
		location: Location{Path: "/cnab/app/outputs/thing"},
		err:      `Path "/cnab/app/outputs/thing" must not be a subpath of "/cnab/app/outputs"`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.location.Validate()

			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBundleSchema(t *testing.T) {
	bunBytes, err := json.Marshal(exampleBundle)
	assert.NoError(t, err, "failed to json.Marshal the bundle")

	valErrors, err := schema.ValidateBundle(bunBytes)
	assert.NoError(t, err, "failed to validate bundle schema")

	if len(valErrors) > 0 {
		t.Log("bundle validation against the JSON schema failed:")
		for _, error := range valErrors {
			t.Log(error)
		}
		t.Fail()
	}
}

func TestBundle_IsOutputSensitive(t *testing.T) {
	var writeOnly = true
	b := Bundle{
		Definitions: map[string]*definition.Schema{
			"port": {
				Type: "integer",
			},
			"password": {
				Type:      "string",
				WriteOnly: &writeOnly,
			},
		},
		Outputs: map[string]Output{
			"port": {
				Definition: "port",
			},
			"password": {
				Definition: "password",
			},
			"no-def": {
				Definition: "no-def",
			},
		},
	}

	t.Run("write-only unset", func(t *testing.T) {
		sensitive, err := b.IsOutputSensitive("port")
		require.NoError(t, err, "IsOutputSensitive failed")
		assert.False(t, sensitive, "expected port to NOT be sensitive because write-only is false")
	})

	t.Run("write-only true", func(t *testing.T) {
		sensitive, err := b.IsOutputSensitive("password")
		require.NoError(t, err, "IsOutputSensitive failed")
		assert.True(t, sensitive, "expected password to be sensitive because write-only is true")
	})

	t.Run("missing output", func(t *testing.T) {
		_, err := b.IsOutputSensitive("no-output")
		require.EqualError(t, err, `output "no-output" not defined`)
	})

	t.Run("missing definition", func(t *testing.T) {
		_, err := b.IsOutputSensitive("no-def")
		require.EqualError(t, err, `output definition "no-def" not found`)
	})

}
