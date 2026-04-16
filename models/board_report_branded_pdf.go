package models

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// ── Branded Board Report PDF Template Engine ────────────────────
// Professional PDF with org branding, narrative, ROI section, period
// deltas, and styled tables suitable for board-level presentations.

// BoardPDFBranding holds org-specific branding for PDF generation.
type BoardPDFBranding struct {
	OrgName  string `json:"org_name"`
	LogoURL  string `json:"logo_url,omitempty"`
	PrimaryR int    `json:"primary_r"`
	PrimaryG int    `json:"primary_g"`
	PrimaryB int    `json:"primary_b"`
	AccentR  int    `json:"accent_r"`
	AccentG  int    `json:"accent_g"`
	AccentB  int    `json:"accent_b"`
}

// DefaultBranding returns the Nivoxis default branding.
func DefaultBranding() BoardPDFBranding {
	return BoardPDFBranding{
		OrgName:  "Nivoxis",
		PrimaryR: 26, PrimaryG: 35, PrimaryB: 126, // #1a237e
		AccentR: 52, AccentG: 152, AccentB: 219, // #3498db
	}
}

// GenerateBrandedBoardPDF creates a professional branded PDF from a full payload.
func GenerateBrandedBoardPDF(w io.Writer, payload *FullBoardReportPayload, branding BoardPDFBranding) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)

	pR, pG, pB := branding.PrimaryR, branding.PrimaryG, branding.PrimaryB
	aR, aG, aB := branding.AccentR, branding.AccentG, branding.AccentB

	// ─── Cover Page ───
	pdf.AddPage()

	// Header bar
	pdf.SetFillColor(pR, pG, pB)
	pdf.Rect(0, 0, 210, 50, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 28)
	pdf.SetXY(15, 12)
	pdf.CellFormat(180, 14, branding.OrgName, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.SetXY(15, 28)
	pdf.CellFormat(180, 8, "Security Awareness Board Report", "", 1, "L", false, 0, "")

	// Title area
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(20)
	pdf.SetFont("Arial", "B", 22)
	pdf.CellFormat(0, 14, payload.Title, "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 13)
	if payload.Snapshot != nil {
		pdf.CellFormat(0, 8, payload.Snapshot.PeriodLabel, "", 1, "C", false, 0, "")
	}
	pdf.CellFormat(0, 8, "Generated: "+time.Now().Format("2 January 2006"), "", 1, "C", false, 0, "")

	// Security Posture Score — big number
	if payload.Snapshot != nil {
		pdf.Ln(15)
		pdf.SetFont("Arial", "B", 48)
		scoreColor := scoreToColor(payload.Snapshot.SecurityPostureScore)
		pdf.SetTextColor(scoreColor[0], scoreColor[1], scoreColor[2])
		pdf.CellFormat(0, 25, fmt.Sprintf("%.0f", payload.Snapshot.SecurityPostureScore), "", 1, "C", false, 0, "")
		pdf.SetTextColor(100, 100, 100)
		pdf.SetFont("Arial", "", 14)
		trendArrow := "→ Stable"
		if payload.Snapshot.RiskTrend == "improving" {
			trendArrow = "↑ Improving"
		} else if payload.Snapshot.RiskTrend == "declining" {
			trendArrow = "↓ Declining"
		}
		pdf.CellFormat(0, 8, "Security Posture Score  /100   "+trendArrow, "", 1, "C", false, 0, "")
	}

	// ─── Executive Narrative Page ───
	if payload.Narrative != nil && payload.Narrative.FullNarrative != "" {
		pdf.AddPage()
		brandedSectionHeader(pdf, "Executive Summary", pR, pG, pB)
		pdf.SetFont("Arial", "", 11)
		pdf.SetTextColor(40, 40, 40)
		pdf.Ln(2)
		if payload.Narrative.Paragraph1 != "" {
			pdf.MultiCell(0, 6, payload.Narrative.Paragraph1, "", "", false)
			pdf.Ln(4)
		}
		if payload.Narrative.Paragraph2 != "" {
			pdf.MultiCell(0, 6, payload.Narrative.Paragraph2, "", "", false)
			pdf.Ln(4)
		}
		if payload.Narrative.Paragraph3 != "" {
			pdf.MultiCell(0, 6, payload.Narrative.Paragraph3, "", "", false)
		}
	}

	// ─── Period-Over-Period Deltas ───
	if len(payload.Deltas) > 0 {
		pdf.Ln(8)
		brandedSectionHeader(pdf, "Period-Over-Period Comparison", pR, pG, pB)
		pdf.Ln(2)

		headers := []string{"Metric", "Current", "Prior", "Change", "Direction"}
		rows := make([][]string, 0, len(payload.Deltas))
		for _, d := range payload.Deltas {
			dir := d.Arrow + " " + d.Direction
			if d.Favorable {
				dir += " (favourable)"
			}
			rows = append(rows, []string{
				d.Label,
				fmt.Sprintf("%.1f", d.CurrentValue),
				fmt.Sprintf("%.1f", d.PriorValue),
				d.DisplayDelta,
				dir,
			})
		}
		brandedTable(pdf, headers, rows, []float64{50, 25, 25, 45, 35}, aR, aG, aB)
	}

	// ─── Phishing & Training Data Pages ───
	if payload.Snapshot != nil {
		snap := payload.Snapshot

		pdf.AddPage()
		brandedSectionHeader(pdf, "1. Phishing Simulation Results", pR, pG, pB)
		brandedTable(pdf, []string{"Metric", "Value"}, [][]string{
			{"Total Campaigns", strconv.FormatInt(snap.Phishing.TotalCampaigns, 10)},
			{"Total Recipients", strconv.FormatInt(snap.Phishing.TotalRecipients, 10)},
			{"Avg Click Rate", fmt.Sprintf("%.1f%%", snap.Phishing.AvgClickRate)},
			{"Click Rate Change (vs prior)", fmt.Sprintf("%+.1fpp", snap.Phishing.ClickRateChange)},
			{"Avg Submit Rate", fmt.Sprintf("%.1f%%", snap.Phishing.AvgSubmitRate)},
			{"Avg Report Rate", fmt.Sprintf("%.1f%%", snap.Phishing.AvgReportRate)},
			{"Report Rate Change (vs prior)", fmt.Sprintf("%+.1fpp", snap.Phishing.ReportRateChange)},
		}, []float64{100, 80}, aR, aG, aB)

		pdf.Ln(8)
		brandedSectionHeader(pdf, "2. Training & Awareness", pR, pG, pB)
		brandedTable(pdf, []string{"Metric", "Value"}, [][]string{
			{"Completion Rate", fmt.Sprintf("%.1f%%", snap.Training.CompletionRate)},
			{"Total Courses", strconv.FormatInt(snap.Training.TotalCourses, 10)},
			{"Overdue Assignments", strconv.FormatInt(snap.Training.OverdueCount, 10)},
			{"Avg Quiz Score", fmt.Sprintf("%.1f%%", snap.Training.AvgQuizScore)},
			{"Certificates Issued", strconv.FormatInt(snap.Training.CertificatesIssued, 10)},
		}, []float64{100, 80}, aR, aG, aB)

		pdf.AddPage()
		brandedSectionHeader(pdf, "3. Risk Assessment", pR, pG, pB)
		brandedTable(pdf, []string{"Category", "Count"}, [][]string{
			{"High Risk Users", strconv.Itoa(snap.Risk.HighRiskUsers)},
			{"Medium Risk Users", strconv.Itoa(snap.Risk.MediumRiskUsers)},
			{"Low Risk Users", strconv.Itoa(snap.Risk.LowRiskUsers)},
			{"Average Risk Score", fmt.Sprintf("%.1f", snap.Risk.AvgRiskScore)},
		}, []float64{100, 80}, aR, aG, aB)

		pdf.Ln(8)
		brandedSectionHeader(pdf, "4. Compliance Posture", pR, pG, pB)
		brandedTable(pdf, []string{"Metric", "Value"}, [][]string{
			{"Frameworks", strconv.Itoa(snap.Compliance.FrameworkCount)},
			{"Overall Score", fmt.Sprintf("%.1f%%", snap.Compliance.OverallScore)},
			{"Compliant Controls", strconv.Itoa(snap.Compliance.Compliant)},
			{"Partial Controls", strconv.Itoa(snap.Compliance.Partial)},
			{"Non-Compliant Controls", strconv.Itoa(snap.Compliance.NonCompliant)},
		}, []float64{100, 80}, aR, aG, aB)

		pdf.Ln(8)
		brandedSectionHeader(pdf, "5. Remediation Progress", pR, pG, pB)
		brandedTable(pdf, []string{"Metric", "Value"}, [][]string{
			{"Total Paths", strconv.Itoa(snap.Remediation.TotalPaths)},
			{"Active", strconv.Itoa(snap.Remediation.ActivePaths)},
			{"Completed", strconv.Itoa(snap.Remediation.CompletedPaths)},
			{"Critical", strconv.Itoa(snap.Remediation.CriticalCount)},
			{"Avg Completion", fmt.Sprintf("%.0f%%", snap.Remediation.AvgCompletion)},
		}, []float64{100, 80}, aR, aG, aB)

		pdf.AddPage()
		brandedSectionHeader(pdf, "6. Cyber Hygiene", pR, pG, pB)
		brandedTable(pdf, []string{"Metric", "Value"}, [][]string{
			{"Total Devices", strconv.Itoa(snap.Hygiene.TotalDevices)},
			{"Avg Hygiene Score", fmt.Sprintf("%.0f%%", snap.Hygiene.AvgScore)},
			{"Fully Compliant", strconv.Itoa(snap.Hygiene.FullyCompliant)},
			{"At Risk Devices", strconv.Itoa(snap.Hygiene.AtRiskDevices)},
		}, []float64{100, 80}, aR, aG, aB)
	}

	// ─── ROI Section ───
	if payload.ROISummary != nil {
		pdf.Ln(8)
		brandedSectionHeader(pdf, "7. Return on Investment", pR, pG, pB)
		roi := payload.ROISummary
		brandedTable(pdf, []string{"Metric", "Value"}, [][]string{
			{"Cost Avoidance", fmt.Sprintf("$%.0fK", roi.CostAvoidance/1000)},
			{"ROI Percentage", fmt.Sprintf("%.0f%%", roi.ROIPercentage)},
			{"Incidents Avoided", strconv.Itoa(roi.EstIncidentsAvoided)},
			{"Click Rate Reduction", fmt.Sprintf("%.1fpp", roi.ClickRateReduction)},
			{"Report Rate Improvement", fmt.Sprintf("%.1fpp", roi.ReportRateImprovement)},
			{"Cost Per Employee", fmt.Sprintf("$%.0f", roi.CostPerEmployee)},
			{"Payback Period", fmt.Sprintf("%.1f months", roi.PaybackPeriodMonths)},
		}, []float64{100, 80}, aR, aG, aB)
	}

	// ─── Recommendations Page ───
	if payload.Snapshot != nil && len(payload.Snapshot.Recommendations) > 0 {
		pdf.AddPage()
		brandedSectionHeader(pdf, "Key Recommendations", pR, pG, pB)
		pdf.Ln(4)
		pdf.SetFont("Arial", "", 11)
		pdf.SetTextColor(40, 40, 40)
		for i, rec := range payload.Snapshot.Recommendations {
			pdf.MultiCell(0, 7, fmt.Sprintf("%d.  %s", i+1, rec), "", "", false)
			pdf.Ln(2)
		}
	}

	// ─── Footer ───
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.SetY(-15)
	pdf.CellFormat(0, 5, fmt.Sprintf("Confidential — %s — Generated %s",
		branding.OrgName, time.Now().Format("2006-01-02 15:04")), "", 0, "C", false, 0, "")

	return pdf.Output(w)
}

// ─── PDF Helpers ────────────────────────────────────────────────

func brandedSectionHeader(pdf *gofpdf.Fpdf, title string, r, g, b int) {
	pdf.SetFillColor(r, g, b)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "  "+title, "", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(2)
}

func brandedTable(pdf *gofpdf.Fpdf, headers []string, rows [][]string, widths []float64, hR, hG, hB int) {
	// Header row
	pdf.SetFillColor(hR, hG, hB)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 10)
	for i, h := range headers {
		w := widths[0]
		if i < len(widths) {
			w = widths[i]
		}
		pdf.CellFormat(w, 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Data rows
	pdf.SetTextColor(40, 40, 40)
	pdf.SetFont("Arial", "", 10)
	for rowIdx, row := range rows {
		if rowIdx%2 == 0 {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		for i, cell := range row {
			w := widths[0]
			if i < len(widths) {
				w = widths[i]
			}
			align := "L"
			if i > 0 {
				align = "R"
			}
			pdf.CellFormat(w, 7, cell, "1", 0, align, true, 0, "")
		}
		pdf.Ln(-1)
	}
}

func scoreToColor(score float64) [3]int {
	switch {
	case score >= 80:
		return [3]int{46, 204, 113} // green
	case score >= 60:
		return [3]int{241, 196, 15} // yellow
	case score >= 40:
		return [3]int{230, 126, 34} // orange
	default:
		return [3]int{231, 76, 60} // red
	}
}
