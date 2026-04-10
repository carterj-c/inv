package pdf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/model"
	"github.com/carter/inv/internal/money"
	"github.com/go-pdf/fpdf"
)

// Generate creates a PDF for the given invoice and returns the export and archive paths.
func Generate(inv model.Invoice, global model.GlobalConfig, client model.ClientConfig) (string, string, error) {
	cur := client.Currency
	if cur == "" {
		cur = global.Settings.DefaultCurrency
	}
	if cur == "" {
		cur = "CAD"
	}

	exportDir := global.Settings.ExportDir
	if exportDir == "" {
		exportDir = "~/invoices"
	}
	exportDir = config.ExpandPath(exportDir)

	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", "", fmt.Errorf("creating export dir: %w", err)
	}

	archiveDir := config.PDFArchiveDir()
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", "", fmt.Errorf("creating PDF archive dir: %w", err)
	}

	slug := inv.ClientSlug
	filename := fmt.Sprintf("%s_%s_%s.pdf", inv.ID, slug, inv.Date)
	exportPath := filepath.Join(exportDir, filename)
	archivePath := filepath.Join(archiveDir, filename)

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// --- Header: Your info (left) + Invoice details (right) ---
	pdf.SetFont("Helvetica", "B", 20)
	pdf.Cell(100, 10, global.User.Name)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetX(130)
	pdf.Cell(60, 10, fmt.Sprintf("Invoice %s", inv.ID))
	pdf.Ln(7)

	pdf.SetFont("Helvetica", "", 9)
	for _, line := range splitAddress(global.User.Address) {
		pdf.Cell(100, 5, line)
		pdf.Ln(5)
	}

	// Invoice meta (right side)
	pdf.SetY(30)
	pdf.SetFont("Helvetica", "", 9)
	rightX := 130.0
	pdf.SetX(rightX)
	pdf.Cell(25, 5, "Date:")
	pdf.Cell(40, 5, inv.Date)
	pdf.Ln(5)
	pdf.SetX(rightX)
	pdf.Cell(25, 5, "Terms:")
	pdf.Cell(40, 5, inv.PaymentTerms)
	pdf.Ln(12)

	// --- Bill To ---
	y := pdf.GetY()
	if y < 50 {
		y = 50
	}
	pdf.SetY(y)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.Cell(100, 5, "Bill To:")
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 9)
	pdf.Cell(100, 5, client.Name)
	pdf.Ln(5)
	for _, line := range splitAddress(client.Address) {
		pdf.Cell(100, 5, line)
		pdf.Ln(5)
	}

	pdf.Ln(10)

	// --- Line items table ---
	tableTop := pdf.GetY()
	colWidths := []float64{85, 20, 30, 35}
	headers := []string{"Description", "Qty", "Rate", "Total"}

	// Header row
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(245, 245, 245)
	for i, h := range headers {
		align := "L"
		if i > 0 {
			align = "R"
		}
		pdf.CellFormat(colWidths[i], 7, h, "", 0, align, true, 0, "")
	}
	pdf.Ln(7)

	// Separator line
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(2)

	// Data rows
	pdf.SetFont("Helvetica", "", 9)
	for _, li := range inv.LineItems {
		startY := pdf.GetY()

		// Description (may wrap)
		pdf.MultiCell(colWidths[0], 5, li.Description, "", "L", false)
		descEndY := pdf.GetY()

		// Qty, Rate, Total on same row
		pdf.SetXY(20+colWidths[0], startY)
		pdf.CellFormat(colWidths[1], 5, money.FormatQuantity(li.Quantity), "", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[2], 5, money.FormatCents(li.Rate, cur), "", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[3], 5, money.FormatCents(li.LineTotal(), cur), "", 0, "R", false, 0, "")

		if descEndY > startY+5 {
			pdf.SetY(descEndY)
		} else {
			pdf.Ln(6)
		}
	}

	// Separator
	pdf.Ln(2)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(5)

	// --- Totals ---
	subtotal := inv.Subtotal()
	totalX := 135.0
	valX := 170.0 - 35.0

	if len(client.Tax) > 0 {
		// Subtotal
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetX(totalX)
		pdf.CellFormat(30, 6, "Subtotal:", "", 0, "R", false, 0, "")
		pdf.CellFormat(35, 6, money.FormatCents(subtotal, cur), "", 0, "R", false, 0, "")
		pdf.Ln(6)

		// Tax lines
		for _, tax := range client.Tax {
			taxAmt := model.TaxAmount(subtotal, tax.Rate)
			label := fmt.Sprintf("%s (%.4g%%):", tax.Name, tax.Rate*100)
			pdf.SetX(totalX)
			pdf.CellFormat(30, 6, label, "", 0, "R", false, 0, "")
			pdf.CellFormat(35, 6, money.FormatCents(taxAmt, cur), "", 0, "R", false, 0, "")
			pdf.Ln(6)
		}

		// Separator before total
		pdf.SetX(totalX)
		pdf.Line(totalX, pdf.GetY(), 190, pdf.GetY())
		pdf.Ln(3)
	}

	// Total
	total := inv.Total(client.Tax)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetX(totalX)
	pdf.CellFormat(30, 8, "Total:", "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 8, money.FormatCents(total, cur), "", 0, "R", false, 0, "")

	// Suppress unused variable
	_ = tableTop
	_ = valX

	if err := pdf.OutputFileAndClose(exportPath); err != nil {
		return "", "", fmt.Errorf("writing PDF: %w", err)
	}

	if !samePath(exportPath, archivePath) {
		if err := copyFile(exportPath, archivePath); err != nil {
			return "", "", fmt.Errorf("archiving PDF: %w", err)
		}
	}

	return exportPath, archivePath, nil
}

func splitAddress(addr string) []string {
	parts := strings.Split(addr, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{addr}
	}
	return result
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}

	return out.Close()
}
