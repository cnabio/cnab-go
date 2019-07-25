package docker

import (
	"bytes"
	"testing"

	"github.com/deislabs/cnab-go/driver"
	"github.com/stretchr/testify/assert"
)

func TestDriver_Run(t *testing.T) {
	op := &driver.Operation{
		Installation: "example",
		Action:       "install",
		Image:        "pvtlmc/example-outputs@sha256:ae3c1c719d87e981ecfdf84862ff389ce6a1281c99372f45d158f52e3d5be3dc",
		Outputs:      []string{"/cnab/app/outputs/output1", "/cnab/app/outputs/output2"},
	}

	var output bytes.Buffer
	op.Out = &output
	op.Environment = map[string]string{}
	op.Environment["CNAB_ACTION"] = op.Action
	op.Environment["CNAB_INSTALLATION_NAME"] = op.Installation

	docker := &Driver{}
	docker.SetContainerOut(op.Out) // Docker driver writes container stdout to driver.containerOut.
	opResult, err := docker.Run(op)

	assert.NoError(t, err)
	assert.Equal(t, "Install action\nAction install complete for example\n", output.String())
	assert.Equal(t, 2, len(opResult.Outputs), "Expecting two output files")
	assert.Equal(t, map[string]string{
		"/cnab/app/outputs/output1": "SOME INSTALL CONTENT 1\n",
		"/cnab/app/outputs/output2": "SOME INSTALL CONTENT 2\n",
	}, opResult.Outputs)
}
