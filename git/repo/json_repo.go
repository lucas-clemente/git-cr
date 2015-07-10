package repo

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
)

type jsonRepo struct {
	backend Backend
}

// NewJSONRepo returns a Repo implementation that stores revisions as json
func NewJSONRepo(backend Backend) Repo {
	return &jsonRepo{backend: backend}
}

func (r *jsonRepo) GetRevisions() ([]Revision, error) {
	rdr, err := r.backend.ReadBlob("revisions.json")
	if err != nil {
		if err == ErrNotFound {
			return []Revision{}, nil
		}
		return nil, err
	}
	defer rdr.Close()

	var revisions []Revision
	if err := json.NewDecoder(rdr).Decode(&revisions); err != nil {
		return nil, err
	}
	return revisions, nil
}

func (r *jsonRepo) SaveNewRevision(rev Revision, packfile io.Reader) error {
	revisions, err := r.GetRevisions()
	if err != nil {
		return err
	}
	revisions = append(revisions, rev)

	// Write revisions
	revisionsJSON, err := json.Marshal(revisions)
	if err != nil {
		return err
	}
	if err := r.backend.WriteBlob("revisions.json", bytes.NewBuffer(revisionsJSON)); err != nil {
		return err
	}

	// Write pack
	if err := r.backend.WriteBlob(strconv.Itoa(len(revisions)-1)+".pack", packfile); err != nil {
		return err
	}

	return nil
}

func (r *jsonRepo) ReadPackfile(toRev int) (io.ReadCloser, error) {
	return r.backend.ReadBlob(strconv.Itoa(toRev) + ".pack")
}
