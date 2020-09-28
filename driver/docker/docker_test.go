package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/stretchr/testify/assert"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/driver"
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

func TestDriver_ValidateImageDigest(t *testing.T) {
	repoDigests := []string{
		"myreg/myimg@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a",
	}

	t.Run("no image digest", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/myimg"

		err := d.validateImageDigest(image, repoDigests)
		assert.NoError(t, err)
	})

	t.Run("image digest exists - no match exists", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/myimg"
		image.Digest = "sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321"

		err := d.validateImageDigest(image, repoDigests)
		assert.NotNil(t, err, "expected an error")
		assert.Contains(t, err.Error(), "content digest mismatch")
	})

	t.Run("image digest exists - a match exists", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/myimg"
		image.Digest = "sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a"

		err := d.validateImageDigest(image, repoDigests)
		assert.NoError(t, err)
	})

	t.Run("image digest exists - more than one repo digest exists", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/myimg"
		image.Digest = "sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a"

		repoDigests = append(repoDigests,
			"myreg/myimg@sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321")

		err := d.validateImageDigest(image, repoDigests)
		assert.NotNil(t, err, "expected an error")
		assert.EqualError(t, err, "image myreg/myimg has more than one repo digest")
	})
}
