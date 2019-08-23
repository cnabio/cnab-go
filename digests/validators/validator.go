package validators

import (
	"github.com/deislabs/cnab-go/bundle"
)

// Known image types for validation
const (
	ImageTypeDocker = "docker"
	ImageTypeOCI    = "oci"
)

// Create builds a new image validator collection for know image types
func Create() map[string]ImageValidator {
	d := NewDockerValidator()
	return map[string]ImageValidator{
		ImageTypeDocker: d,
		ImageTypeOCI:    d,
	}
}

// ImageValidator valdiates that the content digest for a given image matches
// the source.
type ImageValidator interface {
	Validate(image bundle.Image) error
}
