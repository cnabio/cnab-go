// +build windows

package command

import (
	"os"
	"os/exec"
)

// CheckDriverExists checks to see if the named driver exists
func (d *Driver) CheckDriverExists() bool {
	if d.Path != "" {
		_, err := os.Stat(d.Path)
		return err == nil
	}

	cmd := exec.Command("where", d.cmd())
	err := cmd.Run()
	return err == nil
}
