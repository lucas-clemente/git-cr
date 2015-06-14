package local

import (
	"io"

	"github.com/lucas-clemente/git-cr/git"
)

type localBackend struct {
	path string
}

// NewLocalBackend returns a backend that stores data in the given path
func NewLocalBackend(path string) git.ListingBackend {
	return &localBackend{path: path}
}

func (b *localBackend) FindDelta(from, to string) (git.Delta, error) {
	panic("not implemented")
}

func (b *localBackend) GetRefs() (git.Refs, error) {
	panic("not implemented")
}

func (b *localBackend) ReadPackfile(d git.Delta) (io.ReadCloser, error) {
	panic("not implemented")
}

func (b *localBackend) UpdateRef(update git.RefUpdate) error {
	panic("not implemented")
}

func (b *localBackend) WritePackfile(from, to string, r io.Reader) error {
	panic("not implemented")
}

func (b *localBackend) ListAncestors(target string) ([]string, error) {
	panic("not implemented")
}
