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

		err := d.applyConfigurationOptions()
		is.Nil(err)

		cfg, hostCfg := d.GetContainerConfig()
		is.Equal(&container.Config{}, cfg)
		is.Equal(&container.HostConfig{}, hostCfg)
	})

	t.Run("configuration options", func(t *testing.T) {
		d.containerCfg = &container.Config{}
		d.containerHostCfg = &container.HostConfig{}
		d.AddConfigurationOptions(func(cfg *container.Config, hostCfg *container.HostConfig) error {
			cfg.User = "cnabby"
			hostCfg.Privileged = true
			return nil
		})

		err := d.applyConfigurationOptions()
		is.Nil(err)

		expectedContainerCfg := &container.Config{
			User: "cnabby",
		}
		expectedHostCfg := &container.HostConfig{
			Privileged: true,
		}
		cfg, hostCfg := d.GetContainerConfig()
		is.Equal(expectedContainerCfg, cfg)
		is.Equal(expectedHostCfg, hostCfg)
	})
}
