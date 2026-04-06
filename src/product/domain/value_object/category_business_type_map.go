package value_object

// CategoryBusinessTypeMapping maps a source category to its business type code and name.
type CategoryBusinessTypeMapping struct {
	BusinessTypeCode string
	BusinessTypeName string
}

var categoryToBusinessType = map[string]CategoryBusinessTypeMapping{
	"supermercado":            {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"supermercado_mayorista":  {BusinessTypeCode: "almacen_mayorista", BusinessTypeName: "Almacén / Mayorista"},
	"electronica":            {BusinessTypeCode: "electronica", BusinessTypeName: "Electrónica"},
	"farmacia_perfumeria":    {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"ferreteria_construccion": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"construccion_hogar":     {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"indumentaria":           {BusinessTypeCode: "indumentaria", BusinessTypeName: "Indumentaria"},
	"almacen":                {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"lacteos":                {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"limpieza":               {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"bebidas":                {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"congelados":             {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"desayuno":               {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"golosinas":              {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"aceites":                {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"panaderia":              {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"snacks":                 {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"bazar":                  {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"mascotas":               {BusinessTypeCode: "almacen_supermercado", BusinessTypeName: "Almacén / Supermercado"},
	"electrodomesticos":      {BusinessTypeCode: "electronica", BusinessTypeName: "Electrónica"},
	"tecnologia":             {BusinessTypeCode: "electronica", BusinessTypeName: "Electrónica"},
	"cuidado-personal":       {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"bebes":                  {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"calzado":                {BusinessTypeCode: "indumentaria", BusinessTypeName: "Indumentaria"},
	"construccion_ferreteria": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"sanitarios":             {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
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
