package nacl

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/nacl/secretbox"

	"github.com/lucas-clemente/git-cr/git"
)

type naclRepo struct {
	// repo is not embedded to prevent acidentally leaking info if new methods  are added to git.Repo
	repo git.Repo
	key  [32]byte
}

// NewNaclRepo returns a git.Repo implementation that encrypts data using nacl
func NewNaclRepo(backend git.Repo, key [32]byte) (git.Repo, error) {
	return &naclRepo{
		repo: backend,
		key:  key,
	}, nil
}

func (r *naclRepo) FindDelta(from, to string) (git.Delta, error) {
	return r.repo.FindDelta(from, to)
}

func (r *naclRepo) ListAncestors(target string) ([]string, error) {
	return r.repo.ListAncestors(target)
}

func (r *naclRepo) ReadRefs() (io.ReadCloser, error) {
	panic("not implemented")
}

func (r *naclRepo) WriteRefs(rdr io.Reader) error {
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return err
	}
	nonce := makeNonce()
	out := secretbox.Seal(nonce[:], data, nonce, &r.key)
	return r.repo.WriteRefs(bytes.NewBuffer(out))
}

func (r *naclRepo) ReadPackfile(d git.Delta) (io.ReadCloser, error) {
	panic("not implemented")
}

func (r *naclRepo) WritePackfile(from, to string, rdr io.Reader) error {
	panic("not implemented")
}

func makeNonce() *[24]byte {
	var nonce [24]byte
	_, err := rand.Read(nonce[:])
	if err != nil {
		panic(err)
	}
	return &nonce
}
