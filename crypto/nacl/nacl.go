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

type naclBackend struct {
	backend git.Backend
	key     [32]byte
}

// NewNaClBackend returns a git.Backend implementation that encrypts data using nacl
func NewNaClBackend(backend git.Backend, key [32]byte) git.Backend {
	return &naclBackend{
		backend: backend,
		key:     key,
	}
}

func (r *naclBackend) ReadBlob(name string) (io.ReadCloser, error) {
	encryptedRdr, err := r.backend.ReadBlob(name + ".nacl")
	if err != nil {
		return nil, err
	}
	defer encryptedRdr.Close()

	data, err := ioutil.ReadAll(encryptedRdr)
	if err != nil {
		return nil, err
	}

	if len(data) < 24 {
		return nil, errors.New("encrypted message is too short")
	}
	var nonce [24]byte
	copy(nonce[:], data)
	data = data[24:]

	out, ok := secretbox.Open([]byte{}, data, &nonce, &r.key)
	if !ok {
		return nil, errors.New("error verifying encrypted data")
	}
	return ioutil.NopCloser(bytes.NewBuffer(out)), nil
}

func (r *naclBackend) WriteBlob(name string, rdr io.Reader) error {
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return err
	}
	nonce := makeNonce()
	out := secretbox.Seal(nonce[:], data, nonce, &r.key)
	return r.backend.WriteBlob(name+".nacl", bytes.NewBuffer(out))
}

func makeNonce() *[24]byte {
	var nonce [24]byte
	_, err := rand.Read(nonce[:])
	if err != nil {
		panic(err)
	}
	return &nonce
}
