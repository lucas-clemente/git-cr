package git

// Refs is a list of ref in git, each mapping from the ref name to the commit SHA1
type Refs map[string]string

// A RefUpdate is sent by the client during pushes
type RefUpdate struct {
	Name  string
	OldID string
	NewID string
}
