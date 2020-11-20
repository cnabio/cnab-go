// +build !windows

package command

import (
	"fmt"
	"os"
	"os/exec"
)

// CheckDriverExists checks to see if the named driver exists
func (d *Driver) CheckDriverExists() bool {
	if d.Path != "" {
		_, err := os.Stat(d.Path)
		return err == nil
	}

	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("command -v %s", d.cmd()))
	err := cmd.Run()
	return err == nil
}
