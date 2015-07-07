package handler

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/lucas-clemente/git-cr/git"
	"github.com/lucas-clemente/git-cr/git/merger"
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

	repo git.Repo
}

// A RefUpdate is a delta for a git reference
type RefUpdate struct {
	Name, OldID, NewID string
}

// NewGitRequestHandler makes a handler for the git protocol
func NewGitRequestHandler(out Encoder, in Decoder, repo git.Repo) *GitRequestHandler {
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

	revisions, err := h.repo.GetRevisions()
	if err != nil {
		return err
	}

	currentRevIndex := len(revisions) - 1
	var currentRev git.Revision
	if currentRevIndex == -1 {
		currentRev = git.Revision{}
	} else {
		currentRev = revisions[currentRevIndex]
	}

	if err := h.SendRefs(currentRev, op); err != nil {
		return err
	}

	// TODO(lucas): Split up into two functions
	if op == GitPull {
		wants, err := h.ReceivePullWants()
		if err != nil {
			return err
		}

		if len(wants) == 0 {
			return nil
		}

		fromRev, err := h.NegotiatePullPackfile(revisions)
		if err != nil {
			return err
		}

		packfiles := [][]byte{}
		for i := fromRev; i <= currentRevIndex; i++ {
			rdr, err := h.repo.ReadPackfile(i)
			if err != nil {
				return err
			}
			packfile, err := ioutil.ReadAll(rdr)
			if err != nil {
				return err
			}
			packfiles = append(packfiles, packfile)
		}

		packfile, err := merger.MergePackfiles(packfiles)
		if err != nil {
			return err
		}

		if err := h.SendPackfile(ioutil.NopCloser(bytes.NewBuffer(packfile))); err != nil {
			return err
		}
	} else if op == GitPush {
		refUpdates, err := h.ReceivePushRefs()
		if err != nil {
			return err
		}

		if len(refUpdates) == 0 {
			return nil
		}

		newRevision := git.Revision{}
		for k, v := range currentRev {
			newRevision[k] = v
		}

		for _, update := range refUpdates {
			if update.Name == "refs/heads/master" && update.NewID != "" {
				newRevision["HEAD"] = update.NewID
			}
			if update.NewID == "" {
				delete(newRevision, update.Name)
			} else {
				newRevision[update.Name] = update.NewID
			}
		}

		// Read packfile
		packfile, err := ioutil.ReadAll(h.in)
		if err != nil {
			return err
		}
		if len(packfile) == 0 {
			packfile = []byte{'P', 'A', 'C', 'K', 0, 0, 0, 2, 0, 0, 0, 0}
		}

		if err = h.repo.SaveNewRevision(newRevision, ioutil.NopCloser(bytes.NewBuffer(packfile))); err != nil {
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
func (h *GitRequestHandler) SendRefs(refs map[string]string, op GitOperation) error {
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
func (h *GitRequestHandler) NegotiatePullPackfile(revisions []git.Revision) (int, error) {
	// multi_ack_detailed implementation
	var line []byte

	// Each time we receive a have from a client, we remove it from all revisions.
	// Once we have an empty revision, we return that to be useed as a base.
	// revisionCommits has commits in reverse order.
	revisionCommits := make([]map[string]struct{}, len(revisions))
	for i, r := range revisions {
		m := make(map[string]struct{})
		for _, sha := range r {
			m[sha] = struct{}{}
		}
		revisionCommits[len(revisions)-i-1] = m
	}

	lastCommon := ""

	result := -1

	for {
		if err := h.in.Decode(&line); err != nil {
			return 0, err
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
			return 0, ErrorInvalidHaveLine
		}

		if len(line) < 45 {
			return 0, ErrorInvalidHaveLine
		}
		have := string(line[5:45])

		common := false

		// Remove have from all revisions
		for i, commits := range revisionCommits {
			oldLen := len(commits)
			delete(commits, have)
			newLen := len(commits)

			if newLen == 0 {
				result = len(revisions) - i - 1
				break
			} else if newLen != oldLen {
				common = true
			}
		}

		if result != -1 {
			h.out.Encode([]byte("ACK " + have + " ready"))
			lastCommon = have
		} else if common {
			h.out.Encode([]byte("ACK " + have + " common"))
			lastCommon = have
		}
	}

	if result == -1 {
		// From the beginning
		return 0, nil
	}

	return result, nil
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
