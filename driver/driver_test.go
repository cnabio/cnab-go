package driver

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ Driver = &DebugDriver{}

func TestDebugDriver_Handles(t *testing.T) {
	d := &DebugDriver{}
	is := assert.New(t)
	is.NotNil(d)
	is.True(d.Handles(ImageTypeDocker))
	is.True(d.Handles("anything"))
}

func TestDebugDriver_Run(t *testing.T) {
	d := &DebugDriver{}
	is := assert.New(t)
	is.NotNil(d)

	op := &Operation{
		Installation: "test",
		Image:        "test:1.2.3",
		ImageType:    "oci",
		Out:          ioutil.Discard,
	}
	is.NoError(d.Run(op))
}
