package imagestore

import (
	"io"
	"io/ioutil"
	"net/http"

	"github.com/cnabio/image-relocation/pkg/image"
	"github.com/cnabio/image-relocation/pkg/registry/ggcr"
)

// Store is an abstract image store.
type Store interface {
	// Add copies the image with the given name to the image store.
	Add(img string) (contentDigest string, err error)

	// Push copies the image with the given digest from an image with the given name in the image store to a repository
	// with the given name.
	Push(dig image.Digest, src image.Name, dst image.Name) error
}

// Constructor is a function which creates an images store based on parameters represented as options
type Constructor func(...Option) (Store, error)

// Parameters is used to create image stores.
type Parameters struct {
	ArchiveDir string
	Logs       io.Writer

	// Transport is http.Transport to use when communicating with an OCI registry.
	Transport *http.Transport
}

// BuildRegistryOptions returns a list of applicable ggcr.Option values
// to use when calling ggcr.NewRegistryClient().
func (p Parameters) BuildRegistryOptions() []ggcr.Option {
	var regOpts []ggcr.Option
	if p.Transport != nil {
		regOpts = append(regOpts, ggcr.WithTransport(p.Transport))
	}
	return regOpts
}

// Options is a function which returns updated parameters.
type Option func(Parameters) Parameters

func Create(options ...Option) Parameters {
	b := Parameters{
		Logs: ioutil.Discard,
	}
	for _, op := range options {
		b = op(b)
	}
	return b
}

// WithArchiveDir return an option to set the archive directory parameter.
func WithArchiveDir(archiveDir string) Option {
	return func(b Parameters) Parameters {
		return Parameters{
			ArchiveDir: archiveDir,
			Logs:       b.Logs,
			Transport:  b.Transport,
		}
	}
}

// WithLogs return an option to set the logs parameter.
func WithLogs(logs io.Writer) Option {
	return func(b Parameters) Parameters {
		return Parameters{
			ArchiveDir: b.ArchiveDir,
			Logs:       logs,
			Transport:  b.Transport,
		}
	}
}

// WithTransport returns an option with the Transport parameter set.
func WithTransport(transport *http.Transport) Option {
	return func(b Parameters) Parameters {
		return Parameters{
			ArchiveDir: b.ArchiveDir,
			Logs:       b.Logs,
			Transport:  transport,
		}
	}
}
