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
		// Almacén de Barrio
		{"supermercado", "almacen", "Almacén de Barrio"},
		{"almacen", "almacen", "Almacén de Barrio"},
		{"desayuno_merienda", "almacen", "Almacén de Barrio"},
		// Vinoteca — E19 cierre: Cordiez Bebidas (VTEX id=10001) → vinoteca
		{"bebidas", "vinoteca", "Vinoteca"},
		// Veterinaria y Mascotas — E19 cierre: Cordiez Mascotas (VTEX id=10010) → veterinaria
		{"mascotas", "veterinaria", "Veterinaria y Mascotas"},
		// Fiambrería — E19: lácteos y fiambres/quesos Cordiez van a fiambrería
		{"lacteos", "fiambreria", "Fiambrería"},
		{"fiambres_quesos", "fiambreria", "Fiambrería"},
		// Limpieza — E19: limpieza_hogar y cuidado_ropa Cordiez van a limpieza
		{"limpieza", "limpieza", "Limpieza"},
		{"limpieza_hogar", "limpieza", "Limpieza"},
		{"cuidado_ropa", "limpieza", "Limpieza"},
		// Kiosco
		{"kiosco", "kiosco", "Kiosco"},
		// Mayoristas
		{"supermercado_mayorista", "supermercado", "Supermercado"},
		// Tecnología
		{"electronica", "electrodomesticos", "Casa de Electrodomésticos"},
		// Farmacia
		{"farmacia_perfumeria", "farmacia", "Farmacia"},
		{"cuidado-personal", "farmacia", "Farmacia"},
		{"cuidado_personal", "farmacia", "Farmacia"},
		// Ferretería
		{"ferreteria_construccion", "ferreteria", "Ferretería"},
		{"construccion_hogar", "ferreteria", "Ferretería"},
		// Ropa
		{"indumentaria", "ropa", "Tienda de Ropa"},
		{"calzado", "ropa", "Tienda de Ropa"},
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
