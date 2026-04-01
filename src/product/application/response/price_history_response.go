package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/mercadocercano/webdata-service/src/product/domain/value_object"
)

type PriceHistoryResponse struct {
	ProductID  uuid.UUID `json:"product_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Price      float64   `json:"price"`
	RecordedAt time.Time `json:"recorded_at"`
}

func FromPriceRecord(r value_object.PriceRecord) PriceHistoryResponse {
	return PriceHistoryResponse{
		ProductID:  r.ProductID,
		TenantID:   r.TenantID,
		Price:      r.Price,
		RecordedAt: r.RecordedAt,
	}
}
