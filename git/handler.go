package git

import (
	"bytes"
	"errors"
	"io"
	"strings"
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
	// ErrorInvalidPushRefsLine occurs if the client sends an invalid line during ref update
	ErrorInvalidPushRefsLine = errors.New("invalid line sent by client during ref update")
	// ErrorNoHead occurs if a repo has no HEAD
	ErrorNoHead = errors.New("no HEAD in repo")
)

// A GitOperation can either be a pull or push
type GitOperation int

const (
	// GitPull is a pull
	GitPull GitOperation = iota + 1
	// GitPush is a push
	GitPush
)

// GitRequestHandler handles the git protocol
type GitRequestHandler struct {
	out Encoder
	in  Decoder

	repo Repo
}

// NewGitRequestHandler makes a handler for the git protocol
func NewGitRequestHandler(out Encoder, in Decoder, repo Repo) *GitRequestHandler {
	return &GitRequestHandler{
		out:  out,
		in:   in,
		repo: repo,
	}
}

// ServeRequest handles a single git request
func (h *GitRequestHandler) ServeRequest() error {
	op, err := h.ReceiveHandshake()
	if err != nil {
		return err
	}

	if op == GitPull {
		refs, err := h.repo.ReadRefs()
		if err != nil {
			return err
		}

		if err := h.SendRefs(refs, GitPull); err != nil {
			return err
		}

		wants, err := h.ReceivePullWants()
		if err != nil {
			return err
		}

		deltas, err := h.NegotiatePullPackfile(wants)
		if err != nil {
			return err
		}
		if len(deltas) != 1 {
			panic("not implemented")
		}

		deltaReader, err := h.repo.ReadPackfile(deltas[0])
		defer deltaReader.Close()
		if err != nil {
			return err
		}

		if err := h.SendPackfile(deltaReader); err != nil {
			return err
		}
	} else if op == GitPush {
		refs, err := h.repo.ReadRefs()
		if err != nil {
			return err
		}

		if err := h.SendRefs(refs, GitPush); err != nil {
			return err
		}

		refUpdates, err := h.ReceivePushRefs()
		if err != nil {
			return err
		}
		if len(refUpdates) != 1 {
			panic("not implemented")
		}

		if err := h.repo.UpdateRef(refUpdates[0]); err != nil {
			return err
		}

		if refUpdates[0].Name == "refs/heads/master" && len(refUpdates[0].NewID) > 0 {
			headUpdate := RefUpdate{Name: "HEAD", NewID: refUpdates[0].NewID}
			if err := h.repo.UpdateRef(headUpdate); err != nil {
				return err
			}
		}

		if err := h.repo.WritePackfile(refUpdates[0].OldID, refUpdates[0].NewID, h.in); err != nil {
			return err
		}
	} else {
		panic("unexpected git op")
	}

	return nil
}

// ReceiveHandshake reads repo and host info from the client
func (h *GitRequestHandler) ReceiveHandshake() (GitOperation, error) {
	// format: "git-[upload|receive]-pack repo-name\0host=host-name"
	var handshake []byte

	if err := h.in.Decode(&handshake); err != nil {
		return 0, err
	}

	if bytes.HasPrefix(handshake, []byte("git-upload-pack ")) {
		return GitPull, nil
	} else if bytes.HasPrefix(handshake, []byte("git-receive-pack ")) {
		return GitPush, nil
	}
	return 0, ErrorInvalidHandshake
}

// SendRefs sends the given references to the client
func (h *GitRequestHandler) SendRefs(refs Refs, op GitOperation) error {
	if len(refs) == 0 {
		return h.out.Encode(nil)
	}

	var caps string
	if op == GitPull {
		caps = pullCapabilities
	} else {
		caps = pushCapabilities
	}

	head, ok := refs["HEAD"]
	if !ok {
		return ErrorNoHead
	}
	if err := h.out.Encode([]byte(head + " HEAD\000" + caps)); err != nil {
		return err
	}

	for name, sha1 := range refs {
		if name == "HEAD" {
			continue
		}
		if err := h.out.Encode([]byte(sha1 + " " + name)); err != nil {
			return err
		}
	}

	return h.out.Encode(nil)
}

// ReceivePullWants receives the requested refs from the client
func (h *GitRequestHandler) ReceivePullWants() ([]string, error) {
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

		if len(line) < 45 {
			return nil, ErrorInvalidWantLine
		}
		refs = append(refs, string(line[5:45]))
	}
	return refs, nil
}

// NegotiatePullPackfile receives the client's haves and uses the repo
// to calculate the deltas that should be sent to the client
func (h *GitRequestHandler) NegotiatePullPackfile(wants []string) ([]Delta, error) {
	// multi_ack_detailed implementation
	var line []byte
	deltas := []Delta{}

	unfulfilledWants := map[string]bool{}
	for _, w := range wants {
		unfulfilledWants[w] = true
	}

	lastCommon := ""

	for {
		if err := h.in.Decode(&line); err != nil {
			return nil, err
		}

		if line == nil {
			h.out.Encode([]byte("NAK"))
			continue
		}

		if bytes.HasPrefix(line, []byte("done")) {
			if len(lastCommon) == 0 {
				h.out.Encode([]byte("NAK"))
			} else {
				h.out.Encode([]byte("ACK " + lastCommon))
			}
			break
		}

		if !bytes.HasPrefix(line, []byte("have ")) {
			return nil, ErrorInvalidHaveLine
		}

		if len(line) < 45 {
			return nil, ErrorInvalidHaveLine
		}
		have := string(line[5:45])

		// Check each unfulfilled want
		for want := range unfulfilledWants {
			delta, err := h.repo.FindDelta(have, want)
			if err == ErrorDeltaNotFound {
				continue
			}
			if err != nil {
				return nil, err
			}
			delete(unfulfilledWants, want)
			deltas = append(deltas, delta)

			if len(unfulfilledWants) != 0 {
				h.out.Encode([]byte("ACK " + have + " common"))
				lastCommon = have
			}
		}

		if len(unfulfilledWants) == 0 {
			h.out.Encode([]byte("ACK " + have + " ready"))
			lastCommon = have
		}
	}

	// Left-over wants need to be delta'd from the beginning
	for w := range unfulfilledWants {
		delta, err := h.repo.FindDelta("", w)
		if err != nil {
			return nil, err
		}
		deltas = append(deltas, delta)
	}
	return deltas, nil
}

// SendPackfile sends a packfile using the side-band-64k encoding
func (h *GitRequestHandler) SendPackfile(r io.Reader) error {
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
	return h.out.Encode(nil)
}

// ReceivePushRefs receives the references to be updates in a push from the client
func (h *GitRequestHandler) ReceivePushRefs() ([]RefUpdate, error) {
	var line []byte
	refs := []RefUpdate{}
	for {
		if err := h.in.Decode(&line); err != nil {
			return nil, err
		}

		if line == nil {
			break
		}

		parts := bytes.Split(line, []byte(" "))
		if len(parts) != 3 {
			return nil, ErrorInvalidPushRefsLine
		}

		name := string(parts[2])
		if name[len(name)-1] == 0 {
			name = name[0 : len(name)-1]
		}
		name = strings.TrimSpace(name)
		oldID := string(parts[0])
		if isNullID(oldID) {
			oldID = ""
		}
		newID := string(parts[1])
		if isNullID(newID) {
			newID = ""
		}

		refs = append(refs, RefUpdate{Name: name, OldID: oldID, NewID: newID})
	}
	return refs, nil
}

func isNullID(id string) bool {
	for _, c := range id {
		if c != '0' {
			return false
		}
	}
	return true
}
