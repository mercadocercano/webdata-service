package value_object

import "strings"

// IsValidEAN13 reports whether s is a valid EAN-13 code: exactly 13 ASCII digits
// AND a correct mod-10 check digit. PIM validates both, so anything failing here
// is rejected with HTTP 400. We filter out two classes of garbage: PLU/short
// codes (wrong length) and corrupt retailer data (13 digits, wrong checksum, e.g.
// "7891167022278").
func IsValidEAN13(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 13 {
		return false
	}
	var sum int
	for i, r := range s {
		if r < '0' || r > '9' {
			return false
		}
		d := int(r - '0')
		// EAN-13: positions 1..12 (0-indexed 0..11) weight 1 for even index,
		// 3 for odd index; position 13 (index 12) is the check digit.
		if i == 12 {
			// check digit: total must be a multiple of 10.
			return (sum+d)%10 == 0
		}
		if i%2 == 0 {
			sum += d
		} else {
			sum += 3 * d
		}
	}
	return false // unreachable: len is exactly 13
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
