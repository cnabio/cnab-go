package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
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
			hostCfg.CapAdd = []string{"SUPER_POWERS"}
			return nil
		})

		err := d.ApplyConfigurationOptions()
		is.NoError(err)

		expectedHostCfg := container.HostConfig{
			CapAdd: []string{"SUPER_POWERS"},
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
	ContainerStopId     string
	ContainerStopError  error
	CopyFromContainerFn func(ctx context.Context, containerID string, options client.CopyFromContainerOptions) (client.CopyFromContainerResult, error)
	containerWaitCalls  int
}

func (m *MockContainerResultClient) ContainerWait(ctx context.Context, containerID string, options client.ContainerWaitOptions) client.ContainerWaitResult {
	m.containerWaitCalls++
	statusCh := make(chan container.WaitResponse, 1)
	errCh := make(chan error, 1)
	if m.containerWaitCalls > 1 {
		// Subsequent calls are the post-stop exitWait; signal the container has exited.
		statusCh <- container.WaitResponse{StatusCode: 0}
	}
	return client.ContainerWaitResult{
		Result: statusCh,
		Error:  errCh,
	}
}

func (m *MockContainerResultClient) ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error) {
	m.ContainerStopId = containerID
	return client.ContainerStopResult{}, m.ContainerStopError
}

func (m *MockContainerResultClient) CopyFromContainer(ctx context.Context, containerID string, options client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
	if m.CopyFromContainerFn != nil {
		return m.CopyFromContainerFn(ctx, containerID, options)
	}
	return client.CopyFromContainerResult{}, nil
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

		mockClient := &MockContainerResultClient{}

		result, err := d.getContainerResult(ctx, mockClient, op)

		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, driver.OperationResult{Outputs: map[string]string{}}, result)
		assert.Equal(t, d.containerID, mockClient.ContainerStopId)
	})
}

func TestDriver_exec_ContextCancellation_OutputsCaptured(t *testing.T) {
	t.Run("outputs are captured after context cancellation", func(t *testing.T) {
		const containerID = "unit-test-container-id"
		d := &Driver{containerID: containerID}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		op := &driver.Operation{
			Image: bundle.InvocationImage{
				BaseImage: bundle.BaseImage{Image: "test-image"},
			},
			Outputs: map[string]string{
				"/cnab/app/outputs/myoutput": "myoutput",
			},
		}

		// Build a TAR that mimics what CopyFromContainer returns.
		// CopyFromContainer strips the leading path so the header name is
		// "outputs/myoutput".
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		content := []byte("hello-output")
		_ = tw.WriteHeader(&tar.Header{
			Name: "outputs/myoutput",
			Size: int64(len(content)),
		})
		_, _ = tw.Write(content)
		tw.Close()
		tarBytes := buf.Bytes()

		mockClient := &MockContainerResultClient{
			CopyFromContainerFn: func(_ context.Context, _ string, _ client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
				return client.CopyFromContainerResult{Content: io.NopCloser(bytes.NewReader(tarBytes))}, nil
			},
		}

		result, err := d.getContainerResult(ctx, mockClient, op)

		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, containerID, mockClient.ContainerStopId)
		require.NotNil(t, result.Outputs)
		assert.Equal(t, "hello-output", result.Outputs["myoutput"])
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

		mockClient := &MockContainerResultClient{
			ContainerStopError: fmt.Errorf("unit-test-error"),
		}

		result, err := d.getContainerResult(ctx, mockClient, op)

		assert.Error(t, err, "expected Run to return an error with cancelled context")
		assert.Equal(t, driver.OperationResult{}, result)
		assert.Equal(t, err, mockClient.ContainerStopError)
	})
}
