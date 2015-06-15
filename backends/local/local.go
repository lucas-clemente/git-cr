package local

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

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
	filename := b.path + "/" + from + "_" + to + ".pack"
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil, git.ErrorDeltaNotFound
	}
	if err != nil {
		return nil, err
	}
	return filename, nil
}

func (b *localBackend) GetRefs() (git.Refs, error) {
	data, err := ioutil.ReadFile(b.path + "/refs.json")
	if err != nil {
		return nil, err
	}
	var refs git.Refs
	if err := json.Unmarshal(data, &refs); err != nil {
		return nil, err
	}
	return refs, nil
}

func (b *localBackend) ReadPackfile(d git.Delta) (io.ReadCloser, error) {
	return os.Open(d.(string))
}

func (b *localBackend) UpdateRef(update git.RefUpdate) error {
	refs, err := b.GetRefs()
	if os.IsNotExist(err) {
		refs = git.Refs{}
	} else if err != nil {
		return err
	}
	if update.NewID == "" {
		delete(refs, update.Name)
	} else {
		refs[update.Name] = update.NewID
	}
	data, err := json.Marshal(refs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(b.path+"/refs.json", data, 0644)
}

func (b *localBackend) WritePackfile(from, to string, r io.Reader) error {
	file, err := os.Create(b.path + "/" + from + "_" + to + ".pack")
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, r)
	return err
}

func (b *localBackend) ListAncestors(target string) ([]string, error) {
	panic("not implemented")
}
