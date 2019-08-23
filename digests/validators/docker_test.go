package validators

import (
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/pivotal/image-relocation/pkg/image"
	"github.com/pivotal/image-relocation/pkg/registry"
	"github.com/stretchr/testify/assert"
)

type fakeClient struct {
	copyFake       func(source image.Name, target image.Name) (image.Digest, int64, error)
	digestFake     func(img image.Name) (image.Digest, error)
	layoutFake     func(path string) (registry.Layout, error)
	readLayoutFake func(path string) (registry.Layout, error)
}

func (f fakeClient) Copy(source image.Name, target image.Name) (image.Digest, int64, error) {
	return f.copyFake(source, target)
}

func (f fakeClient) Digest(img image.Name) (image.Digest, error) {
	return f.digestFake(img)
}

func (f fakeClient) NewLayout(path string) (registry.Layout, error) {
	return f.layoutFake(path)
}

func (f fakeClient) ReadLayout(path string) (registry.Layout, error) {
	return f.readLayoutFake(path)
}

func TestDigestMatches(t *testing.T) {

	client := fakeClient{
		digestFake: func(img image.Name) (image.Digest, error) {
			return image.NewDigest("sha256:3eafa67f4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5")
		},
	}

	v := dockerValidator{
		registryClient: client,
	}

	img := bundle.Image{
		BaseImage: bundle.BaseImage{
			Image:  "repo/someimage:v1.0.0",
			Digest: "sha256:3eafa67f4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5",
		},
	}
	err := v.Validate(img)
	assert.NoError(t, err, "expected digests to match")
}

func TestDigestNotEqual(t *testing.T) {

	client := fakeClient{
		digestFake: func(img image.Name) (image.Digest, error) {
			return image.NewDigest("sha256:cafebabe4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5")
		},
	}

	v := dockerValidator{
		registryClient: client,
	}

	img := bundle.Image{
		BaseImage: bundle.BaseImage{
			Image:  "repo/someimage:v1.0.0",
			Digest: "sha256:3eafa67f4db52455532b19b80bd6b9a7b99d12717b96dcf25ced4cb49fa6d2d5",
		},
	}
	err := v.Validate(img)
	assert.Error(t, err, "expected digests not to match")
}
