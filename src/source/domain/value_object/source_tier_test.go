package value_object_test

import (
	"testing"

	"github.com/mercadocercano/webdata-service/src/source/domain/value_object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TEST-ID: T-SRC-D02
func TestSourceTier_ValidValues_CreateSuccessfully(t *testing.T) {
	for _, tier := range []int{1, 2, 3} {
		t.Run("tier_valid", func(t *testing.T) {
			st, err := value_object.NewSourceTier(tier)
			require.NoError(t, err)
			assert.Equal(t, tier, st.Value())
		})
	}
}

func TestSourceTier_InvalidValues_ReturnError(t *testing.T) {
	for _, tier := range []int{0, 4, -1, 100} {
		t.Run("tier_invalid", func(t *testing.T) {
			_, err := value_object.NewSourceTier(tier)
			assert.Error(t, err)
		})
	}
}

func TestSourceTier_String_ReturnsLabel(t *testing.T) {
	tier1, _ := value_object.NewSourceTier(1)
	assert.Equal(t, "premium", tier1.String())

	tier2, _ := value_object.NewSourceTier(2)
	assert.Equal(t, "standard", tier2.String())

	tier3, _ := value_object.NewSourceTier(3)
	assert.Equal(t, "low", tier3.String())
}

func TestSourcePriority_ValidValues_CreateSuccessfully(t *testing.T) {
	for _, p := range []string{"low", "medium", "high"} {
		sp, err := value_object.NewSourcePriority(p)
		require.NoError(t, err)
		assert.Equal(t, p, sp.Value())
	}
}

func TestSourcePriority_InvalidValue_ReturnError(t *testing.T) {
	_, err := value_object.NewSourcePriority("critical")
	assert.Error(t, err)
}
