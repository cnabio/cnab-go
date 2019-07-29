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
		Image:        "pvtlmc/example-outputs@sha256:568461508c8d220742add8abd226b33534d4269868df4b3178fae1cba3818a6e",
		Outputs:      []string{"/cnab/app/outputs/output1", "/cnab/app/outputs/output2"},
	}

	var output bytes.Buffer
	op.Out = &output
	op.Environment = map[string]string{
		"CNAB_ACTION":            op.Action,
		"CNAB_INSTALLATION_NAME": op.Installation,
	}

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
