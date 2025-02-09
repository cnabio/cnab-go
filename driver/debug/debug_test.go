package debug

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/driver"
)

var _ driver.Driver = &Driver{}

func TestDebugDriver_Handles(t *testing.T) {
	d := &Driver{}
	is := assert.New(t)
	is.NotNil(d)
	is.True(d.Handles(driver.ImageTypeDocker))
	is.True(d.Handles("anything"))
}

func TestDebugDriver_Run(t *testing.T) {
	d := &Driver{}
	is := assert.New(t)
	is.NotNil(d)

	op := &driver.Operation{
		Installation: "test",
		Image: bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image:     "test:1.2.3",
				ImageType: "oci",
			},
		},
		Out: ioutil.Discard,
	}

	_, err := d.Run(context.Background(), op)
	is.NoError(err)
}
