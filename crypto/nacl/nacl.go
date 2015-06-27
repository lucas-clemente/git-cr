package nacl

import (
	"bytes"
	"crypto/rand"
	"errors"
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

// NewNaClRepo returns a git.Repo implementation that encrypts data using nacl
func NewNaClRepo(backend git.Repo, key [32]byte) (git.Repo, error) {
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
	backendReader, err := r.repo.ReadRefs()
	if err != nil {
		return nil, err
	}
	return decrypt(backendReader, &r.key)
}

func (r *naclRepo) WriteRefs(rdr io.Reader) error {
	encryptedRdr, err := encrypt(rdr, &r.key)
	if err != nil {
		return err
	}
	return r.repo.WriteRefs(encryptedRdr)
}

func (r *naclRepo) ReadPackfile(d git.Delta) (io.ReadCloser, error) {
	backendReader, err := r.repo.ReadPackfile(d)
	if err != nil {
		return nil, err
	}
	return decrypt(backendReader, &r.key)
}

func (r *naclRepo) WritePackfile(from, to string, rdr io.Reader) error {
	encryptedRdr, err := encrypt(rdr, &r.key)
	if err != nil {
		return err
	}
	return r.repo.WritePackfile(from, to, encryptedRdr)
}

func encrypt(in io.Reader, key *[32]byte) (io.Reader, error) {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}
	nonce := makeNonce()
	out := secretbox.Seal(nonce[:], data, nonce, key)
	return bytes.NewBuffer(out), nil
}

func decrypt(in io.ReadCloser, key *[32]byte) (io.ReadCloser, error) {
	defer in.Close()

	data, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	if len(data) < 24 {
		return nil, errors.New("encrypted message is too short")
	}
	var nonce [24]byte
	copy(nonce[:], data)
	data = data[24:]

	out, ok := secretbox.Open([]byte{}, data, &nonce, key)
	if !ok {
		return nil, errors.New("error verifying encrypted data")
	}
	return ioutil.NopCloser(bytes.NewBuffer(out)), nil
}

func makeNonce() *[24]byte {
	var nonce [24]byte
	_, err := rand.Read(nonce[:])
	if err != nil {
		panic(err)
	}
	return &nonce
}
