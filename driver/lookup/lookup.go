package lookup

import (
	"fmt"

	"github.com/cnabio/cnab-go/driver"
	"github.com/cnabio/cnab-go/driver/command"
	"github.com/cnabio/cnab-go/driver/docker"
	"github.com/cnabio/cnab-go/driver/kubernetes"
)

// Lookup takes a driver name and tries to resolve the most pertinent driver.
func Lookup(name string) (driver.Driver, error) {
	switch name {
	case "docker":
		return &docker.Driver{}, nil
	case "kubernetes", "k8s":
		return &kubernetes.Driver{}, nil
	case "debug":
		return &driver.DebugDriver{}, nil
	default:
		cmddriver := &command.Driver{Name: name}
		if cmddriver.CheckDriverExists() {
			return cmddriver, nil
		}

		return nil, fmt.Errorf("unsupported driver or driver not found in PATH: %s", name)
	}
}
