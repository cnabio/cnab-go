package validators

import (
	"fmt"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/pivotal/image-relocation/pkg/image"
	"github.com/pivotal/image-relocation/pkg/registry"
	"github.com/pkg/errors"
)

type dockerValidator struct {
	registryClient registry.Client
}

// NewDockerValidator creates a new validator for Docker and OCI images
func NewDockerValidator() ImageValidator {
	return dockerValidator{
		registryClient: registry.NewRegistryClient(),
	}
}

// Validate creates a validator that obtains a digest from a repository
// and compares that against the specified digest in the bundle.Image parameter
func (d dockerValidator) Validate(img bundle.Image) error {
	imgName, err := image.NewName(img.Image)
	if err != nil {
		return errors.Wrap(err, "unable to validate digest")
	}
	digest, err := d.registryClient.Digest(imgName)
	if err != nil {
		return errors.Wrap(err, "unable to obtain digest")
	}
	if digest.String() != img.Digest {
		return fmt.Errorf("digest validation failed: expected %s found %s", digest.String(), img.Digest)
	}
	return nil
}
