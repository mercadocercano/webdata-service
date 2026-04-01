package value_object

import "fmt"

type SourceTier struct {
	value int
}

func NewSourceTier(v int) (SourceTier, error) {
	if v < 1 || v > 3 {
		return SourceTier{}, fmt.Errorf("source tier must be between 1 and 3, got %d", v)
	}
	return SourceTier{value: v}, nil
}

func (st SourceTier) Value() int { return st.value }

func (st SourceTier) String() string {
	labels := map[int]string{1: "premium", 2: "standard", 3: "low"}
	return labels[st.value]
}
