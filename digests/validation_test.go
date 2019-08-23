package digests

import (
	"fmt"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/digests/validators"
	"github.com/stretchr/testify/assert"
)

type fakeImageValidator struct {
	digest string
}

func (f fakeImageValidator) Validate(img bundle.Image) error {
	if f.digest != img.Digest {
		return fmt.Errorf("digest validation failed: expected %s found %s", img.Digest, f.digest)
	}
	return nil
}

func fakeValidator(fake fakeImageValidator) Validator {
	return &validator{
		validators: map[string]validators.ImageValidator{
			validators.ImageTypeDocker: fake,
		},
	}
}

func emptyValidator() fakeImageValidator {
	return fakeImageValidator{
		digest: "",
	}
}

func TestNoImages(t *testing.T) {
	v := fakeValidator(emptyValidator())

	imgs := []bundle.Image{}
	err := v.Validate(imgs, true)
	assert.NoError(t, err, "expected no error when no images were present")
}

func TestMissingDigestAllowed(t *testing.T) {
	v := fakeValidator(emptyValidator())

	imgs := []bundle.Image{
		{
			BaseImage: bundle.BaseImage{
				Image:     "repo/someimage:v1.0.0",
				ImageType: validators.ImageTypeDocker,
				Digest:    "",
			},
		},
	}

	err := v.Validate(imgs, true)
	assert.NoError(t, err, "expected no error for empty digest when allowMissingDigest is true")

}

func TestMissingDigestNotAllowed(t *testing.T) {
	v := fakeValidator(emptyValidator())

	imgs := []bundle.Image{
		{
			BaseImage: bundle.BaseImage{
				Image:     "repo/someimage:v1.0.0",
				ImageType: validators.ImageTypeDocker,
				Digest:    "",
			},
		},
	}
	err := v.Validate(imgs, false)
	assert.Error(t, err, "expected an error for empty digest when allowMissingDigest is false")
	assert.EqualError(t, err, "unable to validate repo/someimage:v1.0.0, digest not present")
}

func TestDigestMatches(t *testing.T) {

	v := fakeValidator(fakeImageValidator{
		digest: "sha256:3eafa67f4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5",
	})

	imgs := []bundle.Image{
		{
			BaseImage: bundle.BaseImage{
				Image:     "repo/someimage:v1.0.0",
				ImageType: validators.ImageTypeDocker,
				Digest:    "sha256:3eafa67f4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5",
			},
		},
	}
	err := v.Validate(imgs, false)
	assert.NoError(t, err, "expected digests to match")
}

func TestDigestNotEqual(t *testing.T) {

	actual := "sha256:cafebabe4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5"
	expected := "sha256:3eafa67f4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5"
	imageName := "repo/someimage:v1.0.0"

	v := fakeValidator(fakeImageValidator{
		digest: actual,
	})

	imgs := []bundle.Image{
		{
			BaseImage: bundle.BaseImage{
				Image:     imageName,
				ImageType: validators.ImageTypeDocker,
				Digest:    expected,
			},
		},
	}
	err := v.Validate(imgs, false)
	assert.Error(t, err, "expected digests not to match")
	assert.EqualError(
		t,
		err,
		fmt.Sprintf(
			"validation failed for %s: digest validation failed: expected %s found %s",
			imageName,
			expected,
			actual,
		),
	)
}

func TestUnknownImageType(t *testing.T) {
	v := fakeValidator(emptyValidator())

	imgs := []bundle.Image{
		{
			BaseImage: bundle.BaseImage{
				Image:     "repo/someimage:v1.0.0",
				ImageType: "VirtualMachine",
				Digest:    "sha256:3eafa67f4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5",
			},
		},
	}
	err := v.Validate(imgs, false)
	assert.Error(t, err, "expected VirtualMachine to be an invalid type")
	assert.EqualError(t, err, "unknown image type: VirtualMachine")
}
