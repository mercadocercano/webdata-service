package entity

type ProductData struct {
	GTIN     string
	EAN      string
	Name     string
	Brand    string
	Category string
	ImageURL string
	Source   Source
}
