package digests

import (
	"fmt"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/digests/validators"
	"github.com/pkg/errors"
)

// Validator validates that the contentDigest for each provided image matches
// the authoritative source. By default, validation will fail if a contentDigest
// is not provided for the image. Set allowMissingDigests to true to allow this
type Validator interface {
	Validate(images []bundle.Image, allowMissingDigests bool) error
}

type validator struct {
	validators map[string]validators.ImageValidator
}

// NewValidator returns a digest Validator
func NewValidator() Validator {
	return &validator{
		validators: validators.Create(),
	}
}

func (v *validator) Validate(images []bundle.Image, allowMissingDigests bool) error {
	for _, img := range images {
		validator, ok := v.validators[img.ImageType]
		if !ok {
			return fmt.Errorf("unknown image type: %s", img.ImageType)
		}
		if img.Digest == "" && !allowMissingDigests {
			return fmt.Errorf("unable to validate %s, digest not present", img.Image)
		}
		if img.Digest != "" {
			if err := validator.Validate(img); err != nil {
				return errors.Wrapf(err, "validation failed for %s", img.Image)
			}
		}
	}
	return nil
}
