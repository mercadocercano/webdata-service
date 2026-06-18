package value_object

// CategoryBusinessTypeMapping maps a source category to its business type code and name.
type CategoryBusinessTypeMapping struct {
	BusinessTypeCode string
	BusinessTypeName string
}

var categoryToBusinessType = map[string]CategoryBusinessTypeMapping{
	// Almacén de Barrio → almacen
	"supermercado":           {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"almacen":                {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"congelados":             {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"desayuno":               {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"golosinas":              {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"aceites":                {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"panaderia":              {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"snacks":                 {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	// Cordiez VTEX categories → almacen
	"desayuno_merienda":      {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"kiosco_cordiez":         {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},

	// Vinoteca → vinoteca (E19: Cordiez Bebidas VTEX id=10001 → vinoteca, no almacen)
	"bebidas":                {BusinessTypeCode: "vinoteca", BusinessTypeName: "Vinoteca"},

	// Veterinaria y Mascotas → veterinaria (E19: Cordiez Mascotas VTEX id=10010 → veterinaria, no almacen)
	"mascotas":               {BusinessTypeCode: "veterinaria", BusinessTypeName: "Veterinaria y Mascotas"},

	// Fiambrería → fiambreria (E19: lácteos y fiambres/quesos van a fiambrería)
	"lacteos":                {BusinessTypeCode: "fiambreria", BusinessTypeName: "Fiambrería"},
	"fiambres_quesos":        {BusinessTypeCode: "fiambreria", BusinessTypeName: "Fiambrería"},

	// Kiosco → kiosco
	"kiosco":                 {BusinessTypeCode: "kiosco", BusinessTypeName: "Kiosco"},

	// Limpieza → limpieza (E19: limpieza_hogar y cuidado_ropa van a limpieza)
	"limpieza":               {BusinessTypeCode: "limpieza", BusinessTypeName: "Limpieza"},
	"limpieza_hogar":         {BusinessTypeCode: "limpieza", BusinessTypeName: "Limpieza"},
	"cuidado_ropa":           {BusinessTypeCode: "limpieza", BusinessTypeName: "Limpieza"},

	// Bazar → bazar
	"bazar":                  {BusinessTypeCode: "bazar", BusinessTypeName: "Bazar"},

	// Mayoristas → supermercado
	"supermercado_mayorista": {BusinessTypeCode: "supermercado", BusinessTypeName: "Supermercado"},

	// Tecnología → electrodomesticos
	"electronica":            {BusinessTypeCode: "electrodomesticos", BusinessTypeName: "Casa de Electrodomésticos"},
	"electrodomesticos":      {BusinessTypeCode: "electrodomesticos", BusinessTypeName: "Casa de Electrodomésticos"},
	"tecnologia":             {BusinessTypeCode: "electrodomesticos", BusinessTypeName: "Casa de Electrodomésticos"},

	// Farmacia → farmacia
	"farmacia_perfumeria":    {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"cuidado-personal":       {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"cuidado_personal":       {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"bebes":                  {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},

	// Perfumería → perfumeria
	"perfumeria":             {BusinessTypeCode: "perfumeria", BusinessTypeName: "Perfumería"},

	// Ferretería → ferreteria
	"ferreteria_construccion": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"construccion_hogar":      {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"construccion_ferreteria": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"sanitarios":              {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},

	// Indumentaria → ropa
	"indumentaria":           {BusinessTypeCode: "ropa", BusinessTypeName: "Tienda de Ropa"},
	"calzado":                {BusinessTypeCode: "ropa", BusinessTypeName: "Tienda de Ropa"},

	// E22: Día (VTEX) — 8 categorías raíz con business_type por rubro real
	"dia_almacen":      {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"dia_desayuno":     {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"dia_frescos":      {BusinessTypeCode: "fiambreria", BusinessTypeName: "Fiambrería"},
	"dia_bebidas":      {BusinessTypeCode: "vinoteca", BusinessTypeName: "Vinoteca"},
	"dia_congelados":   {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"dia_perfumeria":   {BusinessTypeCode: "perfumeria", BusinessTypeName: "Perfumería"},
	"dia_limpieza":     {BusinessTypeCode: "limpieza", BusinessTypeName: "Limpieza"},
	"dia_mascotas":     {BusinessTypeCode: "veterinaria", BusinessTypeName: "Veterinaria y Mascotas"},

	// E22: Carrefour (VTEX) — 9 categorías raíz con business_type por rubro real
	"carr_almacen":    {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"carr_desayuno":   {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"carr_bebidas":    {BusinessTypeCode: "vinoteca", BusinessTypeName: "Vinoteca"},
	"carr_lacteos":    {BusinessTypeCode: "fiambreria", BusinessTypeName: "Fiambrería"},
	"carr_panaderia":  {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"carr_congelados": {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"carr_limpieza":   {BusinessTypeCode: "limpieza", BusinessTypeName: "Limpieza"},
	"carr_farmacia":   {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"carr_mascotas":   {BusinessTypeCode: "veterinaria", BusinessTypeName: "Veterinaria y Mascotas"},

	// E22: La Anónima / MasOnline (VTEX) — categorías granulares (117 nivel-1)
	"anon_aceites_aderezos": {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"anon_arroz_pastas":     {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"anon_desayunos":        {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"anon_lacteos":          {BusinessTypeCode: "fiambreria", BusinessTypeName: "Fiambrería"},
	"anon_fiambres":         {BusinessTypeCode: "fiambreria", BusinessTypeName: "Fiambrería"},
	"anon_congelados":       {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	"anon_gaseosas":         {BusinessTypeCode: "vinoteca", BusinessTypeName: "Vinoteca"},
	"anon_limpieza":         {BusinessTypeCode: "limpieza", BusinessTypeName: "Limpieza"},
	"anon_farmacia":         {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},

	// E22: Farmacity (VTEX) — farmacia/perfumería/cuidado personal
	"farm_farmacia":         {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"farm_medicamentos":     {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"farm_cuidado_personal": {BusinessTypeCode: "farmacia", BusinessTypeName: "Farmacia"},
	"farm_cuidado_piel":     {BusinessTypeCode: "perfumeria", BusinessTypeName: "Perfumería"},
	"farm_cuidado_capilar":  {BusinessTypeCode: "perfumeria", BusinessTypeName: "Perfumería"},
	"farm_perfumes":         {BusinessTypeCode: "perfumeria", BusinessTypeName: "Perfumería"},
	"farm_maquillaje":       {BusinessTypeCode: "perfumeria", BusinessTypeName: "Perfumería"},
	"farm_hogar_alimentos":  {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},

	// Coop. Obrera (E21) — 6 categorías raíz con business_type por rubro real
	// Almacén (id=2) → almacen (ya cubierto por "almacen" arriba)
	"coop_almacen":           {BusinessTypeCode: "almacen", BusinessTypeName: "Almacén de Barrio"},
	// Frescos (id=3) → fiambreria (lácteos, quesos, fiambres, embutidos)
	"coop_frescos":           {BusinessTypeCode: "fiambreria", BusinessTypeName: "Fiambrería"},
	// Bebidas (id=4) → vinoteca (igual que Cordiez Bebidas, E19)
	"coop_bebidas":           {BusinessTypeCode: "vinoteca", BusinessTypeName: "Vinoteca"},
	// Perfumería (id=5) → perfumeria (NOT farmacia — Coop la llama Perfumería)
	"coop_perfumeria":        {BusinessTypeCode: "perfumeria", BusinessTypeName: "Perfumería"},
	// Limpieza (id=6) → limpieza (productos de limpieza del hogar)
	"coop_limpieza":          {BusinessTypeCode: "limpieza", BusinessTypeName: "Limpieza"},
	// Casa y Jardín (id=7) → bazar (artículos del hogar, utensilios, decoración)
	"coop_casa_jardin":       {BusinessTypeCode: "bazar", BusinessTypeName: "Bazar"},

	// ── E26: Easy (VTEX) — hogar/construcción, rubros del piloto ──
	// IDs nivel-2 confirmados via /api/catalog_system/pub/category/tree/2 (2026-06-17).
	// Ferretería (rubro núcleo del piloto):
	"easy_ferreteria":     {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"easy_herramientas":   {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"easy_construccion":   {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"easy_plomeria":       {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"easy_aberturas":      {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"easy_pisos":          {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"easy_pinturas":       {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"easy_banos_cocinas":  {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	// Electricidad:
	"easy_electricidad":   {BusinessTypeCode: "electricidad", BusinessTypeName: "Electricidad"},
	"easy_iluminacion":    {BusinessTypeCode: "electricidad", BusinessTypeName: "Electricidad"},
	// Bazar:
	"easy_bazar_deco":     {BusinessTypeCode: "bazar", BusinessTypeName: "Bazar"},
	"easy_muebles":        {BusinessTypeCode: "bazar", BusinessTypeName: "Bazar"},
	// Electrodomésticos:
	"easy_electrodomesticos": {BusinessTypeCode: "electrodomesticos", BusinessTypeName: "Casa de Electrodomésticos"},
	// Jardín y Aire Libre → piletas (particionado por subcategoría, cap VTEX 2500):
	"easy_jardin":              {BusinessTypeCode: "piletas", BusinessTypeName: "Piletas y Jardín"},
	"easy_jardin_muebles_ext":  {BusinessTypeCode: "piletas", BusinessTypeName: "Piletas y Jardín"},
	"easy_jardin_parrillas":    {BusinessTypeCode: "piletas", BusinessTypeName: "Piletas y Jardín"},
	"easy_jardin_piletas":      {BusinessTypeCode: "piletas", BusinessTypeName: "Piletas y Jardín"},
	"easy_jardin_tiempolibre":  {BusinessTypeCode: "piletas", BusinessTypeName: "Piletas y Jardín"},
	"easy_jardin_camping":      {BusinessTypeCode: "piletas", BusinessTypeName: "Piletas y Jardín"},
	"easy_jardin_herramientas": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},
	"easy_jardin_armados":      {BusinessTypeCode: "piletas", BusinessTypeName: "Piletas y Jardín"},
	"easy_jardin_mascotas":     {BusinessTypeCode: "veterinaria", BusinessTypeName: "Veterinaria y Mascotas"},

	// ── E26: Blaisten (VTEX) — baños, pisos, griferías, sanitarios → ferreteria ──
	"blaisten_general": {BusinessTypeCode: "ferreteria", BusinessTypeName: "Ferretería"},

	// ── E26: Puppis (VTEX) — alimento/accesorios mascotas → veterinaria ──
	"puppis_general": {BusinessTypeCode: "veterinaria", BusinessTypeName: "Veterinaria y Mascotas"},
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
