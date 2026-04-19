package value_object

// CategoryBusinessTypeMapping maps a source category to its business type code and name.
type CategoryBusinessTypeMapping struct {
	BusinessTypeCode string
	BusinessTypeName string
}

var categoryToBusinessType = map[string]CategoryBusinessTypeMapping{
	// Alimentarios → almacen (código PIM: almacen)
	"supermercado":           {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"almacen":                {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"lacteos":                {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"limpieza":               {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"bebidas":                {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"congelados":             {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"desayuno":               {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"golosinas":              {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"aceites":                {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"panaderia":              {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"snacks":                 {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"bazar":                  {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"mascotas":               {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},

	// Mayoristas → supermercado (código PIM: supermercado)
	"supermercado_mayorista": {BusinessTypeCode: "supermercado", BusinessTypeName: "Supermercado"},

	// Tecnología → electronica (código PIM: electronica)
	"electronica":            {BusinessTypeCode: "electronica", BusinessTypeName: "Casa de Electrodomésticos"},
	"electrodomesticos":      {BusinessTypeCode: "electronica", BusinessTypeName: "Casa de Electrodomésticos"},
	"tecnologia":             {BusinessTypeCode: "electronica", BusinessTypeName: "Casa de Electrodomésticos"},

	// Farmacia → farmacia (código PIM: farmacia)
	"farmacia_perfumeria":    {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"cuidado-personal":       {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"bebes":                  {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},

	// Ferretería → ferreteria (código PIM: ferreteria)
	"ferreteria_construccion": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"construccion_hogar":      {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"construccion_ferreteria": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"sanitarios":              {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},

	// Indumentaria → ropa (código PIM: ropa)
	"indumentaria":           {BusinessTypeCode: "ropa", BusinessTypeName: "Tienda de Ropa"},
	"calzado":                {BusinessTypeCode: "ropa", BusinessTypeName: "Tienda de Ropa"},
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
