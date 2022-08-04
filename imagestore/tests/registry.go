package tests

import (
	"bytes"
	"embed"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/certs/*
var testcerts embed.FS

// createTestCertsDir creates a temporary directory with a self-signed certificate
// that is cleaned up automatically when the test is done.
func createTestCertsDir(t *testing.T) (string, error) {
	certDir := t.TempDir()

	copyCert := func(certName string) error {
		data, err := testcerts.ReadFile(filepath.Join("testdata/certs", certName))
		if err != nil {
			return err
		}

		return ioutil.WriteFile(filepath.Join(certDir, certName), data, 0600)
	}

	if err := copyCert("domain.key"); err != nil {
		return "", err
	}

	if err := copyCert("registry_auth.key"); err != nil {
		return "", err
	}

	if err := copyCert("registry_auth.crt"); err != nil {
		return "", err
	}

	return certDir, nil
}

// StartTestRegistry runs a temporary insecure docker registry
// that uses self-signed TLS certificates, returning the port it is running on.
func StartTestRegistry(t *testing.T) string {
	certDir, err := createTestCertsDir(t)
	require.NoError(t, err, "Failed to create a temporary directory with our test self-signed certificates")

	cmd := exec.Command("docker", "run", "-d", "-P",
		fmt.Sprintf("-v=%s:/certs", certDir),
		"-e=REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry_auth.crt",
		"-e=REGISTRY_HTTP_TLS_KEY=/certs/registry_auth.key",
		"registry:2")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	require.NoError(t, cmd.Run(), "failed to run a docker container with a test registry")

	// Remove the registry at the end of the test
	containerID := strings.TrimSuffix(stdout.String(), "\n")
	t.Cleanup(func() {
		exec.Command("docker", "rm", "-vf", containerID).Run()
	})

	// Get the dynamic port that the registry is running on
	cmd = exec.Command("docker", "inspect", containerID, "--format", `{{ (index (index .NetworkSettings.Ports "5000/tcp") 0).HostPort }}`)
	stdout.Truncate(0)
	cmd.Stdout = &stdout
	require.NoError(t, cmd.Run(), "failed to retrieve the port that the test registry is running on")

	port := strings.TrimSuffix(stdout.String(), "\n")
	return port
}
