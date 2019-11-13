package crud

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ Store = &fileSystemStore{}

func TestFilesystemStore(t *testing.T) {
	const claims = "claims"

	is := assert.New(t)
	tmdir, err := ioutil.TempDir("", "duffle-test-")
	is.NoError(err)
	defer os.RemoveAll(tmdir)
	s := NewFileSystemStore(tmdir, "data")
	key := "testkey"
	val := []byte("testval")
	is.NoError(s.Save(claims, key, val))
	list, err := s.List(claims)
	is.NoError(err)
	is.Len(list, 1)
	d, err := s.Read(claims, "testkey")
	is.NoError(err)
	is.Equal([]byte("testval"), d)
	is.NoError(s.Delete(claims, key))
	list, err = s.List(claims)
	is.NoError(err)
	is.Len(list, 0)
}
