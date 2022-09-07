package docker

import (
	"os"
	"testing"

	"github.com/docker/go-connections/tlsconfig"
	"github.com/stretchr/testify/assert"
)

func Test_buildDockerClientOptions(t *testing.T) {
	// Tell Docker where its config is located, so that we have repeatable paths in the tests
	os.Setenv("DOCKER_CONFIG", "/home/me/.docker")
	defer os.Unsetenv("DOCKER_CONFIG")

	defaultTLSOptions := &tlsconfig.Options{
		CAFile:   "/home/me/.docker/ca.pem",
		CertFile: "/home/me/.docker/cert.pem",
		KeyFile:  "/home/me/.docker/key.pem",
	}

	customTLSOptions := &tlsconfig.Options{
		CAFile:   "/mycerts/ca.pem",
		CertFile: "/mycerts/cert.pem",
		KeyFile:  "/mycerts/key.pem",
	}

	t.Run("tls disabled", func(t *testing.T) {
		os.Unsetenv(DockerTLSVerifyEnvVar)
		opts := BuildDockerClientOptions()
		assert.False(t, opts.Common.TLS, "expected TLS to be disabled")
		assert.False(t, opts.Common.TLSVerify, "expected TLSVerify to be disabled")
		assert.Nil(t, opts.Common.TLSOptions, "expected TLSOptions to be unset")
	})

	t.Run("tls enabled without certs", func(t *testing.T) {
		os.Setenv(DockerTLSVerifyEnvVar, "true")
		os.Unsetenv(DockerCertPathEnvVar)
		defer func() {
			os.Unsetenv(DockerTLSVerifyEnvVar)
		}()

		opts := BuildDockerClientOptions()
		assert.True(t, opts.Common.TLS, "expected TLS to be enabled")
		assert.True(t, opts.Common.TLSVerify, "expected the certs to be verified")
		assert.Equal(t, defaultTLSOptions, opts.Common.TLSOptions, "expected TLSOptions to be initialized to the default TLS settings")
	})

	t.Run("tls enabled with custom certs", func(t *testing.T) {
		os.Setenv(DockerTLSVerifyEnvVar, "true")
		os.Setenv(DockerCertPathEnvVar, "/mycerts")
		defer func() {
			os.Unsetenv(DockerTLSVerifyEnvVar)
			os.Unsetenv(DockerCertPathEnvVar)
		}()

		opts := BuildDockerClientOptions()
		assert.True(t, opts.Common.TLS, "expected TLS to be enabled")
		assert.True(t, opts.Common.TLSVerify, "expected the certs to be verified")
		assert.Equal(t, customTLSOptions, opts.Common.TLSOptions, "expected TLSOptions to use the custom DOCKER_CERT_PATH set")
	})

	t.Run("tls enabled with self-signed certs", func(t *testing.T) {
		os.Setenv(DockerTLSVerifyEnvVar, "false")
		os.Setenv(DockerCertPathEnvVar, "/mycerts")
		defer func() {
			os.Unsetenv(DockerTLSVerifyEnvVar)
			os.Unsetenv(DockerCertPathEnvVar)
		}()

		opts := BuildDockerClientOptions()
		assert.True(t, opts.Common.TLS, "expected TLS to be enabled")
		assert.False(t, opts.Common.TLSVerify, "expected TLSVerify to be false")
		assert.Equal(t, customTLSOptions, opts.Common.TLSOptions, "expected TLSOptions to use the custom DOCKER_CERT_PATH set")
	})
}
