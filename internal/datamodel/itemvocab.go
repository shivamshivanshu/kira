package datamodel

var (
	Subtypes    = []string{"bug", "task", "feature", "support"}
	Priorities  = []string{"P0", "P1", "P2", "P3"}
	Resolutions = []string{"done", ResolutionDropped, "duplicate", "cannot-reproduce"}
)
