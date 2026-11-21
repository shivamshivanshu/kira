package datamodel

type ReviewResult struct {
	Base         string               `json:"base"`
	MergeBase    string               `json:"merge_base"`
	Head         string               `json:"head"`
	LandedRef    string               `json:"landed_ref"`
	Linked       []ReviewLink         `json:"linked"`
	StateChanges []ReviewStateChange  `json:"state_changes"`
	StateEvents  []ReviewStateHistory `json:"state_events"`
	Closes       []ReviewClose        `json:"closes"`
	Unresolved   []ReviewUnresolved   `json:"unresolved"`
	StderrNotes  []Warning            `json:"-"`
}

type ReviewTicket struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Title  string `json:"title"`
}

type ReviewLink struct {
	Ticket  ReviewTicket   `json:"ticket"`
	Commits []ReviewCommit `json:"commits"`
}

type ReviewCommit struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
}

type ReviewStateChange struct {
	Ticket ReviewTicket `json:"ticket"`
	From   string       `json:"from"`
	To     string       `json:"to"`
}

type ReviewStateHistory struct {
	Ticket ReviewTicket `json:"ticket"`
	Events []Event      `json:"events"`
}

const (
	ReviewCloseEligible         = "eligible"
	ReviewCloseAlreadyLanded    = "already_landed"
	ReviewCloseAlreadyClosed    = "already_closed"
	ReviewCloseInvalidTimestamp = "invalid_timestamp"
	ReviewCloseTicketNewer      = "ticket_newer"
	ReviewCloseNoTarget         = "no_done_target"
)

type ReviewClose struct {
	Ticket    ReviewTicket `json:"ticket"`
	CommitSHA string       `json:"commit_sha"`
	Outcome   string       `json:"outcome"`
	Target    string       `json:"target,omitempty"`
}

type ReviewUnresolved struct {
	Ref       string `json:"ref"`
	Source    string `json:"source"`
	CommitSHA string `json:"commit_sha"`
	Ambiguous bool   `json:"ambiguous"`
}
