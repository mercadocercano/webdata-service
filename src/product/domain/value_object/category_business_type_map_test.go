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
		{"supermercado", "almacen", "Almacén de Barrio"},
		{"almacen", "almacen", "Almacén de Barrio"},
		{"supermercado_mayorista", "supermercado", "Supermercado"},
		{"electronica", "electronica", "Casa de Electrodomésticos"},
		{"farmacia_perfumeria", "farmacia", "Farmacia"},
		{"ferreteria_construccion", "ferreteria", "Ferretería"},
		{"construccion_hogar", "ferreteria", "Ferretería"},
		{"indumentaria", "ropa", "Tienda de Ropa"},
		{"calzado", "ropa", "Tienda de Ropa"},
		{"lacteos", "almacen", "Almacén de Barrio"},
		{"cuidado-personal", "farmacia", "Farmacia"},
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
