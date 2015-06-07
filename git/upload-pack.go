package git

import (
	"bytes"
	"errors"
)

var (
	// ErrorInvalidHandshake occurs if the client presents an invalid handshake
	ErrorInvalidHandshake = errors.New("invalid handshake from client")
)

// HandshakePull extracts repo and host info from a pull / fetch
func HandshakePull(handshake []byte) (string, string, error) {
	// format: "git-upload-pack repo-name\0host=host-name"

	if !bytes.HasPrefix(handshake, []byte("git-upload-pack ")) {
		return "", "", ErrorInvalidHandshake
	}
	handshake = handshake[len("git-upload-pack "):]

	nullPos := bytes.IndexByte(handshake, 0)
	if nullPos == -1 || nullPos == 0 {
		return "", "", ErrorInvalidHandshake
	}
	repo := string(handshake[:nullPos])
	handshake = handshake[nullPos+1:]

	if !bytes.HasPrefix(handshake, []byte("host=")) {
		return "", "", ErrorInvalidHandshake
	}
	handshake = handshake[len("host="):]
	if len(handshake) == 0 {
		return "", "", ErrorInvalidHandshake
	}
	host := string(handshake)

	return repo, host, nil
}
