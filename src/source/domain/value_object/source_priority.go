package value_object

import "fmt"

type SourcePriority struct {
	value string
}

var validPriorities = map[string]bool{
	"low": true, "medium": true, "high": true,
}

func NewSourcePriority(v string) (SourcePriority, error) {
	if !validPriorities[v] {
		return SourcePriority{}, fmt.Errorf("source priority must be low, medium or high, got %q", v)
	}
	return SourcePriority{value: v}, nil
}

func (sp SourcePriority) Value() string { return sp.value }
func (sp SourcePriority) String() string { return sp.value }
