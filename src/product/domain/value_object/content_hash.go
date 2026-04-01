package value_object

import (
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
)

type ContentHash string

func GenerateContentHash(tenantID, sourceID uuid.UUID, title, url string) ContentHash {
	input := fmt.Sprintf("%s:%s:%s:%s", tenantID.String(), sourceID.String(), title, url)
	sum := sha256.Sum256([]byte(input))
	return ContentHash(fmt.Sprintf("%x", sum))
}

func (ch ContentHash) String() string { return string(ch) }
func (ch ContentHash) IsEmpty() bool  { return ch == "" }
