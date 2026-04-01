package value_object

import (
	"encoding/json"
	"errors"
)

type ExtractionSchema struct {
	raw json.RawMessage
}

func NewExtractionSchema(raw json.RawMessage) (ExtractionSchema, error) {
	if len(raw) == 0 {
		return ExtractionSchema{}, nil
	}
	var check interface{}
	if err := json.Unmarshal(raw, &check); err != nil {
		return ExtractionSchema{}, errors.New("extraction_schema must be valid JSON")
	}
	return ExtractionSchema{raw: raw}, nil
}

func (es ExtractionSchema) Raw() json.RawMessage { return es.raw }
func (es ExtractionSchema) IsEmpty() bool        { return len(es.raw) == 0 }
