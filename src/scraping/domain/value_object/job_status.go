package value_object

import "fmt"

type JobStatus struct {
	value string
}

var validStatuses = map[string]bool{
	"pending": true, "running": true, "completed": true,
	"failed": true, "cancelled": true,
}

func NewJobStatus(v string) (JobStatus, error) {
	if !validStatuses[v] {
		return JobStatus{}, fmt.Errorf("invalid job status: %q", v)
	}
	return JobStatus{value: v}, nil
}

func JobStatusPending() JobStatus    { return JobStatus{value: "pending"} }
func JobStatusRunning() JobStatus    { return JobStatus{value: "running"} }
func JobStatusCompleted() JobStatus  { return JobStatus{value: "completed"} }
func JobStatusFailed() JobStatus     { return JobStatus{value: "failed"} }
func JobStatusCancelled() JobStatus  { return JobStatus{value: "cancelled"} }

func (js JobStatus) Value() string   { return js.value }
func (js JobStatus) String() string  { return js.value }
func (js JobStatus) IsPending() bool   { return js.value == "pending" }
func (js JobStatus) IsRunning() bool   { return js.value == "running" }
func (js JobStatus) IsTerminal() bool  { return js.value == "completed" || js.value == "failed" || js.value == "cancelled" }
