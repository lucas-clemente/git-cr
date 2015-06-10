package git

import (
	"bytes"
	"errors"
	"io"
)

const pullCapabilities = "multi_ack_detailed side-band-64k thin-pack"
const pushCapabilities = "delete-refs ofs-delta"

var (
	// ErrorInvalidHandshake occurs if the client presents an invalid handshake
	ErrorInvalidHandshake = errors.New("invalid handshake from client")
	// ErrorInvalidWantLine occurs if the client sends an invalid want line
	ErrorInvalidWantLine = errors.New("invalid `want` line sent by client")
	// ErrorInvalidHaveLine occurs if the client sends an invalid have line
	ErrorInvalidHaveLine = errors.New("invalid `have` line sent by client")
)

// A GitOperation can either be a pull or push
type GitOperation int

const (
	// GitPull is a pull
	GitPull GitOperation = iota
	// GitPush is a push
	GitPush
)

// GitServer handles the git protocol
type GitServer struct {
	Repo string
	Host string

	out Encoder
	in  Decoder

	backend Backend
}

// NewGitServer makes a handler for the git protocol
func NewGitServer(out Encoder, in Decoder, backend Backend) *GitServer {
	return &GitServer{
		out:     out,
		in:      in,
		backend: backend,
	}
}

// ReceiveHandshake reads repo and host info from the client
func (h *GitServer) ReceiveHandshake() (GitOperation, error) {
	// format: "git-[upload|receive]-pack repo-name\0host=host-name"
	var handshake []byte
	var op GitOperation

	if err := h.in.Decode(&handshake); err != nil {
		return 0, err
	}

	if bytes.HasPrefix(handshake, []byte("git-upload-pack ")) {
		op = GitPull
		handshake = handshake[len("git-upload-pack "):]
	} else if bytes.HasPrefix(handshake, []byte("git-receive-pack ")) {
		op = GitPush
		handshake = handshake[len("git-receive-pack "):]
	} else {
		return 0, ErrorInvalidHandshake
	}

	nullPos := bytes.IndexByte(handshake, 0)
	if nullPos == -1 || nullPos == 0 {
		return 0, ErrorInvalidHandshake
	}
	h.Repo = string(handshake[:nullPos])
	handshake = handshake[nullPos+1:]

	if !bytes.HasPrefix(handshake, []byte("host=")) {
		return 0, ErrorInvalidHandshake
	}
	handshake = handshake[len("host="):]
	if len(handshake) == 0 {
		return 0, ErrorInvalidHandshake
	}
	h.Host = string(handshake)

	return op, nil
}

// SendRefs sends the given references to the client
func (h *GitServer) SendRefs(refs []Ref, op GitOperation) error {
	for i, r := range refs {
		line := r.Sha1 + " " + r.Name
		if i == 0 {
			if op == GitPull {
				line += "\000" + pullCapabilities
			} else {
				line += "\000" + pushCapabilities
			}
		}
		if err := h.out.Encode([]byte(line)); err != nil {
			return err
		}
	}

	return h.out.Encode(nil)
}

// ReceiveClientWants receives the requested refs from the client
func (h *GitServer) ReceiveClientWants() ([]string, error) {
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
func (h *GitServer) HandleClientHaves(wants []string) ([]Delta, error) {
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

// SendPackfile sends a packfile using the side-band-64k encoding
func (h *GitServer) SendPackfile(r io.Reader) error {
	for {
		line := make([]byte, 65520)
		line[0] = 1
		n, err := r.Read(line[1:])
		if n != 0 {
			h.out.Encode(line[0 : n+1])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}