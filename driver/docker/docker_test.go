package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestDriver_SetConfig(t *testing.T) {
	testcases := []struct {
		name      string
		settings  map[string]string
		wantError string
	}{
		{
			name: "valid settings",
			settings: map[string]string{
				"VERBOSE": "true",
			},
			wantError: "",
		},
		{
			name: "cleanup containers: true",
			settings: map[string]string{
				"CLEANUP_CONTAINERS": "true",
			},
			wantError: "",
		},
		{
			name: "cleanup containers: false",
			settings: map[string]string{
				"CLEANUP_CONTAINERS": "false",
			},
			wantError: "",
		},
		{
			name: "cleanup containers - invalid",
			settings: map[string]string{
				"CLEANUP_CONTAINERS": "1",
			},
			wantError: "environment variable CLEANUP_CONTAINERS has unexpected value",
		},
	}

	for _, tc := range testcases {
		d := Driver{}
		err := d.SetConfig(tc.settings)

		if tc.wantError == "" {
			require.NoError(t, err, "expected SetConfig to succeed")
			assert.Equal(t, tc.settings, d.config, "expected all of the specified settings to be copied")
		} else {
			require.Error(t, err, "expected SetConfig to fail")
			assert.Contains(t, err.Error(), tc.wantError)
		}
	}
}

func TestDriver_ValidateImageDigest(t *testing.T) {
	// Mimic the digests created when a bundle is pushed with cnab-to-oci
	// there is one for the original invocation image and another
	// for the relocated invocation image inside the bundle repository
	repoDigests := []string{
		"myreg/mybun-installer:v1.0.0",
		"myreg/mybun@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a",
	}

	t.Run("no image digest", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/myimg"

		err := d.validateImageDigest(image, repoDigests)
		assert.NoError(t, err)
	})

	t.Run("image digest exists - no match found", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/mybun@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a"
		image.Digest = "sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321"

		err := d.validateImageDigest(image, repoDigests)
		require.NotNil(t, err, "expected an error")
		assert.Contains(t, err.Error(), "content digest mismatch")
	})

	t.Run("image digest exists - repo digest unparseable", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/mybun@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a"
		image.Digest = "sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321"

		badRepoDigests := []string{"myreg/mybun@sha256:deadbeef"}

		err := d.validateImageDigest(image, badRepoDigests)
		require.NotNil(t, err, "expected an error")
		assert.Contains(t, err.Error(), "unable to parse repo digest")
	})

	t.Run("image digest exists - match found", func(t *testing.T) {
		d := &Driver{}

		image := bundle.InvocationImage{}
		image.Image = "myreg/mybun@sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a"
		image.Digest = "sha256:d366a4665ab44f0648d7a00ae3fae139d55e32f9712c67accd604bb55df9d05a"

		err := d.validateImageDigest(image, repoDigests)
		require.NoError(t, err, "validateImageDigest failed")
	})
}
