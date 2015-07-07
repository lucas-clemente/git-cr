package merger

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"io"
	"io/ioutil"

	"github.com/lucas-clemente/git-cr/git"
)

type merger struct {
	git.Repo
}

var _ git.Repo = &merger{}

// NewMerger generates a git.Repo instance that merges multiple deltas into one.
// E.g. if a repo knows how to get from A -> B and B -> C, merger builds a delta from A -> C.
func NewMerger(repo git.Repo) git.Repo {
	return &merger{Repo: repo}
}

func (m *merger) ReadPackfile(fromRev, toRev int) (io.ReadCloser, error) {
	var packfiles [][]byte
	for i := fromRev; i < toRev; i++ {
		rdr, err := m.Repo.ReadPackfile(i, i+1)
		if err != nil {
			return nil, err
		}
		packfile, err := ioutil.ReadAll(rdr)
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
