package git

import (
	"errors"
	"io"
)

// ErrorDeltaNotFound should be returned by Repo implementations
var ErrorDeltaNotFound = errors.New("delta not found")

// A Delta is the difference between two commits
type Delta interface{}

// A Repo for git data
type Repo interface {
	FindDelta(from, to string) (Delta, error)
	ListAncestors(target string) ([]string, error)

	ReadRefs() (Refs, error)
	UpdateRef(update RefUpdate) error

	ReadPackfile(d Delta) (io.ReadCloser, error)
	WritePackfile(from, to string, r io.Reader) error
}
