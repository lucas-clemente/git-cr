package git

// A Ref in git
type Ref struct {
	Name string
	Sha1 string
}

// A RefUpdate is sent by the client during pushes
type RefUpdate struct {
	Name  string
	OldID string
	NewID string
}
