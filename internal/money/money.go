package money

import (
	"fmt"
	"strings"
)

var currencySymbols = map[string]string{
	"CAD": "$",
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"AUD": "$",
	"JPY": "¥",
}

// Symbol returns the currency symbol for a code, or the code itself if unknown.
func Symbol(currency string) string {
	if s, ok := currencySymbols[strings.ToUpper(currency)]; ok {
		return s
	}
	return currency + " "
}

// FormatCents formats an integer cents value as a currency string.
// e.g. FormatCents(150025, "CAD") => "$1,500.25"
func FormatCents(cents int, currency string) string {
	sym := Symbol(currency)
	negative := cents < 0
	if negative {
		cents = -cents
	}

	dollars := cents / 100
	remainder := cents % 100

	dollarStr := formatWithCommas(dollars)

	result := fmt.Sprintf("%s%s.%02d", sym, dollarStr, remainder)
	if negative {
		result = "-" + result
	}
	return result
}

// FormatRate formats a rate in cents for display (e.g. 15000 => "$150.00")
func FormatRate(cents int, currency string) string {
	return FormatCents(cents, currency)
}

// FormatQuantity formats a quantity, dropping the decimal if it's a whole number.
func FormatQuantity(qty float64) string {
	if qty == float64(int(qty)) {
		return fmt.Sprintf("%d", int(qty))
	}
	return fmt.Sprintf("%.2f", qty)
}

func formatWithCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	offset := len(s) % 3
	if offset > 0 {
		result.WriteString(s[:offset])
	}
	for i := offset; i < len(s); i += 3 {
		if result.Len() > 0 {
			result.WriteByte(',')
		}
		result.WriteString(s[i : i+3])
	}
	return result.String()
}
