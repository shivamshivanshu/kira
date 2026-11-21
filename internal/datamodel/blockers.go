package datamodel

type BlockerItem struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

type ActionableBlocker struct {
	Item     BlockerItem `json:"item"`
	Affected int         `json:"affected"`
	Path     []string    `json:"path"`
}

type BlockerIssue struct {
	Path   []string `json:"path"`
	Reason string   `json:"reason"`
}

type BlockersResult struct {
	Item        BlockerItem         `json:"item"`
	Actionable  []ActionableBlocker `json:"actionable"`
	Direct      []ActionableBlocker `json:"direct"`
	Incomplete  []BlockerIssue      `json:"incomplete"`
	StderrNotes []Warning           `json:"-"`
}
