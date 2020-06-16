package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/cnabio/cnab-go/driver"
)

var _ driver.Driver = &Driver{}

func TestCheckDriverExists(t *testing.T) {
	name := "missing-driver"
	cmddriver := &Driver{Name: name}
	if cmddriver.CheckDriverExists() {
		t.Errorf("Expected driver %s not to exist", name)
	}

	name = "existing-driver"
	testfunc := func(t *testing.T, cmddriver *Driver) {
		if !cmddriver.CheckDriverExists() {
			t.Fatalf("Expected driver %s to exist", cmddriver.Name)
		}

	}
	CreateAndRunTestCommandDriver(t, name, "", testfunc)
}

func CreateAndRunTestCommandDriver(t *testing.T, name string, content string, testfunc func(t *testing.T, d *Driver)) {
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
	path := os.Getenv("PATH")
	pathlist := []string{dirname, path}
	newpath := strings.Join(pathlist, string(os.PathListSeparator))
	defer os.Setenv("PATH", path)
	os.Setenv("PATH", newpath)
	testfunc(t, cmddriver)
}
