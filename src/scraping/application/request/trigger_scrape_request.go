package request

import "github.com/google/uuid"

type TriggerScrapeRequest struct {
	SourceID uuid.UUID `json:"source_id"`
}
