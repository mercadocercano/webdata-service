package exception

import "fmt"

type JobNotFoundError struct {
	ID string
}

func (e JobNotFoundError) Error() string {
	return fmt.Sprintf("scraping job not found: %s", e.ID)
}

type InvalidJobTransitionError struct {
	CurrentStatus string
	TargetStatus  string
}

func (e InvalidJobTransitionError) Error() string {
	return fmt.Sprintf("invalid job transition from %q to %q", e.CurrentStatus, e.TargetStatus)
}

type JobNotCancellableError struct {
	Status string
}

func (e JobNotCancellableError) Error() string {
	return fmt.Sprintf("job in status %q cannot be cancelled", e.Status)
}
