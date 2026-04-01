package value_object

import "fmt"

type TriggerType struct {
	value string
}

var validTriggerTypes = map[string]bool{
	"scheduled": true, "manual": true,
}

func NewTriggerType(v string) (TriggerType, error) {
	if !validTriggerTypes[v] {
		return TriggerType{}, fmt.Errorf("trigger type must be scheduled or manual, got %q", v)
	}
	return TriggerType{value: v}, nil
}

func (tt TriggerType) Value() string  { return tt.value }
func (tt TriggerType) String() string { return tt.value }
