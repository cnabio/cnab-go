package crud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockStore(t *testing.T) {
	s := NewMockStore()
	is := assert.New(t)
	is.NoError(s.Save(testItemType, "test", []byte("data")))
	list, err := s.List(testItemType)
	is.NoError(err)
	is.Len(list, 1)
	data, err := s.Read(testItemType, "test")
	is.NoError(err)
	is.Equal(data, []byte("data"))
}
