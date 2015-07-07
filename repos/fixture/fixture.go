package fixture

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"

	"github.com/lucas-clemente/git-cr/git"
)

// A FixtureRepo for tests
type FixtureRepo struct {
	Revisions []git.Revision
	Packfiles [][]byte
}

var _ git.Repo = &FixtureRepo{}

// NewFixtureRepo makes a new fixture repo
func NewFixtureRepo() *FixtureRepo {
	return &FixtureRepo{}
}

// GetRevisions implements git.Repo
func (r *FixtureRepo) GetRevisions() ([]git.Revision, error) {
	return r.Revisions, nil
}

// SaveNewRevision implements git.Repo
func (r *FixtureRepo) SaveNewRevision(rev git.Revision, packfile io.Reader) error {
	r.Revisions = append(r.Revisions, rev)
	data, err := ioutil.ReadAll(packfile)
	if err != nil {
		return err
	}
	r.Packfiles = append(r.Packfiles, data)
	return nil
}

// ReadPackfile implements git.Repo
func (r *FixtureRepo) ReadPackfile(toRev int) (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewBuffer(r.Packfiles[toRev])), nil
}

// SaveNewRevisionB64 adds a base64-encoded packfile to the repo
func (r *FixtureRepo) SaveNewRevisionB64(rev git.Revision, b64 string) {
	pack, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic("invalid base64 in FixtureRepo.AddPackfile")
	}
	if err := r.SaveNewRevision(rev, bytes.NewBuffer(pack)); err != nil {
		panic(err)
	}
}
