package action_test

import (
	"fmt"
	"time"

	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/driver/lookup"
	"github.com/cnabio/cnab-go/valuesource"
)

// Invoke the bundle and only record the success/failure of the operation.
func Example_invoke() {
	// Use the debug driver to only print debug information about the bundle but not actually execute it
	// Use "docker" to execute it for real
	d, err := lookup.Lookup("debug")
	if err != nil {
		panic(err)
	}

	// Pass an empty set of parameters
	var parameters map[string]interface{}

	// Load the previous claim for the installation
	existingClaim := createInstallClaim()
	if err != nil {
		panic(err)
	}

	// Create a claim for running the custom logs action based on the previous claim
	c, err := existingClaim.NewClaim("logs", existingClaim.Bundle, parameters)
	if err != nil {
		panic(err)
	}

	// Set to consistent values so we can compare output reliably
	c.ID = "claim-id"
	c.Revision = "claim-rev"
	c.Created = time.Date(2020, time.April, 18, 1, 2, 3, 4, time.UTC)

	// Create the action that will execute the operation
	a := action.New(d)

	// Determine if the action modifies the bundle's resources or if it doesn't
	// For example like "logs", or "dry-run" would have modifies = false
	modifies, err := c.IsModifyingAction()
	if err != nil {
		panic(err)
	}

	// Pass an empty set of credentials
	var creds valuesource.Set

	opResult, claimResult, err := a.Run(c, creds)
	if err != nil {
		// Something terrible has occurred and we could not even run the bundle
		panic(err)
	}

	// Only record the result of the operation when operation modifies resources
	if modifies {
		// Only save the claim if it modified the bundle's resources
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
	//   "action": "logs",
	//   "parameters": null,
	//   "image": {
	//     "imageType": "docker",
	//     "image": "example.com/myorg/myinstaller",
	//     "contentDigest": "sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474"
	//   },
	//   "environment": {
	//     "CNAB_ACTION": "logs",
	//     "CNAB_BUNDLE_NAME": "mybuns",
	//     "CNAB_BUNDLE_VERSION": "1.0.0",
	//     "CNAB_CLAIMS_VERSION": "1.0.0-DRAFT+b5ed2f3",
	//     "CNAB_INSTALLATION_NAME": "hello",
	//     "CNAB_REVISION": "claim-rev"
	//   },
	//   "files": {
	//     "/cnab/app/image-map.json": "{}",
	//     "/cnab/bundle.json": "{\"schemaVersion\":\"1.2.0\",\"name\":\"mybuns\",\"version\":\"1.0.0\",\"description\":\"\",\"invocationImages\":[{\"imageType\":\"docker\",\"image\":\"example.com/myorg/myinstaller\",\"contentDigest\":\"sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474\"}],\"actions\":{\"logs\":{}}}",
	//     "/cnab/claim.json": "{\"schemaVersion\":\"1.0.0-DRAFT+b5ed2f3\",\"id\":\"claim-id\",\"installation\":\"hello\",\"revision\":\"claim-rev\",\"created\":\"2020-04-18T01:02:03.000000004Z\",\"action\":\"logs\",\"bundle\":{\"schemaVersion\":\"1.2.0\",\"name\":\"mybuns\",\"version\":\"1.0.0\",\"description\":\"\",\"invocationImages\":[{\"imageType\":\"docker\",\"image\":\"example.com/myorg/myinstaller\",\"contentDigest\":\"sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474\"}],\"actions\":{\"logs\":{}}}}"
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
