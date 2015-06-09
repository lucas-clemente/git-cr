package git

import (
	"bytes"
	"errors"
)

const capabilities = "multi_ack_detailed"

var (
	// ErrorInvalidHandshake occurs if the client presents an invalid handshake
	ErrorInvalidHandshake = errors.New("invalid handshake from client")
	// ErrorInvalidWantLine occurs if the client sends an invalid want line
	ErrorInvalidWantLine = errors.New("invalid `want` line sent by client")
	// ErrorInvalidHaveLine occurs if the client sends an invalid have line
	ErrorInvalidHaveLine = errors.New("invalid `have` line sent by client")
)

// UploadPackHandler handles git fetch / pull
type UploadPackHandler struct {
	Repo string
	Host string

	out Encoder
	in  Decoder

	backend Backend
}

// NewUploadPackHandler makes a handler for a fetch/pull with the handshake line
func NewUploadPackHandler(out Encoder, in Decoder, backend Backend) *UploadPackHandler {
	return &UploadPackHandler{
		out:     out,
		in:      in,
		backend: backend,
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
	for _, r := range refs {
		if err := h.out.Encode([]byte(r.Sha1 + " " + r.Name)); err != nil {
			return err
		}
	}

	return h.out.Encode(nil)
}

// ReceiveClientWants receives the requested refs from the client
func (h *UploadPackHandler) ReceiveClientWants() ([]string, error) {
	refs := []string{}
	var line []byte
	for {
		if err := h.in.Decode(&line); err != nil {
			return nil, err
		}

		if line == nil {
			break
		}

		if !bytes.HasPrefix(line, []byte("want ")) {
			return nil, ErrorInvalidWantLine
		}

		line = line[5:]

		if nullPos := bytes.IndexByte(line, 0); nullPos != -1 {
			line = line[0:nullPos]
		}

		refs = append(refs, string(line))
	}
	return refs, nil
}

// HandleClientHaves receives the client's haves and uses the backend
// to calculate the deltas that should be sent to the client
func (h *UploadPackHandler) HandleClientHaves(wants []string) ([]Delta, error) {
	// multi_ack_detailed implementation
	var line []byte
	deltas := []Delta{}

	unfulfilledWants := map[string]bool{}
	for _, w := range wants {
		unfulfilledWants[w] = true
	}

	for {
		if err := h.in.Decode(&line); err != nil {
			return nil, err
		}

		if line == nil {
			h.out.Encode([]byte("NACK"))
			continue
		}

		if bytes.Equal(line, []byte("done")) {
			h.out.Encode([]byte("NACK"))
			break
		}

		if !bytes.HasPrefix(line, []byte("have ")) {
			return nil, ErrorInvalidHaveLine
		}
		line = line[5:]

		have := string(line)

		// Check each unfulfilled want
		for want := range unfulfilledWants {
			delta, err := h.backend.FindDelta(have, want)
			if err != nil {
				return nil, err
			}
			if delta != nil {
				delete(unfulfilledWants, want)
				deltas = append(deltas, delta)

				if len(unfulfilledWants) != 0 {
					h.out.Encode([]byte("ACK " + have + " common"))
				}
			}
		}

		if len(unfulfilledWants) == 0 {
			h.out.Encode([]byte("ACK " + have + " ready"))
		}
	}

	// Left-over wants need to be delta'd from the beginning
	for w := range unfulfilledWants {
		delta, err := h.backend.DeltaFromZero(w)
		if err != nil {
			return nil, err
		}
		deltas = append(deltas, delta)
	}

	return deltas, nil
}
