package summary

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Color constants for the PDF report.
var (
	colorDarkBlue  = [3]int{26, 60, 110}
	colorMedBlue   = [3]int{45, 95, 154}
	colorLightGray = [3]int{245, 245, 245}
	colorFailedBg  = [3]int{255, 235, 235}
	colorWhite     = [3]int{255, 255, 255}
	colorBlack     = [3]int{0, 0, 0}
	colorGreen     = [3]int{34, 139, 34}
	colorRed       = [3]int{200, 0, 0}
)

// RenderPDF builds a PDF document from the migration summary and returns the bytes.
func RenderPDF(summary *MigrationSummary) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.SetAutoPageBreak(true, 20)

	pdf.SetHeaderFuncMode(func() {
		addPageHeader(pdf, summary.RunID)
	}, true)
	pdf.SetFooterFunc(func() {
		addPageFooter(pdf)
	})

	pdf.AddPage()
	renderTitlePage(pdf, summary)

	for _, section := range summary.Sections {
		renderSection(pdf, section)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}
	return buf.Bytes(), nil
}

func addPageHeader(pdf *fpdf.Fpdf, runID string) {
	pdf.SetY(5)
	setColor(pdf, colorDarkBlue)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.CellFormat(0, 6, "SonarQube Migration Summary - "+runID, "", 0, "R", false, 0, "")
	pdf.Ln(8)
}

func addPageFooter(pdf *fpdf.Fpdf) {
	pdf.SetY(-15)
	pdf.SetFont("Helvetica", "", 8)
	setColor(pdf, colorBlack)
	pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "C", false, 0, "")
}

func renderTitlePage(pdf *fpdf.Fpdf, summary *MigrationSummary) {
	pdf.SetY(30)

	setColor(pdf, colorDarkBlue)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.CellFormat(0, 12, "SonarQube Migration Summary", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	setColor(pdf, colorBlack)
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 7, "Run ID: "+summary.RunID, "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 7, "Generated: "+summary.GeneratedAt.Format("2006-01-02 15:04:05"), "", 1, "C", false, 0, "")
	pdf.Ln(10)

	renderExecutiveSummary(pdf, summary.Sections)
}

func renderExecutiveSummary(pdf *fpdf.Fpdf, sections []Section) {
	headers := []string{"Section", "Succeeded", "Failed", "Skipped", "Total"}
	widths := []float64{55, 30, 30, 30, 30}

	setFillColor(pdf, colorDarkBlue)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 10)
	for i, h := range headers {
		align := "C"
		if i == 0 {
			align = "L"
		}
		pdf.CellFormat(widths[i], 8, h, "1", 0, align, true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 10)
	var totalS, totalF, totalSk int
	for i, sec := range sections {
		s, f, sk := len(sec.Succeeded), len(sec.Failed), len(sec.Skipped)
		totalS += s
		totalF += f
		totalSk += sk

		if i%2 == 0 {
			setFillColor(pdf, colorLightGray)
		} else {
			setFillColor(pdf, colorWhite)
		}
		setColor(pdf, colorBlack)
		pdf.CellFormat(widths[0], 7, sec.Name, "1", 0, "L", true, 0, "")
		renderCountCell(pdf, widths[1], s, colorGreen)
		renderCountCell(pdf, widths[2], f, colorRed)
		setColor(pdf, colorBlack)
		pdf.CellFormat(widths[3], 7, itoa(sk), "1", 0, "C", true, 0, "")
		pdf.CellFormat(widths[4], 7, itoa(s+f+sk), "1", 0, "C", true, 0, "")
		pdf.Ln(-1)
	}

	// Totals row
	setFillColor(pdf, colorDarkBlue)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(widths[0], 8, "Total", "1", 0, "L", true, 0, "")
	pdf.CellFormat(widths[1], 8, itoa(totalS), "1", 0, "C", true, 0, "")
	pdf.CellFormat(widths[2], 8, itoa(totalF), "1", 0, "C", true, 0, "")
	pdf.CellFormat(widths[3], 8, itoa(totalSk), "1", 0, "C", true, 0, "")
	pdf.CellFormat(widths[4], 8, itoa(totalS+totalF+totalSk), "1", 0, "C", true, 0, "")
	pdf.Ln(-1)
}

func renderCountCell(pdf *fpdf.Fpdf, w float64, count int, color [3]int) {
	if count > 0 {
		pdf.SetTextColor(color[0], color[1], color[2])
	} else {
		setColor(pdf, colorBlack)
	}
	pdf.CellFormat(w, 7, itoa(count), "1", 0, "C", true, 0, "")
}

func renderSection(pdf *fpdf.Fpdf, section Section) {
	if len(section.Succeeded) == 0 && len(section.Failed) == 0 && len(section.Skipped) == 0 {
		return
	}

	pdf.Ln(8)
	checkPageBreak(pdf, 30)

	setColor(pdf, colorMedBlue)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 10, section.Name, "", 1, "L", false, 0, "")

	setColor(pdf, colorBlack)
	pdf.SetFont("Helvetica", "", 9)
	counts := fmt.Sprintf("%d succeeded, %d failed, %d skipped",
		len(section.Succeeded), len(section.Failed), len(section.Skipped))
	pdf.CellFormat(0, 6, counts, "", 1, "L", false, 0, "")
	pdf.Ln(2)

	if len(section.Succeeded) > 0 {
		renderSubsection(pdf, "Succeeded", section.Succeeded, false)
	}
	if len(section.Failed) > 0 {
		renderSubsection(pdf, "Failed", section.Failed, true)
	}
	if len(section.Skipped) > 0 {
		renderSubsection(pdf, "Skipped", section.Skipped, false)
	}
}

func renderSubsection(pdf *fpdf.Fpdf, label string, items []EntityItem, isFailed bool) {
	checkPageBreak(pdf, 20)

	pdf.SetFont("Helvetica", "B", 9)
	setColor(pdf, colorBlack)
	pdf.CellFormat(0, 6, label, "", 1, "L", false, 0, "")

	headers, widths := subsectionColumns(isFailed, items)
	renderTableHeader(pdf, headers, widths)

	pdf.SetFont("Helvetica", "", 8)
	for i, item := range items {
		checkPageBreak(pdf, 7)
		renderItemRow(pdf, item, widths, isFailed, i%2 == 0)
	}
}

func subsectionColumns(isFailed bool, items []EntityItem) ([]string, []float64) {
	if isFailed {
		return []string{"Name", "Organization", "Error"},
			[]float64{50, 40, 106}
	}
	// Check if any item has scan history info in Detail
	hasScanHistory := false
	for _, item := range items {
		if strings.Contains(item.Detail, "|scan:") {
			hasScanHistory = true
			break
		}
	}
	if hasScanHistory {
		return []string{"Name", "Organization", "Cloud Key", "Scan History"},
			[]float64{45, 35, 75, 41}
	}
	return []string{"Name", "Organization", "Detail"},
		[]float64{50, 40, 106}
}

func renderTableHeader(pdf *fpdf.Fpdf, headers []string, widths []float64) {
	setFillColor(pdf, colorMedBlue)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 6, h, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
}

func renderItemRow(pdf *fpdf.Fpdf, item EntityItem, widths []float64, isFailed bool, even bool) {
	if isFailed {
		setFillColor(pdf, colorFailedBg)
	} else if even {
		setFillColor(pdf, colorLightGray)
	} else {
		setFillColor(pdf, colorWhite)
	}
	setColor(pdf, colorBlack)

	if isFailed {
		renderFailedRow(pdf, item, widths)
	} else {
		renderSuccessRow(pdf, item, widths)
	}
}

func renderFailedRow(pdf *fpdf.Fpdf, item EntityItem, widths []float64) {
	pdf.CellFormat(widths[0], 6, truncate(item.Name, 30), "1", 0, "L", true, 0, "")
	pdf.CellFormat(widths[1], 6, truncate(item.Organization, 24), "1", 0, "L", true, 0, "")
	pdf.CellFormat(widths[2], 6, truncate(item.ErrorMessage, 65), "1", 0, "L", true, 0, "")
	pdf.Ln(-1)
}

func renderSuccessRow(pdf *fpdf.Fpdf, item EntityItem, widths []float64) {
	detail, scanStatus := parseScanHistory(item.Detail)

	pdf.CellFormat(widths[0], 6, truncate(item.Name, 28), "1", 0, "L", true, 0, "")
	pdf.CellFormat(widths[1], 6, truncate(item.Organization, 22), "1", 0, "L", true, 0, "")
	pdf.CellFormat(widths[2], 6, truncate(detail, 46), "1", 0, "L", true, 0, "")

	if len(widths) > 3 {
		label := scanStatusLabel(scanStatus)
		pdf.CellFormat(widths[3], 6, label, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
}

func parseScanHistory(detail string) (string, string) {
	idx := strings.Index(detail, "|scan:")
	if idx < 0 {
		return detail, ""
	}
	return detail[:idx], detail[idx+6:]
}

func scanStatusLabel(status string) string {
	switch status {
	case "success":
		return "Yes"
	case "failed":
		return "Failed"
	case "skipped":
		return "No"
	default:
		return ""
	}
}

func checkPageBreak(pdf *fpdf.Fpdf, h float64) {
	_, pageH := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	if pdf.GetY()+h > pageH-bottom-10 {
		pdf.AddPage()
	}
}

func setColor(pdf *fpdf.Fpdf, c [3]int) {
	pdf.SetTextColor(c[0], c[1], c[2])
}

func setFillColor(pdf *fpdf.Fpdf, c [3]int) {
	pdf.SetFillColor(c[0], c[1], c[2])
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
