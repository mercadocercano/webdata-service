package value_object

import (
	"fmt"
	"time"
)

type BusinessTypeAssignment struct {
	BusinessTypeCode string
	BusinessTypeName string
	CreatedAt        time.Time
}

func NewBusinessTypeAssignment(code, name string) (BusinessTypeAssignment, error) {
	if code == "" {
		return BusinessTypeAssignment{}, fmt.Errorf("business type code is required")
	}
	return BusinessTypeAssignment{
		BusinessTypeCode: code,
		BusinessTypeName: name,
		CreatedAt:        time.Now(),
	}, nil
}
