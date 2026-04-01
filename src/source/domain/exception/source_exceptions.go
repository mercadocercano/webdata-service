package exception

import "fmt"

type SourceNotFoundError struct {
	ID string
}

func (e SourceNotFoundError) Error() string {
	return fmt.Sprintf("source not found: %s", e.ID)
}

type DuplicateSourceNameError struct {
	Name     string
	TenantID string
}

func (e DuplicateSourceNameError) Error() string {
	return fmt.Sprintf("source with name %q already exists for tenant %s", e.Name, e.TenantID)
}

type InvalidSourceError struct {
	Field   string
	Message string
}

func (e InvalidSourceError) Error() string {
	return fmt.Sprintf("invalid source field %q: %s", e.Field, e.Message)
}
