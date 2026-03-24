package model

// GlobalConfig represents ~/.config/invoice/config.toml
type GlobalConfig struct {
	User     UserConfig     `toml:"user"`
	Settings SettingsConfig `toml:"settings"`
}

type UserConfig struct {
	Name    string `toml:"name"`
	Address string `toml:"address"`
}

type SettingsConfig struct {
	ExportDir       string `toml:"export_dir"`
	ActiveClient    string `toml:"active_client"`
	DefaultCurrency string `toml:"default_currency"`
}

// ClientConfig represents a per-client TOML file
type ClientConfig struct {
	Name          string     `toml:"name"`
	Address       string     `toml:"address"`
	PaymentTerms  string     `toml:"payment_terms"`
	Currency      string     `toml:"currency"`
	Tax           []TaxEntry `toml:"tax"`
	LastLineItems []LineItem `toml:"last_line_items"`
}

type TaxEntry struct {
	Name string  `toml:"name"`
	Rate float64 `toml:"rate"`
}

type LineItem struct {
	Description string  `json:"description" toml:"description"`
	Quantity    float64 `json:"quantity"    toml:"quantity"`
	Rate        int     `json:"rate"        toml:"rate"` // cents
}

// LineTotal returns quantity * rate in cents
func (li LineItem) LineTotal() int {
	return int(li.Quantity * float64(li.Rate))
}

// Invoice represents a single invoice record
type Invoice struct {
	ID           string     `json:"id"`            // "2026-003"
	ClientSlug   string     `json:"client_slug"`   // "acme-corp"
	Date         string     `json:"date"`          // "2026-03-24"
	PaymentTerms string     `json:"payment_terms"`
	Status       string     `json:"status"` // "draft" or "sent"
	LineItems    []LineItem `json:"line_items"`
}

// Subtotal returns sum of all line totals in cents
func (inv Invoice) Subtotal() int {
	total := 0
	for _, li := range inv.LineItems {
		total += li.LineTotal()
	}
	return total
}

// TaxAmount computes a single tax entry's value in cents
func TaxAmount(subtotal int, rate float64) int {
	return int(float64(subtotal)*rate + 0.5) // round half-up
}

// Total returns the invoice total including all taxes
func (inv Invoice) Total(taxes []TaxEntry) int {
	sub := inv.Subtotal()
	total := sub
	for _, t := range taxes {
		total += TaxAmount(sub, t.Rate)
	}
	return total
}

// IsSent returns true if invoice is locked
func (inv Invoice) IsSent() bool {
	return inv.Status == "sent"
}

// InvoiceStore is the root structure for invoices.json
type InvoiceStore struct {
	NextNumbers map[string]int `json:"next_numbers"` // year -> next seq number
	Invoices    []Invoice      `json:"invoices"`
}
