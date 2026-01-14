package action_test

import (
	"context"
	"fmt"
	"time"

	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/driver/lookup"
	"github.com/cnabio/cnab-go/valuesource"
)

func saveResult(c claim.Claim, status string) {
	r, err := c.NewResult(status)
	if err != nil {
		panic(err)
	}

	fmt.Println("status:", r.Status)
	// TODO: persist the result
}

// Upgrade the bundle and record the operation as running immediately so that you can
// track how long the operation took.
func Example_runningStatus() {
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

	// Create a claim for the upgrade operation based on the previous claim
	c, err := existingClaim.NewClaim(claim.ActionUpgrade, existingClaim.Bundle, parameters)
	if err != nil {
		panic(err)
	}

	// Set to consistent values so we can compare output reliably
	c.ID = "claim-id"
	c.Revision = "claim-rev"
	c.Created = time.Date(2020, time.April, 18, 1, 2, 3, 4, time.UTC)

	// Create the action that will execute the operation
	a := action.New(d)

	// Pass an empty set of credentials
	var creds valuesource.Set

	// Save the upgrade claim in the Running Status
	saveResult(c, claim.StatusRunning)

	opResult, claimResult, err := a.Run(context.Background(), c, creds)
	if err != nil {
		// If the bundle isn't run due to an error preparing,
		// record a failure so we aren't left stuck in running
		saveResult(c, claim.StatusFailed)
		panic(err)
	}

	// When there is no error, then both opResult and claimResult are populated
	if opResult.Error != nil {
		// The bundle ran but there was an error during execution
		fmt.Println("WARNING: bundle execution was unsuccessful")
	}
	// TODO: Persist claimResult
	fmt.Println("status:", claimResult.Status)

	// Output: status: running
	// {
	//   "installation_name": "hello",
	//   "revision": "claim-rev",
	//   "action": "upgrade",
	//   "parameters": null,
	//   "image": {
	//     "imageType": "docker",
	//     "image": "example.com/myorg/myinstaller",
	//     "contentDigest": "sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474"
	//   },
	//   "environment": {
	//     "CNAB_ACTION": "upgrade",
	//     "CNAB_BUNDLE_NAME": "mybuns",
	//     "CNAB_BUNDLE_VERSION": "1.0.0",
	//     "CNAB_CLAIMS_VERSION": "1.0.0-DRAFT+b5ed2f3",
	//     "CNAB_INSTALLATION_NAME": "hello",
	//     "CNAB_REVISION": "claim-rev"
	//   },
	//   "files": {
	//     "/cnab/app/image-map.json": "{}",
	//     "/cnab/bundle.json": "{\"schemaVersion\":\"1.2.0\",\"name\":\"mybuns\",\"version\":\"1.0.0\",\"description\":\"\",\"invocationImages\":[{\"imageType\":\"docker\",\"image\":\"example.com/myorg/myinstaller\",\"contentDigest\":\"sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474\"}],\"actions\":{\"logs\":{}}}",
	//     "/cnab/claim.json": "{\"schemaVersion\":\"1.0.0-DRAFT+b5ed2f3\",\"id\":\"claim-id\",\"installation\":\"hello\",\"revision\":\"claim-rev\",\"created\":\"2020-04-18T01:02:03.000000004Z\",\"action\":\"upgrade\",\"bundle\":{\"schemaVersion\":\"1.2.0\",\"name\":\"mybuns\",\"version\":\"1.0.0\",\"description\":\"\",\"invocationImages\":[{\"imageType\":\"docker\",\"image\":\"example.com/myorg/myinstaller\",\"contentDigest\":\"sha256:7cc0618539fe11e801ce68911a0c9441a3dfaa9ba63057526c4016cf9db19474\"}],\"actions\":{\"logs\":{}}}}"
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
