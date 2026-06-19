package value_object

// E24 / ADR-005 §5 — Thin wrapper sobre go-shared/domain/businesstype.
//
// La tabla de reglas (productCategoryRules) y las funciones de normalización
// fueron promovidas a libs/go-shared/domain/businesstype como fuente de verdad
// única del ecosistema mercado-cercano.
//
// Este archivo re-exporta ResolveBusinessTypeFromProductCategory desde go-shared
// para mantener compatibilidad con los callers internos del paquete value_object
// sin requerir cambios en los archivos que usan el alias de paquete "vo".
//
// NO agregar lógica de reglas aquí. Toda modificación de la taxonomía debe
// hacerse en libs/go-shared/domain/businesstype/resolver.go.

import (
	"github.com/hornosg/go-shared/domain/businesstype"
)

// ResolveBusinessTypeFromProductCategory delega al resolver compartido de go-shared.
// Retorna (BusinessTypeAssignment, true) si la categoría matchea una regla conocida,
// (zero, false) en caso contrario.
//
// Ver github.com/hornosg/go-shared/domain/businesstype para la tabla de reglas
// y la documentación de comportamiento (orden load-bearing, guards de colisión).
func ResolveBusinessTypeFromProductCategory(rawCategory string) (BusinessTypeAssignment, bool) {
	sharedAssignment, ok := businesstype.ResolveBusinessTypeFromProductCategory(rawCategory)
	if !ok {
		return BusinessTypeAssignment{}, false
	}
	// Convertir el tipo de go-shared al tipo local para mantener compatibilidad
	// con los callers existentes que usan value_object.BusinessTypeAssignment.
	local, err := NewBusinessTypeAssignment(sharedAssignment.BusinessTypeCode, sharedAssignment.BusinessTypeName)
	if err != nil {
		return BusinessTypeAssignment{}, false
	}
	return local, true
}
