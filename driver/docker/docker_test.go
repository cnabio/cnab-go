package docker

import (
	"context"
	"fmt"
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

func TestDriver_setConfigurationOptions(t *testing.T) {
	img := "example.com/myimage"
	op := &driver.Operation{
		Image: bundle.InvocationImage{
			BaseImage: bundle.BaseImage{Image: img},
		},
	}

	t.Run("defaults", func(t *testing.T) {
		d := &Driver{}

		err := d.setConfigurationOptions(op)
		require.NoError(t, err)

		cfg := d.containerCfg
		wantCfg := container.Config{
			Image:        img,
			AttachStdout: true,
			AttachStderr: true,
			Entrypoint:   []string{"/cnab/app/run"},
		}
		assert.Equal(t, wantCfg, cfg)

		hostCfg := d.containerHostCfg
		assert.Equal(t, container.HostConfig{}, hostCfg)
	})

	t.Run("docker network", func(t *testing.T) {
		net := "mynetwork"
		d := &Driver{}
		d.SetConfig(map[string]string{SettingNetwork: net})

		err := d.setConfigurationOptions(op)
		require.NoError(t, err)

		hostCfg := d.containerHostCfg
		assert.Equal(t, net, string(hostCfg.NetworkMode))
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
				"DOCKER_DRIVER_QUIET": "1",
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

func TestGetContainerUserId(t *testing.T) {
	testcases := []struct {
		name    string
		user    string
		wantUID int
	}{
		{"no user specified", "", 0},
		{"user name specified", "someuser", 0}, // We can't determine the user id from outside the container
		{"uid specified", "65532", 65532},
		{"uid and gid specified", "31:33", 31},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantUID, getContainerUserID(tc.user))
		})
	}
}

type MockContainerResultClient struct {
	ContainerStopId    string
	ContainerStopError error
}

func (m *MockContainerResultClient) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	statusCh := make(chan container.WaitResponse, 1)
	errCh := make(chan error, 1)
	return statusCh, errCh
}
func (m *MockContainerResultClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	m.ContainerStopId = containerID
	return m.ContainerStopError
}

func TestDriver_exec_ContextCancellation(t *testing.T) {
	t.Run("context cancellation stops container", func(t *testing.T) {
		d := &Driver{
			containerID: "unit-test-container-id",
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		op := &driver.Operation{
			Image: bundle.InvocationImage{
				BaseImage: bundle.BaseImage{Image: "test-image"},
			},
		}

		client := &MockContainerResultClient{}

		result, err := d.getContainerResult(ctx, client, op)

		assert.Error(t, err, "expected Run to return an error with cancelled context")
		assert.Equal(t, driver.OperationResult{}, result)
		assert.Equal(t, d.containerID, client.ContainerStopId)
	})
}

func TestDriver_exec_ContextCancellationError(t *testing.T) {
	t.Run("context cancellation stops container", func(t *testing.T) {
		d := &Driver{}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		op := &driver.Operation{
			Image: bundle.InvocationImage{
				BaseImage: bundle.BaseImage{Image: "test-image"},
			},
		}

		client := &MockContainerResultClient{
			ContainerStopError: fmt.Errorf("unit-test-error"),
		}

		result, err := d.getContainerResult(ctx, client, op)

		assert.Error(t, err, "expected Run to return an error with cancelled context")
		assert.Equal(t, driver.OperationResult{}, result)
		assert.Equal(t, err, client.ContainerStopError)
	})
}
