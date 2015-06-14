package git

import (
	"errors"
	"io"
)

// ErrorDeltaNotFound should be returned by Backend implementations
var ErrorDeltaNotFound = errors.New("delta not found")

// A Delta is the difference between two commits
type Delta interface{}

// A Backend for git data
type Backend interface {
	FindDelta(from, to string) (Delta, error)

	GetRefs() (Refs, error)

	ReadPackfile(d Delta) (io.ReadCloser, error)

	UpdateRef(update RefUpdate) error

	WritePackfile(from, to string, r io.Reader) error
}

// A ListingBackend is a backend that supports listing all delta
type ListingBackend interface {
	Backend

	ListAncestors(target string) ([]string, error)
}
