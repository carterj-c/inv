package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/model"
)

func storePath() string {
	return filepath.Join(config.DataDir(), "invoices.json")
}

// Load reads the invoice store from disk.
func Load() (model.InvoiceStore, error) {
	var s model.InvoiceStore
	data, err := os.ReadFile(storePath())
	if err != nil {
		if os.IsNotExist(err) {
			s.NextNumbers = make(map[string]int)
			return s, nil
		}
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	if s.NextNumbers == nil {
		s.NextNumbers = make(map[string]int)
	}
	return s, nil
}

// Save writes the invoice store to disk.
func Save(s model.InvoiceStore) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(storePath(), data, 0644)
}

// NextInvoiceNumber allocates and returns the next invoice number for the given year.
// Format: "YYYY-NNN" (e.g., "2026-001"). The counter is incremented immediately (burned).
func NextInvoiceNumber(s *model.InvoiceStore, year string) string {
	n := s.NextNumbers[year] + 1
	s.NextNumbers[year] = n
	return fmt.Sprintf("%s-%03d", year, n)
}

// InvoicesForClient returns invoices for a client, sorted by ID descending (newest first).
func InvoicesForClient(s model.InvoiceStore, slug string) []model.Invoice {
	var result []model.Invoice
	for _, inv := range s.Invoices {
		if inv.ClientSlug == slug {
			result = append(result, inv)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID > result[j].ID
	})
	return result
}

// FindInvoice returns a pointer to the invoice with the given ID, or nil.
func FindInvoice(s *model.InvoiceStore, id string) *model.Invoice {
	for i := range s.Invoices {
		if s.Invoices[i].ID == id {
			return &s.Invoices[i]
		}
	}
	return nil
}

// AddInvoice adds an invoice to the store.
func AddInvoice(s *model.InvoiceStore, inv model.Invoice) {
	s.Invoices = append(s.Invoices, inv)
}

// UpdateInvoice replaces the invoice with the matching ID.
func UpdateInvoice(s *model.InvoiceStore, inv model.Invoice) error {
	for i := range s.Invoices {
		if s.Invoices[i].ID == inv.ID {
			s.Invoices[i] = inv
			return nil
		}
	}
	return fmt.Errorf("invoice %s not found", inv.ID)
}

// DeleteInvoice removes a draft invoice. Returns error if sent or not found.
func DeleteInvoice(s *model.InvoiceStore, id string) error {
	for i := range s.Invoices {
		if s.Invoices[i].ID == id {
			if s.Invoices[i].Status == "sent" {
				return fmt.Errorf("cannot delete sent invoice %s", id)
			}
			s.Invoices = append(s.Invoices[:i], s.Invoices[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("invoice %s not found", id)
}

// MarkSent marks an invoice as sent (locked).
func MarkSent(s *model.InvoiceStore, id string) error {
	inv := FindInvoice(s, id)
	if inv == nil {
		return fmt.Errorf("invoice %s not found", id)
	}
	if inv.Status == "sent" {
		return fmt.Errorf("invoice %s is already sent", id)
	}
	inv.Status = "sent"
	return nil
}

// FirstLineDescription returns the first line item description or empty string.
func FirstLineDescription(inv model.Invoice) string {
	if len(inv.LineItems) > 0 {
		desc := inv.LineItems[0].Description
		// Truncate long descriptions
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		// Only use first line if multi-line
		if idx := strings.IndexByte(desc, '\n'); idx >= 0 {
			desc = desc[:idx]
		}
		return desc
	}
	return ""
}
