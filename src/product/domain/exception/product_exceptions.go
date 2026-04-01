package exception

import "fmt"

type ProductNotFoundError struct {
	ID string
}

func (e ProductNotFoundError) Error() string {
	return fmt.Sprintf("product not found: %s", e.ID)
}

type DuplicateProductError struct {
	ContentHash string
}

func (e DuplicateProductError) Error() string {
	return fmt.Sprintf("product with content_hash %q already exists", e.ContentHash)
}
