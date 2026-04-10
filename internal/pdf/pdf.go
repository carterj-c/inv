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
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AddPage()

	drawHeader(pdf, inv, global)
	drawRecipient(pdf, client)

	// --- Line items table ---
	tableTop := pdf.GetY() + 10
	if tableTop < 102 {
		tableTop = 102
	}
	pdf.SetY(tableTop)

	colWidths := []float64{96, 18, 30, 30}
	headers := []string{"Description", "Qty", "Rate", "Total"}
	lineHeight := 6.0
	textColor := rgb{40, 44, 52}
	mutedText := rgb{107, 114, 128}
	lineColor := rgb{226, 232, 240}
	headerFill := rgb{241, 245, 249}
	accent := rgb{37, 99, 235}

	// Header row
	pdf.SetFont("Helvetica", "B", 10)
	setTextColor(pdf, textColor)
	setFillColor(pdf, headerFill)
	for i, h := range headers {
		align := "L"
		if i > 0 {
			align = "R"
		}
		pdf.CellFormat(colWidths[i], 8, h, "", 0, align, true, 0, "")
	}
	pdf.Ln(8)

	// Separator line
	setDrawColor(pdf, lineColor)
	pdf.SetLineWidth(0.3)
	pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
	pdf.Ln(3)

	// Data rows
	pdf.SetFont("Helvetica", "", 10)
	for _, li := range inv.LineItems {
		startY := pdf.GetY()
		pdf.SetX(18)

		// Description (may wrap)
		setTextColor(pdf, textColor)
		pdf.MultiCell(colWidths[0], lineHeight, li.Description, "", "L", false)
		descEndY := pdf.GetY()
		rowHeight := descEndY - startY
		if rowHeight < lineHeight {
			rowHeight = lineHeight
		}

		// Qty, Rate, Total on same row
		pdf.SetXY(18+colWidths[0], startY)
		setTextColor(pdf, mutedText)
		pdf.CellFormat(colWidths[1], rowHeight, money.FormatQuantity(li.Quantity), "", 0, "R", false, 0, "")
		setTextColor(pdf, textColor)
		pdf.CellFormat(colWidths[2], rowHeight, money.FormatCents(li.Rate, cur), "", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[3], rowHeight, money.FormatCents(li.LineTotal(), cur), "", 0, "R", false, 0, "")

		pdf.SetY(startY + rowHeight + 1.5)
		setDrawColor(pdf, lineColor)
		pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
		pdf.Ln(3.5)
	}

	// --- Totals ---
	subtotal := inv.Subtotal()
	drawTotals(pdf, subtotal, client.Tax, inv.Total(client.Tax), cur, accent, textColor, mutedText, lineColor)

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

type rgb struct {
	r int
	g int
	b int
}

func drawHeader(pdf *fpdf.Fpdf, inv model.Invoice, global model.GlobalConfig) {
	accent := rgb{37, 99, 235}
	textColor := rgb{17, 24, 39}
	mutedText := rgb{107, 114, 128}
	panelFill := rgb{248, 250, 252}
	lineColor := rgb{226, 232, 240}

	setFillColor(pdf, accent)
	pdf.Rect(18, 18, 34, 2.5, "F")

	setTextColor(pdf, textColor)
	pdf.SetFont("Helvetica", "B", 24)
	pdf.SetXY(18, 24)
	pdf.CellFormat(96, 9, global.User.Name, "", 0, "L", false, 0, "")

	setTextColor(pdf, accent)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(130, 23)
	pdf.CellFormat(62, 5, "INVOICE", "", 0, "R", false, 0, "")

	setTextColor(pdf, textColor)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(130, 28)
	pdf.CellFormat(62, 8, inv.ID, "", 0, "R", false, 0, "")

	setTextColor(pdf, mutedText)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(18, 36)
	pdf.MultiCell(82, 5.2, formatAddress(global.User.Address), "", "L", false)

	metaX := 122.0
	metaY := 39.0
	metaW := 70.0
	metaH := 24.0
	setFillColor(pdf, panelFill)
	setDrawColor(pdf, lineColor)
	pdf.Rect(metaX, metaY, metaW, metaH, "FD")

	writeMetaRow(pdf, metaX+4, metaY+5, "Date", inv.Date, accent, textColor)
	writeMetaRow(pdf, metaX+4, metaY+12, "Terms", inv.PaymentTerms, accent, textColor)

	pdf.SetY(72)
}

func drawRecipient(pdf *fpdf.Fpdf, client model.ClientConfig) {
	accent := rgb{37, 99, 235}
	textColor := rgb{17, 24, 39}
	mutedText := rgb{107, 114, 128}

	setTextColor(pdf, accent)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(80, 5, "BILL TO", "", 0, "L", false, 0, "")
	pdf.Ln(7)

	setTextColor(pdf, textColor)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(100, 6, client.Name, "", 0, "L", false, 0, "")
	pdf.Ln(7)

	setTextColor(pdf, mutedText)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(88, 5.2, formatAddress(client.Address), "", "L", false)
}

func drawTotals(
	pdf *fpdf.Fpdf,
	subtotal int,
	taxes []model.TaxEntry,
	total int,
	currency string,
	accent rgb,
	textColor rgb,
	mutedText rgb,
	lineColor rgb,
) {
	cardX := 126.0
	cardW := 66.0
	cardY := pdf.GetY() + 6
	cardH := 16.0
	if len(taxes) > 0 {
		cardH += float64(len(taxes)+1) * 6.5
	}

	setFillColor(pdf, rgb{248, 250, 252})
	setDrawColor(pdf, lineColor)
	pdf.Rect(cardX, cardY, cardW, cardH, "FD")

	setFillColor(pdf, accent)
	pdf.Rect(cardX, cardY, cardW, 2, "F")

	rowY := cardY + 7
	if len(taxes) > 0 {
		pdf.SetFont("Helvetica", "", 10)
		setTextColor(pdf, mutedText)
		writeMoneyRow(pdf, cardX+4, rowY, cardW-8, "Subtotal", money.FormatCents(subtotal, currency))
		rowY += 6.5

		for _, tax := range taxes {
			writeMoneyRow(
				pdf,
				cardX+4,
				rowY,
				cardW-8,
				fmt.Sprintf("%s (%.4g%%)", tax.Name, tax.Rate*100),
				money.FormatCents(model.TaxAmount(subtotal, tax.Rate), currency),
			)
			rowY += 6.5
		}

		setDrawColor(pdf, lineColor)
		pdf.Line(cardX+4, rowY, cardX+cardW-4, rowY)
		rowY += 3
	}

	setTextColor(pdf, textColor)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetXY(cardX+4, rowY)
	pdf.CellFormat((cardW-8)/2, 8, "Total", "", 0, "L", false, 0, "")
	pdf.CellFormat((cardW-8)/2, 8, money.FormatCents(total, currency), "", 0, "R", false, 0, "")
}

func writeMetaRow(pdf *fpdf.Fpdf, x, y float64, label, value string, labelColor, valueColor rgb) {
	setTextColor(pdf, labelColor)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetXY(x, y)
	pdf.CellFormat(20, 4, strings.ToUpper(label), "", 0, "L", false, 0, "")

	setTextColor(pdf, valueColor)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(42, 4, value, "", 0, "R", false, 0, "")
}

func writeMoneyRow(pdf *fpdf.Fpdf, x, y, width float64, label, value string) {
	pdf.SetXY(x, y)
	pdf.CellFormat(width*0.55, 4, label, "", 0, "L", false, 0, "")
	pdf.CellFormat(width*0.45, 4, value, "", 0, "R", false, 0, "")
}

func formatAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}

	if strings.Contains(addr, "\n") {
		lines := strings.Split(addr, "\n")
		for i := range lines {
			lines[i] = strings.TrimSpace(lines[i])
		}
		return strings.Join(lines, "\n")
	}

	parts := strings.Split(addr, ",")
	if len(parts) == 1 {
		return addr
	}

	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if i < len(parts)-1 {
			parts[i] += ","
		}
	}
	return strings.Join(parts, "\n")
}

func setTextColor(pdf *fpdf.Fpdf, c rgb) {
	pdf.SetTextColor(c.r, c.g, c.b)
}

func setFillColor(pdf *fpdf.Fpdf, c rgb) {
	pdf.SetFillColor(c.r, c.g, c.b)
}

func setDrawColor(pdf *fpdf.Fpdf, c rgb) {
	pdf.SetDrawColor(c.r, c.g, c.b)
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
