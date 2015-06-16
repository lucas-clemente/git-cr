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

	GetRefs() (Refs, error)

	ReadPackfile(d Delta) (io.ReadCloser, error)

	UpdateRef(update RefUpdate) error

	WritePackfile(from, to string, r io.Reader) error
}

// A ListingRepo is a repo that supports listing all deltas
type ListingRepo interface {
	Repo

	ListAncestors(target string) ([]string, error)
}
