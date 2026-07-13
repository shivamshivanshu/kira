package datamodel

type WorkonResult struct {
	ID            string `json:"id"`
	Number        string `json:"number"`
	Branch        string `json:"branch"`
	BranchCreated bool   `json:"branch_created"`
	Worktree      string `json:"worktree,omitempty"`
	Moved         bool   `json:"moved"`
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
}
