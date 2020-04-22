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

	t.Run("no configuration options", func(t *testing.T) {
		d.containerCfg = &container.Config{}
		d.containerHostCfg = &container.HostConfig{}

		err := d.ApplyConfigurationOptions()
		is.NoError(err)

		cfg, err := d.GetContainerConfig()
		is.NoError(err)
		is.Equal(container.Config{}, cfg)

		hostCfg, err := d.GetContainerHostConfig()
		is.NoError(err)
		is.Equal(container.HostConfig{}, hostCfg)
	})

	t.Run("configuration options", func(t *testing.T) {
		d.containerCfg = &container.Config{}
		d.containerHostCfg = &container.HostConfig{}
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

		cfg, err := d.GetContainerConfig()
		is.NoError(err)
		is.Equal(expectedContainerCfg, cfg)

		hostCfg, err := d.GetContainerHostConfig()
		is.NoError(err)
		is.Equal(expectedHostCfg, hostCfg)
	})
}
