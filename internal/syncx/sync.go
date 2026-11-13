// Package syncx defines the report shape and dirty-tree policy of `kira sync`.
package syncx

type StepStatus string

const (
	StepDone   StepStatus = "done"
	StepFailed StepStatus = "failed"
)

var StepStatuses = []StepStatus{StepDone, StepFailed}

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
	DirtyFail   DirtyPolicy = "fail"
	DirtyCommit DirtyPolicy = "commit"
)
