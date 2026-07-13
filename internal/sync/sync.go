// Package sync defines the report shape, dirty-tree policy, and seam interfaces
// composed by `kira sync`. The reindex step is a seam: core injects a real
// reindexer backed by the index; the no-op stays for callers that run without
// one and reports the step as skipped.
package sync

type StepStatus string

const (
	StepDone    StepStatus = "done"
	StepSkipped StepStatus = "skipped"
	StepFailed  StepStatus = "failed"
)

type Step struct {
	Name   string     `json:"name"`
	Status StepStatus `json:"status"`
	Detail string     `json:"detail,omitempty"`
}

type Report struct {
	Steps []Step `json:"steps"`
}

func (r *Report) Add(name string, status StepStatus, detail string) {
	r.Steps = append(r.Steps, Step{Name: name, Status: status, Detail: detail})
}

type DirtyPolicy string

const (
	DirtyAuto   DirtyPolicy = ""
	DirtyCommit DirtyPolicy = "commit"
	DirtyStash  DirtyPolicy = "stash"
)

// Reindexer rebuilds the derived index after a pull. Its own Step reports how
// the step went, so a no-op can honestly declare itself skipped.
type Reindexer interface {
	Reindex() Step
}

type NoopReindexer struct{}

func (NoopReindexer) Reindex() Step {
	return Step{Name: "reindex", Status: StepSkipped, Detail: "no reindexer configured"}
}
