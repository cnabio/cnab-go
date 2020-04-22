package docker

import (
	"testing"

	"github.com/cnabio/cnab-go/driver"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

func TestDriver_GetConfigurationOptions(t *testing.T) {
	d := &Driver{}
	is := assert.New(t)
	is.NotNil(d)
	is.True(d.Handles(driver.ImageTypeDocker))

	t.Run("empty configuration options", func(t *testing.T) {
		err := d.ApplyConfigurationOptions()
		is.NoError(err)

		cfg := d.GetContainerConfig()
		is.Equal(container.Config{}, cfg)

		hostCfg := d.GetContainerHostConfig()
		is.Equal(container.HostConfig{}, hostCfg)
	})

	t.Run("configuration options", func(t *testing.T) {
		d.AddConfigurationOptions(func(cfg *container.Config, hostCfg *container.HostConfig) error {
			cfg.User = "cnabby"
			hostCfg.Privileged = true
			return nil
		})

		err := d.ApplyConfigurationOptions()
		is.NoError(err)

		expectedContainerCfg := container.Config{
			User: "cnabby",
		}
		expectedHostCfg := container.HostConfig{
			Privileged: true,
		}

		cfg := d.GetContainerConfig()
		is.Equal(expectedContainerCfg, cfg)

		hostCfg := d.GetContainerHostConfig()
		is.Equal(expectedHostCfg, hostCfg)
	})
}
