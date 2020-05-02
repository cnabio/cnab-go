package docker

import (
	"testing"

	"github.com/cnabio/cnab-go/driver"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/stretchr/testify/assert"
)

func TestDriver_GetConfigurationOptions(t *testing.T) {
	is := assert.New(t)

	t.Run("empty configuration options", func(t *testing.T) {
		d := &Driver{}
		is.NotNil(d)
		is.True(d.Handles(driver.ImageTypeDocker))

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
		d := &Driver{}

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

	t.Run("configuration options - no unintentional modification", func(t *testing.T) {
		d := &Driver{}

		d.AddConfigurationOptions(func(cfg *container.Config, hostCfg *container.HostConfig) error {
			hostCfg.CapAdd = strslice.StrSlice{"SUPER_POWERS"}
			return nil
		})

		err := d.ApplyConfigurationOptions()
		is.NoError(err)

		expectedHostCfg := container.HostConfig{
			CapAdd: strslice.StrSlice{"SUPER_POWERS"},
		}

		hostCfg, err := d.GetContainerHostConfig()
		is.NoError(err)
		is.Equal(expectedHostCfg, hostCfg)

		hostCfg.CapAdd[0] = "NORMAL_POWERS"

		hostCfg, err = d.GetContainerHostConfig()
		is.NoError(err)
		is.Equal(expectedHostCfg, hostCfg)
	})
}
