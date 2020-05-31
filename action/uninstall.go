package action

import (
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/driver"
	"github.com/cnabio/cnab-go/valuesource"
)

// Uninstall runs an uninstall action
type Uninstall struct {
	Driver driver.Driver
}

// Run performs the uninstall steps and updates the Claim
func (u *Uninstall) Run(c *claim.Claim, creds valuesource.Set, opCfgs ...OperationConfigFunc) error {
	invocImage, err := selectInvocationImage(u.Driver, c)
	if err != nil {
		return err
	}

	op, err := opFromClaim(claim.ActionUninstall, stateful, c, invocImage, creds)
	if err != nil {
		return err
	}

	err = OperationConfigs(opCfgs).ApplyConfig(op)
	if err != nil {
		return err
	}

	opResult, err := u.Driver.Run(op)
	outputErrors := setOutputsOnClaim(c, opResult.Outputs)

	if err != nil {
		c.Update(claim.ActionUninstall, claim.StatusFailed)
		c.Result.Message = err.Error()
		return err
	}
	c.Update(claim.ActionUninstall, claim.StatusSucceeded)

	return outputErrors
}
