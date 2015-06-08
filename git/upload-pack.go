package git

import (
	"bytes"
	"errors"
)

var (
	// ErrorInvalidHandshake occurs if the client presents an invalid handshake
	ErrorInvalidHandshake = errors.New("invalid handshake from client")
)

// UploadPackHandler handles git fetch / pull
type UploadPackHandler struct {
	Repo string
	Host string

	out Encoder
	in  Decoder
}

// NewUploadPackHandler makes a handler for a fetch/pull with the handshake line
func NewUploadPackHandler(out Encoder, in Decoder) *UploadPackHandler {
	return &UploadPackHandler{
		out: out,
		in:  in,
	}
}

// ParseHandshake reads repo and host info from the client
func (h *UploadPackHandler) ParseHandshake() error {
	// format: "git-upload-pack repo-name\0host=host-name"
	var handshake []byte

	if err := h.in.Decode(&handshake); err != nil {
		return err
	}

	if !bytes.HasPrefix(handshake, []byte("git-upload-pack ")) {
		return ErrorInvalidHandshake
	}
	handshake = handshake[len("git-upload-pack "):]

	nullPos := bytes.IndexByte(handshake, 0)
	if nullPos == -1 || nullPos == 0 {
		return ErrorInvalidHandshake
	}
	h.Repo = string(handshake[:nullPos])
	handshake = handshake[nullPos+1:]

	if !bytes.HasPrefix(handshake, []byte("host=")) {
		return ErrorInvalidHandshake
	}
	handshake = handshake[len("host="):]
	if len(handshake) == 0 {
		return ErrorInvalidHandshake
	}
	h.Host = string(handshake)

	return nil
}

// SendRefs sends the given references to the client
func (h *UploadPackHandler) SendRefs(refs []Ref) error {
	return errors.New("not implemented")
}
