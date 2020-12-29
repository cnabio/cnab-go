package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/cnabio/cnab-go/driver"

	"github.com/stretchr/testify/assert"
)

var _ driver.Driver = &Driver{}

func TestDriver_CheckDriverExists(t *testing.T) {
	t.Run("missing driver", func(t *testing.T) {
		cmddriver := &Driver{Name: "missing-driver"}
		exists := cmddriver.CheckDriverExists()
		assert.False(t, exists, "expected driver to not exist")
	})

	t.Run("missing driver - executable path set", func(t *testing.T) {
		test := func(cmddriver *Driver) {
			cmddriver.Path = "/missing-driver.sh"

			exists := cmddriver.CheckDriverExists()
			assert.False(t, exists, "expected driver to not be found")
		}

		CreateAndRunTestCommandDriver(t, "missing-driver", true, "", test)
	})

	t.Run("existing driver", func(t *testing.T) {
		test := func(cmddriver *Driver) {
			exists := cmddriver.CheckDriverExists()
			assert.True(t, exists, "expected driver to exist")
		}
		CreateAndRunTestCommandDriver(t, "existing-driver", false, "", test)
	})

	t.Run("existing driver - executable path set", func(t *testing.T) {
		test := func(cmddriver *Driver) {
			exists := cmddriver.CheckDriverExists()
			assert.True(t, exists, "expected driver to exist")
		}
		CreateAndRunTestCommandDriver(t, "existing-driver", true, "", test)
	})
}

func TestDriver_Handles(t *testing.T) {
	content := `#!/bin/sh
echo "test,debug"
`

	t.Run("can handle", func(t *testing.T) {
		test := func(cmddriver *Driver) {
			handles := cmddriver.Handles("test")
			assert.True(t, handles, "expected driver to handle the image type")
		}
		CreateAndRunTestCommandDriver(t, "can-handle-driver", false, content, test)
	})

	t.Run("can handle - path set", func(t *testing.T) {
		test := func(cmddriver *Driver) {
			handles := cmddriver.Handles("test")
			assert.True(t, handles, "expected driver to handle the image type")
		}
		CreateAndRunTestCommandDriver(t, "can-handle-driver", true, content, test)
	})
}

func CreateAndRunTestCommandDriver(t *testing.T, name string, explicitPath bool, content string, testfunc func(d *Driver)) {
	cmddriver := &Driver{Name: name}
	dirname, err := ioutil.TempDir("", "cnab")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dirname)
	filename := fmt.Sprintf("%s/cnab-%s", dirname, name)
	newfile, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}

	if len(content) > 0 {
		newfile.WriteString(content)
	}

	newfile.Chmod(0755)
	newfile.Close()

	if explicitPath {
		cmddriver.Path = filename
	} else { // Add the driver in PATH so it can be found
		path := os.Getenv("PATH")
		pathlist := []string{dirname, path}
		newpath := strings.Join(pathlist, string(os.PathListSeparator))
		defer os.Setenv("PATH", path)
		os.Setenv("PATH", newpath)
	}

	testfunc(cmddriver)
}
