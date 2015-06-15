package merger

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"io"
	"io/ioutil"

	"github.com/lucas-clemente/git-cr/git"
)

// Merger is a wrapper around a git.Backend instance that merges multiple deltas into one.
// E.g. if a backend knows how to get from A -> B and B -> C, Merger builds a delta from A -> C.
type Merger struct {
	git.ListingBackend
}

var _ git.Backend = &Merger{}

// FindDelta finds a delta as described in Merger doc
func (m *Merger) FindDelta(from, to string) (git.Delta, error) {
	ancestors, err := m.ListingBackend.ListAncestors(to)
	if err != nil {
		return nil, err
	}
	for _, ancestor := range ancestors {
		if ancestor == from {
			delta, err := m.ListingBackend.FindDelta(from, to)
			if err != nil {
				return nil, err
			}
			return mergerDeltas([]git.Delta{delta}), nil
		}
		deltas, err := m.FindDelta(from, ancestor)
		if err == nil {
			delta, err := m.ListingBackend.FindDelta(ancestor, to)
			if err != nil {
				return nil, err
			}
			return append(deltas.(mergerDeltas), delta), nil
		}
		if err != git.ErrorDeltaNotFound {
			return nil, err
		}
	}

	return nil, git.ErrorDeltaNotFound
}

// ReadPackfile reads a packfile as described in Merger doc
func (m *Merger) ReadPackfile(delta git.Delta) (io.ReadCloser, error) {
	deltas := delta.(mergerDeltas)
	var packfiles [][]byte

	for _, d := range deltas {
		reader, err := m.ListingBackend.ReadPackfile(d)
		if err != nil {
			return nil, err
		}
		packfile, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		packfiles = append(packfiles, packfile)
	}

	packfile, err := MergePackfiles(packfiles)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(bytes.NewBuffer(packfile)), nil
}

type mergerDeltas []git.Delta

var _ git.Delta = &mergerDeltas{}

// MergePackfiles merges two packfiles
func MergePackfiles(packfiles [][]byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.WriteString("PACK")
	// Version 2
	buf.Write([]byte{0, 0, 0, 2})
	// Leave object count empty, will be filled later
	buf.Write([]byte{0, 0, 0, 0})

	var count uint32

	for _, pack := range packfiles {
		count += binary.BigEndian.Uint32(pack[8:12])
		buf.Write(pack[12 : len(pack)-sha1.Size])
	}

	data := buf.Bytes()
	// Write object count
	binary.BigEndian.PutUint32(data[8:12], count)
	// Write checksum
	hash := sha1.New()
	hash.Write(data)
	data = hash.Sum(data)
	return data, nil
}
