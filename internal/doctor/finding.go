package doctor

import "github.com/shivamshivanshu/kira/internal/datamodel"

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Class string

const (
	ClassSchema        Class = "schema"
	ClassState         Class = "state"
	ClassVocab         Class = "vocab"
	ClassRef           Class = "dangling-ref"
	ClassCollision     Class = "id-collision"
	ClassCycle         Class = "epic-cycle"
	ClassRefCycle      Class = "ref-cycle"
	ClassEpicKind      Class = "epic-kind"
	ClassComment       Class = "comment"
	ClassFreshness     Class = "index-freshness"
	ClassHooks         Class = "hooks"
	ClassEnv           Class = "environment"
	ClassBoard         Class = "board"
	ClassNumberOutlier Class = "number-outlier"
	ClassStray         Class = "stray-file"
)

type CollisionKind string

const (
	CollisionLiveLive   CollisionKind = "live-live"
	CollisionLiveAlias  CollisionKind = "live-alias"
	CollisionAliasAlias CollisionKind = "alias-alias"
)

type Finding struct {
	Class     Class      `json:"class"`
	Severity  Severity   `json:"severity"`
	Path      string     `json:"path,omitempty"`
	ItemID    string     `json:"item_id,omitempty"`
	Number    string     `json:"number,omitempty"`
	Field     string     `json:"field,omitempty"`
	Message   string     `json:"message"`
	Collision *Collision `json:"collision,omitempty"`
}

func stamp(findings []Finding, path string, it *datamodel.Item) []Finding {
	for i := range findings {
		findings[i].Path = path
		if it != nil {
			if findings[i].ItemID == "" {
				findings[i].ItemID = it.ID
			}
			if findings[i].Number == "" {
				findings[i].Number = it.Number
			}
		}
	}
	return findings
}

type Collision struct {
	Value    string        `json:"value"`
	Kind     CollisionKind `json:"kind"`
	LiveIDs  []string      `json:"live_ids"`
	AliasIDs []string      `json:"alias_ids"`
	Keep     string        `json:"keep"`
}

type Summary struct {
	Error   int `json:"error"`
	Warning int `json:"warning"`
	Info    int `json:"info"`
}

type Report struct {
	OK       bool      `json:"ok"`
	Findings []Finding `json:"findings"`
	Summary  Summary   `json:"summary"`
}

func newReport(findings []Finding) *Report {
	if findings == nil {
		findings = []Finding{}
	}
	var sum Summary
	for _, f := range findings {
		switch f.Severity {
		case SeverityError:
			sum.Error++
		case SeverityWarning:
			sum.Warning++
		case SeverityInfo:
			sum.Info++
		}
	}
	return &Report{OK: sum.Error == 0, Findings: findings, Summary: sum}
}
