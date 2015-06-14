package git_test

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"

	"github.com/lucas-clemente/git-cr/git"
)

// A fixtureBackend for tests
type fixtureBackend struct {
	currentRefs     []git.Ref
	packfilesFromTo map[string]map[string][]byte

	updatedRefs     []git.RefUpdate
	pushedPackfiles [][]byte
	pushedRevs      []string
}

var _ git.Backend = &fixtureBackend{}

func newFixtureBackend() *fixtureBackend {
	return &fixtureBackend{
		packfilesFromTo: map[string]map[string][]byte{"": map[string][]byte{}},
	}
}

func (b *fixtureBackend) addPackfile(from, to, b64 string) {
	m, ok := b.packfilesFromTo[from]
	if !ok {
		b.packfilesFromTo[from] = map[string][]byte{}
		m = b.packfilesFromTo[from]
	}
	pack, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic("invalid base64 in fixtureBackend.addPackfile")
	}
	m[to] = pack
}

func (b *fixtureBackend) FindDelta(from, to string) (git.Delta, error) {
	m, ok := b.packfilesFromTo[from]
	if !ok {
		return nil, nil
	}
	p, ok := m[to]
	if !ok {
		return nil, nil
	}
	return ioutil.NopCloser(bytes.NewBuffer(p)), nil
}

func (b *fixtureBackend) GetRefs() ([]git.Ref, error) {
	return b.currentRefs, nil
}

func (*fixtureBackend) ReadPackfile(d git.Delta) (io.ReadCloser, error) {
	return d.(io.ReadCloser), nil
}

func (b *fixtureBackend) UpdateRef(update git.RefUpdate) error {
	b.updatedRefs = append(b.updatedRefs, update)
	return nil
}

func (b *fixtureBackend) WritePackfile(from, to string, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	b.pushedPackfiles = append(b.pushedPackfiles, data)
	b.pushedRevs = append(b.pushedRevs, to)
	return nil
}
