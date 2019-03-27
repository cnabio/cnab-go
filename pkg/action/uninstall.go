package action

import (
	"io"

	"github.com/radu-matei/cnab-go/pkg/claim"
	"github.com/radu-matei/cnab-go/pkg/credentials"
	"github.com/radu-matei/cnab-go/pkg/driver"
)

// Uninstall runs an uninstall action
type Uninstall struct {
	Driver driver.Driver
}

// Run performs the uninstall steps and updates the Claim
func (u *Uninstall) Run(c *claim.Claim, creds credentials.Set, w io.Writer) error {
	invocImage, err := selectInvocationImage(u.Driver, c)
	if err != nil {
		return err
	}

	op, err := opFromClaim(claim.ActionUninstall, notStateless, c, invocImage, creds, w)
	if err != nil {
		return err
	}
	if err := u.Driver.Run(op); err != nil {
		c.Update(claim.ActionUninstall, claim.StatusFailure)
		c.Result.Message = err.Error()
		return err
	}

	c.Update(claim.ActionUninstall, claim.StatusSuccess)
	return nil
}
