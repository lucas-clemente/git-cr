package local

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucas-clemente/git-cr/git"
)

type localRepo struct {
	path string
}

// NewLocalRepo returns a repo that stores data in the given path
func NewLocalRepo(path string) (git.Repo, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	return &localRepo{path: path}, nil
}

func (b *localRepo) FindDelta(from, to string) (git.Delta, error) {
	filename := b.buildPackfileName(from, to)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil, git.ErrorDeltaNotFound
	}
	if err != nil {
		return nil, err
	}
	return filename, nil
}

func (b *localRepo) ReadRefs() (io.ReadCloser, error) {
	r, err := os.Open(b.path + "/refs.json")
	if os.IsNotExist(err) {
		return nil, git.ErrorRepoEmpty
	}
	return r, err
}

func (b *localRepo) ReadPackfile(d git.Delta) (io.ReadCloser, error) {
	return os.Open(d.(string))
}

func (b *localRepo) WriteRefs(r io.Reader) error {
	file, err := os.Create(b.path + "/refs.json")
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, r)
	return err
}

func (b *localRepo) WritePackfile(from, to string, r io.Reader) error {
	file, err := os.Create(b.buildPackfileName(from, to))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, r)
	return err
}

func (b *localRepo) ListAncestors(target string) ([]string, error) {
	matches, err := filepath.Glob(b.buildPackfileName("*", target))
	if err != nil {
		return nil, err
	}
	for i, m := range matches {
		matches[i] = strings.TrimSuffix(strings.TrimPrefix(m, b.path+"/"), "_"+target+".pack")
	}
	return matches, nil
}

func (b *localRepo) buildPackfileName(from, to string) string {
	return b.path + "/" + from + "_" + to + ".pack"
}
