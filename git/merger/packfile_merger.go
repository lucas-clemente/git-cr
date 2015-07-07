package merger

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
)

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
