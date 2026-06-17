package value_object

import "strings"

// IsValidEAN13 reports whether s is a syntactically valid EAN-13 code:
// exactly 13 ASCII digits. It does NOT validate the check digit — PIM owns the
// canonical validation; here we only filter out garbage (PLU codes, internal
// SKUs, short numbers) that PIM would reject with HTTP 400.
func IsValidEAN13(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 13 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// NormalizeEANForSync returns the EAN to send to PIM: the trimmed value when it
// is a valid EAN-13, otherwise an empty string. Weighable products (fiambres,
// verdulería, etc.) often carry a short PLU code in the EAN field; sending it to
// PIM fails its strict EAN-13 validation and blocks the whole sync. Dropping it
// lets the product sync and fall back to name+brand dedup.
func NormalizeEANForSync(ean string) string {
	if IsValidEAN13(ean) {
		return strings.TrimSpace(ean)
	}
	return ""
}
