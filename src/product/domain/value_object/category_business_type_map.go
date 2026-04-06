package value_object

// CategoryBusinessTypeMapping maps a source category to its business type code and name.
type CategoryBusinessTypeMapping struct {
	BusinessTypeCode string
	BusinessTypeName string
}

var categoryToBusinessType = map[string]CategoryBusinessTypeMapping{
	"supermercado":           {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"supermercado_mayorista": {BusinessTypeCode: "almacen_mayorista", BusinessTypeName: "Almacén / Mayorista"},
	"electronica":           {BusinessTypeCode: "electronica", BusinessTypeName: "Electrónica"},
	"farmacia_perfumeria":   {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"ferreteria_construccion": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"construccion_hogar":    {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"indumentaria":          {BusinessTypeCode: "indumentaria", BusinessTypeName: "Indumentaria"},
}

// MapCategoryToBusinessType returns the business type assignment for a source category.
// Returns the assignment and true if found, zero value and false if the category is unknown.
func MapCategoryToBusinessType(category string) (BusinessTypeAssignment, bool) {
	mapping, ok := categoryToBusinessType[category]
	if !ok {
		return BusinessTypeAssignment{}, false
	}
	assignment, err := NewBusinessTypeAssignment(mapping.BusinessTypeCode, mapping.BusinessTypeName)
	if err != nil {
		return BusinessTypeAssignment{}, false
	}
	return assignment, true
}
