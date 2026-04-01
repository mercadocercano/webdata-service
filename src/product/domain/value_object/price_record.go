package value_object

import (
	"time"

	"github.com/google/uuid"
)

type PriceRecord struct {
	ProductID  uuid.UUID
	TenantID   uuid.UUID
	Price      float64
	RecordedAt time.Time
}

func NewPriceRecord(tenantID, productID uuid.UUID, price float64) PriceRecord {
	return PriceRecord{
		TenantID:   tenantID,
		ProductID:  productID,
		Price:      price,
		RecordedAt: time.Now(),
	}
}
