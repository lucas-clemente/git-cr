package git

import (
	"errors"
	"io"
)

// ErrorDeltaNotFound should be returned by Repo implementations
var ErrorDeltaNotFound = errors.New("delta not found")

// ErrorRepoEmpty should be returned by ReadRefs is a repo is empty
var ErrorRepoEmpty = errors.New("repo is empty")

// A Delta is the difference between two commits
type Delta interface{}

// A Repo for git data
type Repo interface {
	FindDelta(from, to string) (Delta, error)
	ListAncestors(target string) ([]string, error)

	ReadRefs() (io.ReadCloser, error)
	WriteRefs(io.Reader) error

	ReadPackfile(d Delta) (io.ReadCloser, error)
	WritePackfile(from, to string, r io.Reader) error
}
