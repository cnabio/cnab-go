package crud

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ Store = FileSystemStore{}

func TestFilesystemStore(t *testing.T) {
	is := assert.New(t)
	tmdir, err := ioutil.TempDir("", "duffle-test-")
	is.NoError(err)
	defer os.RemoveAll(tmdir)
	s := NewFileSystemStore(tmdir, map[string]string{testItemType: ".json"})
	key := "testkey"
	val := []byte("testval")
	is.NoError(s.Save(testItemType, testGroup, key, val))
	list, err := s.List(testItemType, testGroup)
	is.NoError(err)
	is.Len(list, 1)
	d, err := s.Read(testItemType, "testkey")
	is.NoError(err)
	is.Equal([]byte("testval"), d)
	is.NoError(s.Delete(testItemType, key))
	list, err = s.List(testItemType, testGroup)
	is.NoError(err)
	is.Len(list, 0)
}
