package git

// A Delta is the difference between two commits
type Delta interface {
}

// A Backend for git data
type Backend interface {
	// Should return a nil Delta if not found, not an error
	FindDelta(from, to string) (Delta, error)

	DeltaFromZero(to string) (Delta, error)
}
