package value_object_test

import (
	"testing"

	vo "github.com/mercadocercano/webdata-service/src/product/domain/value_object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapCategoryToBusinessType_AllKnownCategories(t *testing.T) {
	cases := []struct {
		category         string
		expectedCode     string
		expectedName     string
	}{
		{"supermercado", "almacen_supermercado", "Almacén / Supermercado"},
		{"supermercado_mayorista", "almacen_mayorista", "Almacén / Mayorista"},
		{"electronica", "electronica", "Electrónica"},
		{"farmacia_perfumeria", "farmacia", "Farmacia"},
		{"ferreteria_construccion", "ferreteria", "Ferretería"},
		{"construccion_hogar", "ferreteria", "Ferretería"},
		{"indumentaria", "indumentaria", "Indumentaria"},
	}

	for _, tc := range cases {
		t.Run(tc.category, func(t *testing.T) {
			assignment, ok := vo.MapCategoryToBusinessType(tc.category)
			require.True(t, ok, "category %q should be mapped", tc.category)
			assert.Equal(t, tc.expectedCode, assignment.BusinessTypeCode)
			assert.Equal(t, tc.expectedName, assignment.BusinessTypeName)
			assert.False(t, assignment.CreatedAt.IsZero(), "CreatedAt should be set")
		})
	}
}

func TestMapCategoryToBusinessType_UnknownCategory_ReturnsFalse(t *testing.T) {
	_, ok := vo.MapCategoryToBusinessType("unknown_category")
	assert.False(t, ok)
}

func TestMapCategoryToBusinessType_EmptyCategory_ReturnsFalse(t *testing.T) {
	_, ok := vo.MapCategoryToBusinessType("")
	assert.False(t, ok)
}
