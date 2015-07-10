package repo

import (
	"errors"
	"io"
)

// A Revision is a version of the server's state
type Revision map[string]string

// A Repo for git data
type Repo interface {
	// GetRevisions should return all revisions in chronological order
	GetRevisions() ([]Revision, error)

	SaveNewRevision(rev Revision, packfile io.Reader) error

	ReadPackfile(toRev int) (io.ReadCloser, error)
}

// ErrNotFound should be returned by Backend.ReadBlob if a blob was not found.
var ErrNotFound = errors.New("not found")

// A Backend for a crypto repo
type Backend interface {
	ReadBlob(name string) (io.ReadCloser, error)
	WriteBlob(name string, r io.Reader) error
}
