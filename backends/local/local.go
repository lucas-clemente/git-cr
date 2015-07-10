package local

import (
	"io"
	"os"

	"github.com/lucas-clemente/git-cr/git/repo"
)

type localBackend struct {
	path string
}

// NewLocalBackend returns a backend that stores data in the given path
func NewLocalBackend(path string) (repo.Backend, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	return &localBackend{path: path}, nil
}

func (b *localBackend) ReadBlob(name string) (io.ReadCloser, error) {
	f, err := os.Open(b.path + "/" + name)
	if os.IsNotExist(err) {
		return nil, repo.ErrNotFound
	}
	return f, err
}

func (b *localBackend) WriteBlob(name string, r io.Reader) error {
	f, err := os.Create(b.path + "/" + name)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}
