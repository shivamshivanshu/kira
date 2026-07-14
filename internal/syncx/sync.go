// Package syncx defines the report shape, dirty-tree policy, and seam interfaces
// composed by `kira sync`. The reindex step is a seam: core injects a real
// reindexer backed by the index.
package syncx

type StepStatus string

const (
	StepDone   StepStatus = "done"
	StepFailed StepStatus = "failed"
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
	DirtyFail   DirtyPolicy = "fail"
	DirtyCommit DirtyPolicy = "commit"
	DirtyStash  DirtyPolicy = "stash"
)

// Reindexer rebuilds the derived index after a pull. Its own Step reports how
// the step went.
type Reindexer interface {
	Reindex() Step
}
