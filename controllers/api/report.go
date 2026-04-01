package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/jung-kurt/gofpdf"
)

// ReportOverview handles GET /api/reports/overview
func (as *Server) ReportOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	overview, err := models.GetReportOverview(getOrgScope(r))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, overview, http.StatusOK)
}

// ReportTrend handles GET /api/reports/trend?days=30
func (as *Server) ReportTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	trend, err := models.GetReportTrend(getOrgScope(r), days)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, trend, http.StatusOK)
}

// ReportRiskScores handles GET /api/reports/risk-scores
func (as *Server) ReportRiskScores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scores, err := models.GetRiskScores(getOrgScope(r))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, scores, http.StatusOK)
}

// ReportTrainingSummary handles GET /api/reports/training-summary
func (as *Server) ReportTrainingSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	summary, err := models.GetTrainingSummaryReport(getOrgScope(r))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// ReportGroupComparison handles GET /api/reports/group-comparison
func (as *Server) ReportGroupComparison(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	comparisons, err := models.GetGroupComparison(getOrgScope(r))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, comparisons, http.StatusOK)
}

// sanitizeCSVField prefixes values that start with formula-injection characters.
func sanitizeCSVField(s string) string {
	if len(s) > 0 {
		switch s[0] {
		case '=', '+', '-', '@':
			return "'" + s
		}
	}
	return s
}

// ReportExport handles GET /api/reports/export?type=X&format=csv
func (as *Server) ReportExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	reportType := r.URL.Query().Get("type")
	format := r.URL.Query().Get("format")

	if format == "pdf" {
		as.reportExportPDF(w, r, scope, reportType)
		return
	}
	if format != "csv" {
		JSONResponse(w, models.Response{Success: false, Message: "Unsupported format. Use 'csv' or 'pdf'."}, http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("nivoxis-report-%s-%s.csv", reportType, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	switch reportType {
	case "overview":
		overview, err := models.GetReportOverview(scope)
		if err != nil {
			log.Error(err)
			http.Error(w, "Error generating report", http.StatusInternalServerError)
			return
		}
		writer.Write([]string{"Metric", "Value"})
		writer.Write([]string{"Total Campaigns", strconv.FormatInt(overview.TotalCampaigns, 10)})
		writer.Write([]string{"Active Campaigns", strconv.FormatInt(overview.ActiveCampaigns, 10)})
		writer.Write([]string{"Total Recipients", strconv.FormatInt(overview.TotalRecipients, 10)})
		writer.Write([]string{"Emails Sent", strconv.FormatInt(overview.Stats.EmailsSent, 10)})
		writer.Write([]string{"Emails Opened", strconv.FormatInt(overview.Stats.OpenedEmail, 10)})
		writer.Write([]string{"Links Clicked", strconv.FormatInt(overview.Stats.ClickedLink, 10)})
		writer.Write([]string{"Data Submitted", strconv.FormatInt(overview.Stats.SubmittedData, 10)})
		writer.Write([]string{"Emails Reported", strconv.FormatInt(overview.Stats.EmailReported, 10)})
		writer.Write([]string{"Avg Click Rate (%)", fmt.Sprintf("%.2f", overview.AvgClickRate)})
		writer.Write([]string{"Avg Submit Rate (%)", fmt.Sprintf("%.2f", overview.AvgSubmitRate)})
		writer.Write([]string{"Avg Report Rate (%)", fmt.Sprintf("%.2f", overview.AvgReportRate)})

	case "risk-scores":
		scores, err := models.GetRiskScores(scope)
		if err != nil {
			log.Error(err)
			http.Error(w, "Error generating report", http.StatusInternalServerError)
			return
		}
		writer.Write([]string{"Email", "First Name", "Last Name", "Total Emails", "Clicked", "Submitted", "Reported", "Risk Score"})
		for _, s := range scores {
			writer.Write([]string{
				sanitizeCSVField(s.Email),
				sanitizeCSVField(s.FirstName),
				sanitizeCSVField(s.LastName),
				strconv.FormatInt(s.Total, 10),
				strconv.FormatInt(s.Clicked, 10),
				strconv.FormatInt(s.Submitted, 10),
				strconv.FormatInt(s.Reported, 10),
				fmt.Sprintf("%.2f", s.RiskScore),
			})
		}

	case "training":
		summary, err := models.GetTrainingSummaryReport(scope)
		if err != nil {
			log.Error(err)
			http.Error(w, "Error generating report", http.StatusInternalServerError)
			return
		}
		writer.Write([]string{"Metric", "Value"})
		writer.Write([]string{"Total Courses", strconv.FormatInt(summary.TotalCourses, 10)})
		writer.Write([]string{"Total Assignments", strconv.FormatInt(summary.TotalAssignments, 10)})
		writer.Write([]string{"Completed", strconv.FormatInt(summary.CompletedCount, 10)})
		writer.Write([]string{"In Progress", strconv.FormatInt(summary.InProgressCount, 10)})
		writer.Write([]string{"Not Started", strconv.FormatInt(summary.NotStartedCount, 10)})
		writer.Write([]string{"Overdue", strconv.FormatInt(summary.OverdueCount, 10)})
		writer.Write([]string{"Completion Rate (%)", fmt.Sprintf("%.2f", summary.CompletionRate)})
		writer.Write([]string{"Certificates Issued", strconv.FormatInt(summary.CertificatesIssued, 10)})
		writer.Write([]string{"Avg Quiz Score (%)", fmt.Sprintf("%.2f", summary.AvgQuizScore)})

	case "groups":
		comparisons, err := models.GetGroupComparison(scope)
		if err != nil {
			log.Error(err)
			http.Error(w, "Error generating report", http.StatusInternalServerError)
			return
		}
		writer.Write([]string{"Group", "Total", "Sent", "Opened", "Clicked", "Submitted", "Reported", "Click Rate (%)", "Submit Rate (%)"})
		for _, gc := range comparisons {
			writer.Write([]string{
				sanitizeCSVField(gc.GroupName),
				strconv.FormatInt(gc.Stats.Total, 10),
				strconv.FormatInt(gc.Stats.EmailsSent, 10),
				strconv.FormatInt(gc.Stats.OpenedEmail, 10),
				strconv.FormatInt(gc.Stats.ClickedLink, 10),
				strconv.FormatInt(gc.Stats.SubmittedData, 10),
				strconv.FormatInt(gc.Stats.EmailReported, 10),
				fmt.Sprintf("%.2f", gc.ClickRate),
				fmt.Sprintf("%.2f", gc.SubmitRate),
			})
		}

	default:
		http.Error(w, "Unknown report type. Use: overview, risk-scores, training, groups", http.StatusBadRequest)
		return
	}
}

// pdfTable is a helper that writes a table with header and rows to a gofpdf document.
func pdfTable(pdf *gofpdf.Fpdf, headers []string, rows [][]string, colWidths []float64) {
	pdf.SetFont("Arial", "B", 10)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Arial", "", 9)
	for _, row := range rows {
		for i, cell := range row {
			pdf.CellFormat(colWidths[i], 7, cell, "1", 0, "", false, 0, "")
		}
		pdf.Ln(-1)
	}
}

// reportExportPDF generates a PDF report for the given report type.
func (as *Server) reportExportPDF(w http.ResponseWriter, r *http.Request, scope models.OrgScope, reportType string) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Title page header
	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(0, 15, "Nivoxis CyberAwareness", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 8, "Report: "+reportType, "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, "Generated: "+time.Now().Format("January 2, 2006 15:04"), "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Fill color for header rows
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)

	switch reportType {
	case "overview":
		as.pdfOverview(pdf, scope)
	case "risk-scores":
		as.pdfRiskScores(pdf, scope)
	case "training":
		as.pdfTraining(pdf, scope)
	case "groups":
		as.pdfGroups(pdf, scope)
	default:
		http.Error(w, "Unknown report type", http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("nivoxis-report-%s-%s.pdf", reportType, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := pdf.Output(w); err != nil {
		log.Error(err)
	}
}

func (as *Server) pdfOverview(pdf *gofpdf.Fpdf, scope models.OrgScope) {
	overview, err := models.GetReportOverview(scope)
	if err != nil {
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, 10, "Error loading overview data.", "", 1, "", false, 0, "")
		return
	}
	headers := []string{"Metric", "Value"}
	widths := []float64{100, 80}
	rows := [][]string{
		{"Total Campaigns", strconv.FormatInt(overview.TotalCampaigns, 10)},
		{"Active Campaigns", strconv.FormatInt(overview.ActiveCampaigns, 10)},
		{"Total Recipients", strconv.FormatInt(overview.TotalRecipients, 10)},
		{"Emails Sent", strconv.FormatInt(overview.Stats.EmailsSent, 10)},
		{"Emails Opened", strconv.FormatInt(overview.Stats.OpenedEmail, 10)},
		{"Links Clicked", strconv.FormatInt(overview.Stats.ClickedLink, 10)},
		{"Data Submitted", strconv.FormatInt(overview.Stats.SubmittedData, 10)},
		{"Emails Reported", strconv.FormatInt(overview.Stats.EmailReported, 10)},
		{"Avg Click Rate (%)", fmt.Sprintf("%.2f", overview.AvgClickRate)},
		{"Avg Submit Rate (%)", fmt.Sprintf("%.2f", overview.AvgSubmitRate)},
		{"Avg Report Rate (%)", fmt.Sprintf("%.2f", overview.AvgReportRate)},
	}
	pdfTable(pdf, headers, rows, widths)
}

func (as *Server) pdfRiskScores(pdf *gofpdf.Fpdf, scope models.OrgScope) {
	scores, err := models.GetRiskScores(scope)
	if err != nil {
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, 10, "Error loading risk score data.", "", 1, "", false, 0, "")
		return
	}
	headers := []string{"Email", "Name", "Emails", "Clicked", "Submitted", "Reported", "Risk"}
	widths := []float64{50, 40, 20, 20, 25, 22, 18}
	var rows [][]string
	for _, s := range scores {
		rows = append(rows, []string{
			s.Email,
			s.FirstName + " " + s.LastName,
			strconv.FormatInt(s.Total, 10),
			strconv.FormatInt(s.Clicked, 10),
			strconv.FormatInt(s.Submitted, 10),
			strconv.FormatInt(s.Reported, 10),
			fmt.Sprintf("%.1f", s.RiskScore),
		})
	}
	pdfTable(pdf, headers, rows, widths)
}

func (as *Server) pdfTraining(pdf *gofpdf.Fpdf, scope models.OrgScope) {
	summary, err := models.GetTrainingSummaryReport(scope)
	if err != nil {
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, 10, "Error loading training data.", "", 1, "", false, 0, "")
		return
	}
	headers := []string{"Metric", "Value"}
	widths := []float64{100, 80}
	rows := [][]string{
		{"Total Courses", strconv.FormatInt(summary.TotalCourses, 10)},
		{"Total Assignments", strconv.FormatInt(summary.TotalAssignments, 10)},
		{"Completed", strconv.FormatInt(summary.CompletedCount, 10)},
		{"In Progress", strconv.FormatInt(summary.InProgressCount, 10)},
		{"Not Started", strconv.FormatInt(summary.NotStartedCount, 10)},
		{"Overdue", strconv.FormatInt(summary.OverdueCount, 10)},
		{"Completion Rate (%)", fmt.Sprintf("%.2f", summary.CompletionRate)},
		{"Certificates Issued", strconv.FormatInt(summary.CertificatesIssued, 10)},
		{"Avg Quiz Score (%)", fmt.Sprintf("%.2f", summary.AvgQuizScore)},
	}
	pdfTable(pdf, headers, rows, widths)
}

func (as *Server) pdfGroups(pdf *gofpdf.Fpdf, scope models.OrgScope) {
	comparisons, err := models.GetGroupComparison(scope)
	if err != nil {
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(0, 10, "Error loading group data.", "", 1, "", false, 0, "")
		return
	}
	headers := []string{"Group", "Total", "Sent", "Opened", "Clicked", "Submitted", "Reported", "Click%"}
	widths := []float64{40, 20, 20, 22, 22, 25, 22, 22}
	var rows [][]string
	for _, gc := range comparisons {
		rows = append(rows, []string{
			gc.GroupName,
			strconv.FormatInt(gc.Stats.Total, 10),
			strconv.FormatInt(gc.Stats.EmailsSent, 10),
			strconv.FormatInt(gc.Stats.OpenedEmail, 10),
			strconv.FormatInt(gc.Stats.ClickedLink, 10),
			strconv.FormatInt(gc.Stats.SubmittedData, 10),
			strconv.FormatInt(gc.Stats.EmailReported, 10),
			fmt.Sprintf("%.1f", gc.ClickRate),
		})
	}
	pdfTable(pdf, headers, rows, widths)
}
