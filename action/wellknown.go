package action

import (
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/credentials"
	"github.com/cnabio/cnab-go/driver"
)

// Well known constants define the Well Known CNAB actions to be taken
const (
	ActionDryRun = "io.cnab.dry-run"
	ActionHelp   = "io.cnab.help"
	ActionLog    = "io.cnab.log"
	ActionStatus = "io.cnab.status"
)

// DryRun runs a dry-run action on a CNAB bundle.
type DryRun struct {
	Driver driver.Driver
}

// Help runs a help action on a CNAB bundle.
type Help struct {
	Driver driver.Driver
}

// Log runs a log action on a CNAB bundle.
type Log struct {
	Driver driver.Driver
}

// Status runs a status action on a CNAB bundle.
type Status struct {
	Driver driver.Driver
}

// Run executes a dry-run action in an image
func (i *DryRun) Run(c *claim.Claim, creds credentials.Set, opCfgs ...OperationConfigFunc) error {
	return (&RunCustom{Driver: i.Driver, Action: ActionDryRun}).Run(c, creds, opCfgs...)
}

// Run executes a help action in an image
func (i *Help) Run(c *claim.Claim, creds credentials.Set, opCfgs ...OperationConfigFunc) error {
	return (&RunCustom{Driver: i.Driver, Action: ActionHelp}).Run(c, creds, opCfgs...)
}

// Run executes a log action in an image
func (i *Log) Run(c *claim.Claim, creds credentials.Set, opCfgs ...OperationConfigFunc) error {
	return (&RunCustom{Driver: i.Driver, Action: ActionLog}).Run(c, creds, opCfgs...)
}

// Run executes a status action in an image
func (i *Status) Run(c *claim.Claim, creds credentials.Set, opCfgs ...OperationConfigFunc) error {
	return (&RunCustom{Driver: i.Driver, Action: ActionStatus}).Run(c, creds, opCfgs...)
}
