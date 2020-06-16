package debug

import (
	"bytes"
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

	op := driver.Operation{
		Installation: "test",
		Image: bundle.InvocationImage{
			BaseImage: bundle.BaseImage{
				Image:     "test:1.2.3",
				ImageType: "oci",
			},
		},
	}

	t.Run("success", func(t *testing.T) {
		op.Out = ioutil.Discard

		_, err := d.Run(context.Background(), &op)
		is.NoError(err)
	})

	t.Run("cancelled", func(t *testing.T) {
		output := bytes.Buffer{}
		op.Out = &output

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := d.Run(ctx, &op)
		is.Empty(output.String())
		is.EqualError(err, "context canceled")
	})
}
