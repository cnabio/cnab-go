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
		"myreg/myimg@sha256:deadbeef",
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
		image.Digest = "sha256:livebeef"

		err := d.validateImageDigest(image, repoDigests)
		assert.EqualError(t, err,
			"content digest mismatch: image myreg/myimg has digest(s) [sha256:deadbeef] but the digest should be sha256:livebeef according to the bundle file")
	})

	t.Run("image digest exists - a match exists", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/myimg"
		image.Digest = "sha256:deadbeef"

		err := d.validateImageDigest(image, repoDigests)
		assert.NoError(t, err)
	})
}
