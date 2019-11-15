package host

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/cnabio/cnab-go/secrets"
)

const (
	SourceEnv     = "env"
	SourceCommand = "command"
	SourcePath    = "path"
	SourceValue   = "value"
)

var _ secrets.Store = &SecretStore{}

type SecretStore struct{}

func (h *SecretStore) Resolve(keyName string, keyValue string) (string, error) {
	// Precedence is command, path, env, value
	switch strings.ToLower(keyName) {
	case SourceCommand:
		data, err := execCmd(keyValue)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case SourcePath:
		data, err := ioutil.ReadFile(os.ExpandEnv(keyValue))
		if err != nil {
			return "", err
		}
		return string(data), nil
	case SourceEnv:
		var ok bool
		data, ok := os.LookupEnv(keyValue)
		if !ok {
			return "", fmt.Errorf("environment variable %s is not defined", keyName)
		}
		return data, nil
	case SourceValue:
		return keyValue, nil
	default:
		return "", fmt.Errorf("invalid credential source: %s", keyName)
	}
}

func execCmd(cmd string) ([]byte, error) {
	parts := strings.Split(cmd, " ")
	c := parts[0]
	args := parts[1:]
	run := exec.Command(c, args...)

	return run.CombinedOutput()
}
