package action_test

import (
	"fmt"
	"time"

	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/driver"
	"github.com/cnabio/cnab-go/driver/lookup"
	"github.com/cnabio/cnab-go/valuesource"
)

// Install the bundle and only record the success/failure of the operation.
func Example_install() {
	// Use the debug driver to only print debug information about the bundle but not actually execute it
	// Use "docker" to execute it for real
	d, err := lookup.Lookup("debug")
	if err != nil {
		panic(err)
	}

	// Create the action that will execute the operation
	a := action.New(d)

	// Get the definition of the bundle, usually you get this by pulling
	// the bundle from an OCI registry using github.com/cnabio/cnab-to-oci
	b := bundle.Bundle{
		SchemaVersion: bundle.GetDefaultSchemaVersion(),
		Name:          "mybuns",
		Version:       "1.0.0",
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					ImageType: driver.ImageTypeDocker,
					Image:     "example.com/myorg/myinstaller",
					Digest:    "sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474",
				},
			},
		},
		Actions: map[string]bundle.Action{
			"logs": {Modifies: false, Stateless: false},
		},
	}

	// Pass an empty set of parameters
	var parameters map[string]interface{}

	// Create a claim representing the inputs to the install operation
	c, err := claim.New("hello", claim.ActionInstall, b, parameters)
	if err != nil {
		panic(err)
	}

	// Set to consistent values so we can compare output reliably
	c.ID = "claim-id"
	c.Revision = "claim-rev"
	c.Created = time.Date(2020, time.April, 18, 1, 2, 3, 4, time.UTC)

	// Pass an empty set of credentials
	var creds valuesource.Set

	opResult, claimResult, err := a.Run(c, creds)
	if err != nil {
		// Something terrible has occurred and we could not even run the bundle
		panic(err)
	}

	// When there is no error, then both opResult and claimResult are populated
	if opResult.Error != nil {
		// The bundle ran but there was an error during execution
		fmt.Println("WARNING: bundle execution was unsuccessful")
	}
	// TODO: Persist claimResult
	fmt.Println("status:", claimResult.Status)

	// Output: {
	//   "installation_name": "hello",
	//   "revision": "claim-rev",
	//   "action": "install",
	//   "parameters": null,
	//   "image": {
	//     "imageType": "docker",
	//     "image": "example.com/myorg/myinstaller",
	//     "contentDigest": "sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474"
	//   },
	//   "environment": {
	//     "CNAB_ACTION": "install",
	//     "CNAB_BUNDLE_NAME": "mybuns",
	//     "CNAB_BUNDLE_VERSION": "1.0.0",
	//     "CNAB_CLAIMS_VERSION": "1.0.0-DRAFT+b5ed2f3",
	//     "CNAB_INSTALLATION_NAME": "hello",
	//     "CNAB_REVISION": "claim-rev"
	//   },
	//   "files": {
	//     "/cnab/app/image-map.json": "{}",
	//     "/cnab/bundle.json": "{\"schemaVersion\":\"1.2.0\",\"name\":\"mybuns\",\"version\":\"1.0.0\",\"description\":\"\",\"invocationImages\":[{\"imageType\":\"docker\",\"image\":\"example.com/myorg/myinstaller\",\"contentDigest\":\"sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474\"}],\"actions\":{\"logs\":{}}}",
	//     "/cnab/claim.json": "{\"schemaVersion\":\"1.0.0-DRAFT+b5ed2f3\",\"id\":\"claim-id\",\"installation\":\"hello\",\"revision\":\"claim-rev\",\"created\":\"2020-04-18T01:02:03.000000004Z\",\"action\":\"install\",\"bundle\":{\"schemaVersion\":\"1.2.0\",\"name\":\"mybuns\",\"version\":\"1.0.0\",\"description\":\"\",\"invocationImages\":[{\"imageType\":\"docker\",\"image\":\"example.com/myorg/myinstaller\",\"contentDigest\":\"sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474\"}],\"actions\":{\"logs\":{}}}}"
	//   },
	//   "outputs": {},
	//   "Bundle": {
	//     "schemaVersion": "1.2.0",
	//     "name": "mybuns",
	//     "version": "1.0.0",
	//     "description": "",
	//     "invocationImages": [
	//       {
	//         "imageType": "docker",
	//         "image": "example.com/myorg/myinstaller",
	//         "contentDigest": "sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474"
	//       }
	//     ],
	//     "actions": {
	//       "logs": {}
	//     }
	//   }
	// }
	// status: succeeded
}
