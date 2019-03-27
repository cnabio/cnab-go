package mock

import (
	"testing"

	"github.com/radu-matei/cnab-go/pkg/imagebuilder"
)

// test mock Builder is assignable to the imagebuilder.ImageBuilder interface
func TestBuilder_implBuilder(t *testing.T) {
	var _ imagebuilder.ImageBuilder = (*Builder)(nil)
}
