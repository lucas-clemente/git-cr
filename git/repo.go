package git

import "io"

// A Revision is a version of the server's state
type Revision map[string]string

// A Repo for git data
type Repo interface {
	// GetRevisions should return all revisions in chronological order
	GetRevisions() ([]Revision, error)

	SaveNewRevision(rev Revision, packfile io.Reader) error

	ReadPackfile(toRev int) (io.ReadCloser, error)
}
