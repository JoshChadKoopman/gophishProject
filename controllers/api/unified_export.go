package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// ── Unified Export API ──────────────────────────────────────────
// Standardizes all exports behind a single pattern:
//   GET /api/export/{type}?format=pdf|xlsx|csv&start=...&end=...
//
// Supported types: campaigns, training, phishing_tickets, email_security,
//   network_events, risk_scores, executive_summary, compliance, hygiene
//
// Consistent column naming, branding, and date handling across all exports.

const (
	exportDateFmt     = "2006-01-02"
	exportDefaultDays = 30
)

// UnifiedExport handles GET /api/export/{type}?format=pdf|xlsx|csv&start=...&end=...
func (as *Server) UnifiedExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)

	// Parse parameters
	reportType := r.URL.Query().Get("type")
	if reportType == "" {
		// Try mux var
		reportType = r.URL.Query().Get("report_type")
	}
	if !models.ValidReportTypes[reportType] {
		JSONResponse(w, models.Response{Success: false,
			Message: "Invalid report type. Valid types: executive_summary, campaigns, training, phishing_tickets, email_security, network_events, roi, compliance, hygiene, risk_scores"},
			http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf"
	}
	if !models.ValidExportFormats[format] {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid format. Use: pdf, xlsx, or csv"}, http.StatusBadRequest)
		return
	}

	start, _ := time.Parse(exportDateFmt, r.URL.Query().Get("start"))
	end, _ := time.Parse(exportDateFmt, r.URL.Query().Get("end"))
	if start.IsZero() {
		start = time.Now().UTC().AddDate(0, 0, -exportDefaultDays)
	}
	if end.IsZero() {
		end = time.Now().UTC()
	}

	dateStr := time.Now().Format(exportDateFmt)
	baseName := fmt.Sprintf("nivoxis-%s-%s", reportType, dateStr)

	// Collect data
	sections := unifiedCollectData(reportType, scope, start, end)

	switch format {
	case "pdf":
		buf := unifiedExportPDF(reportType, sections, start, end)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.pdf\"", baseName))
		w.Write(buf.Bytes())

	case "xlsx":
		buf := unifiedExportXLSX(reportType, sections)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.xlsx\"", baseName))
		w.Write(buf.Bytes())

	case "csv":
		buf := unifiedExportCSV(sections)
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.csv\"", baseName))
		w.Write(buf.Bytes())
	}
}

// ── Data Collection ──

type exportSection struct {
	Title   string
	Headers []string
	Rows    [][]string
}

func unifiedCollectData(reportType string, scope models.OrgScope, start, end time.Time) []exportSection {
	days := int(end.Sub(start).Hours()/24) + 1
	if days < 1 {
		days = exportDefaultDays
	}

	switch reportType {
	case models.ReportTypeExecutiveSummary:
		return unifiedExecutiveSummary(scope, days, start, end)
	case models.ReportTypeCampaigns:
		return unifiedCampaigns(scope, start, end)
	case models.ReportTypeTraining:
		return unifiedTraining(scope, days, start, end)
	case models.ReportTypePhishingTickets:
		return unifiedTickets(scope, days, start, end)
	case models.ReportTypeEmailSecurity:
		return unifiedEmailSecurity(scope, days, start, end)
	case models.ReportTypeNetworkEvents:
		return unifiedNetworkEvents(scope, start, end)
	case models.ReportTypeRiskScores:
		return unifiedRiskScores(scope)
	case models.ReportTypeCompliance:
		return unifiedCompliance(scope)
	case models.ReportTypeHygiene:
		return unifiedHygiene(scope)
	default:
		return unifiedExecutiveSummary(scope, days, start, end)
	}
}

func unifiedExecutiveSummary(scope models.OrgScope, days int, start, end time.Time) []exportSection {
	emailsSent := models.GetDailyMetricSum(scope, "emails_sent", days)
	linksClicked := models.GetDailyMetricSum(scope, "links_clicked", days)
	reported := models.GetDailyMetricSum(scope, "emails_reported", days)
	campaignsLaunched := models.GetDailyMetricSum(scope, "campaigns_launched", days)
	clickRate := models.GetDailyMetricAvg(scope, "click_rate", days)
	reportRate := models.GetDailyMetricAvg(scope, "report_rate", days)
	trainingCompleted := models.GetDailyMetricSum(scope, "training_completed", days)
	completionRate := models.GetDailyMetricAvg(scope, "training_completion_rate", 1)
	overdue := models.GetDailyMetricSum(scope, "training_overdue", 1)
	ticketsOpened := models.GetDailyMetricSum(scope, "tickets_opened", days)
	ticketsResolved := models.GetDailyMetricSum(scope, "tickets_resolved", days)
	avgRisk := models.GetDailyMetricAvg(scope, "avg_risk_score", 1)
	highRiskUsers := models.GetDailyMetricSum(scope, "high_risk_user_count", 1)

	return []exportSection{
		{
			Title:   "Executive Summary",
			Headers: []string{"Metric", "Value"},
			Rows: [][]string{
				{"Period", fmt.Sprintf("%s to %s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006"))},
				{"Campaigns Launched", fmtFloat(campaignsLaunched)},
				{"Emails Sent", fmtFloat(emailsSent)},
				{"Links Clicked", fmtFloat(linksClicked)},
				{"Click Rate (%)", fmt.Sprintf("%.1f", clickRate)},
				{"Emails Reported", fmtFloat(reported)},
				{"Report Rate (%)", fmt.Sprintf("%.1f", reportRate)},
				{"Training Completed", fmtFloat(trainingCompleted)},
				{"Training Completion Rate (%)", fmt.Sprintf("%.1f", completionRate)},
				{"Training Overdue", fmtFloat(overdue)},
				{"Tickets Opened", fmtFloat(ticketsOpened)},
				{"Tickets Resolved", fmtFloat(ticketsResolved)},
				{"Avg Risk Score", fmt.Sprintf("%.1f", avgRisk)},
				{"High Risk Users", fmtFloat(highRiskUsers)},
			},
		},
	}
}

func unifiedCampaigns(scope models.OrgScope, start, end time.Time) []exportSection {
	var campaigns []models.Campaign
	q := models.GetDB().Table("campaigns").Where("created_date BETWEEN ? AND ?", start, end)
	if !scope.IsSuperAdmin {
		q = q.Where("org_id = ?", scope.OrgId)
	}
	q.Order("created_date DESC").Find(&campaigns)

	rows := make([][]string, 0, len(campaigns))
	for _, c := range campaigns {
		completed := "-"
		if !c.CompletedDate.IsZero() {
			completed = c.CompletedDate.Format(exportDateFmt)
		}
		rows = append(rows, []string{
			strconv.FormatInt(c.Id, 10),
			c.Name,
			c.Status,
			c.CreatedDate.Format(exportDateFmt),
			c.LaunchDate.Format(exportDateFmt),
			completed,
			c.CampaignType,
		})
	}

	return []exportSection{
		{
			Title:   "Campaigns",
			Headers: []string{"ID", "Name", "Status", "Created", "Launch Date", "Completed", "Type"},
			Rows:    rows,
		},
	}
}

func unifiedTraining(scope models.OrgScope, days int, start, end time.Time) []exportSection {
	summaryRows := [][]string{
		{"Period", fmt.Sprintf("%s to %s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006"))},
		{"Assigned", fmtFloat(models.GetDailyMetricSum(scope, "training_assigned", days))},
		{"Completed", fmtFloat(models.GetDailyMetricSum(scope, "training_completed", days))},
		{"Overdue", fmtFloat(models.GetDailyMetricSum(scope, "training_overdue", 1))},
		{"Completion Rate (%)", fmt.Sprintf("%.1f", models.GetDailyMetricAvg(scope, "training_completion_rate", 1))},
		{"Avg Quiz Score", fmt.Sprintf("%.1f", models.GetDailyMetricAvg(scope, "avg_quiz_score", days))},
		{"Certificates Issued", fmtFloat(models.GetDailyMetricSum(scope, "certificates_issued", days))},
	}

	// Sparkline data as a daily breakdown
	sparkline := models.GetDailyMetricSparkline(scope, "training_completed", days)
	dailyRows := make([][]string, 0, len(sparkline))
	for _, p := range sparkline {
		dailyRows = append(dailyRows, []string{p.Date, fmtFloat(p.Value)})
	}

	return []exportSection{
		{
			Title:   "Training Summary",
			Headers: []string{"Metric", "Value"},
			Rows:    summaryRows,
		},
		{
			Title:   "Daily Completions",
			Headers: []string{"Date", "Completions"},
			Rows:    dailyRows,
		},
	}
}

func unifiedTickets(scope models.OrgScope, days int, start, end time.Time) []exportSection {
	summaryRows := [][]string{
		{"Period", fmt.Sprintf("%s to %s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006"))},
		{"Tickets Opened", fmtFloat(models.GetDailyMetricSum(scope, "tickets_opened", days))},
		{"Tickets Resolved", fmtFloat(models.GetDailyMetricSum(scope, "tickets_resolved", days))},
		{"Incidents Created", fmtFloat(models.GetDailyMetricSum(scope, "incidents_created", days))},
		{"Incidents Resolved", fmtFloat(models.GetDailyMetricSum(scope, "incidents_resolved", days))},
	}

	sparkline := models.GetDailyMetricSparkline(scope, "tickets_opened", days)
	dailyRows := make([][]string, 0, len(sparkline))
	for _, p := range sparkline {
		dailyRows = append(dailyRows, []string{p.Date, fmtFloat(p.Value)})
	}

	return []exportSection{
		{
			Title:   "Phishing Tickets Summary",
			Headers: []string{"Metric", "Value"},
			Rows:    summaryRows,
		},
		{
			Title:   "Daily Tickets Opened",
			Headers: []string{"Date", "Count"},
			Rows:    dailyRows,
		},
	}
}

func unifiedEmailSecurity(scope models.OrgScope, days int, start, end time.Time) []exportSection {
	rows := [][]string{
		{"Period", fmt.Sprintf("%s to %s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006"))},
		{"Emails Sent", fmtFloat(models.GetDailyMetricSum(scope, "emails_sent", days))},
		{"Emails Opened", fmtFloat(models.GetDailyMetricSum(scope, "emails_opened", days))},
		{"Links Clicked", fmtFloat(models.GetDailyMetricSum(scope, "links_clicked", days))},
		{"Data Submitted", fmtFloat(models.GetDailyMetricSum(scope, "data_submitted", days))},
		{"Emails Reported", fmtFloat(models.GetDailyMetricSum(scope, "emails_reported", days))},
		{"Click Rate (%)", fmt.Sprintf("%.1f", models.GetDailyMetricAvg(scope, "click_rate", days))},
		{"Report Rate (%)", fmt.Sprintf("%.1f", models.GetDailyMetricAvg(scope, "report_rate", days))},
		{"Tickets Opened", fmtFloat(models.GetDailyMetricSum(scope, "tickets_opened", days))},
		{"Tickets Resolved", fmtFloat(models.GetDailyMetricSum(scope, "tickets_resolved", days))},
		{"Network Events", fmtFloat(models.GetDailyMetricSum(scope, "network_events_ingested", days))},
	}

	return []exportSection{
		{
			Title:   "Email Security Overview",
			Headers: []string{"Metric", "Value"},
			Rows:    rows,
		},
	}
}

func unifiedNetworkEvents(scope models.OrgScope, start, end time.Time) []exportSection {
	type evtRow struct {
		Id          int64
		EventType   string
		Severity    string
		SourceIP    string
		Description string
		EventDate   time.Time
	}
	var events []evtRow
	q := models.GetDB().Table("network_events").
		Select("id, event_type, severity, source_ip, description, event_date").
		Where("event_date BETWEEN ? AND ?", start, end)
	if !scope.IsSuperAdmin {
		q = q.Where("org_id = ?", scope.OrgId)
	}
	q.Order("event_date DESC").Limit(1000).Find(&events)

	rows := make([][]string, 0, len(events))
	for _, e := range events {
		rows = append(rows, []string{
			strconv.FormatInt(e.Id, 10),
			e.EventType,
			e.Severity,
			e.SourceIP,
			truncateStr(e.Description, 80),
			e.EventDate.Format(exportDateFmt + " 15:04"),
		})
	}

	return []exportSection{
		{
			Title:   "Network Events",
			Headers: []string{"ID", "Type", "Severity", "Source IP", "Description", "Date"},
			Rows:    rows,
		},
	}
}

func unifiedRiskScores(scope models.OrgScope) []exportSection {
	type riskRow struct {
		UserId   int64
		Email    string
		Score    float64
		Category string
	}
	var users []riskRow
	q := models.GetDB().Table("user_risk_scores urs").
		Select("urs.user_id, u.email, urs.overall_score as score, urs.category").
		Joins("JOIN users u ON u.id = urs.user_id")
	if !scope.IsSuperAdmin {
		q = q.Where("urs.org_id = ?", scope.OrgId)
	}
	q.Order("urs.overall_score DESC").Find(&users)

	rows := make([][]string, 0, len(users))
	for _, u := range users {
		rows = append(rows, []string{
			strconv.FormatInt(u.UserId, 10),
			u.Email,
			fmt.Sprintf("%.1f", u.Score),
			u.Category,
		})
	}

	summaryRows := [][]string{
		{"Avg Risk Score", fmt.Sprintf("%.1f", models.GetDailyMetricAvg(scope, "avg_risk_score", 1))},
		{"High Risk Users", fmtFloat(models.GetDailyMetricSum(scope, "high_risk_user_count", 1))},
		{"Total Users", fmtFloat(models.GetDailyMetricSum(scope, "total_users", 1))},
	}

	return []exportSection{
		{
			Title:   "Risk Score Summary",
			Headers: []string{"Metric", "Value"},
			Rows:    summaryRows,
		},
		{
			Title:   "User Risk Scores",
			Headers: []string{"User ID", "Email", "Risk Score", "Category"},
			Rows:    rows,
		},
	}
}

func unifiedCompliance(scope models.OrgScope) []exportSection {
	rows := [][]string{
		{"Overall Compliance Score (%)", fmt.Sprintf("%.1f", models.GetDailyMetricAvg(scope, "compliance_score", 1))},
	}

	// Framework-level breakdown
	type fwRow struct {
		Name  string
		Score float64
	}
	var frameworks []fwRow
	q := models.GetDB().Table("org_compliance_frameworks ocf").
		Select("cf.name, COALESCE(AVG(CASE WHEN cca.status='met' THEN 100 WHEN cca.status='partial' THEN 50 ELSE 0 END),0) as score").
		Joins("JOIN compliance_frameworks cf ON cf.id = ocf.framework_id").
		Joins("LEFT JOIN compliance_control_assessments cca ON cca.framework_id = cf.id AND cca.org_id = ocf.org_id").
		Where("ocf.enabled = ?", true)
	if !scope.IsSuperAdmin {
		q = q.Where("ocf.org_id = ?", scope.OrgId)
	}
	q.Group("cf.name").Find(&frameworks)

	fwRows := make([][]string, 0, len(frameworks))
	for _, f := range frameworks {
		fwRows = append(fwRows, []string{f.Name, fmt.Sprintf("%.1f%%", f.Score)})
	}

	sections := []exportSection{
		{
			Title:   "Compliance Summary",
			Headers: []string{"Metric", "Value"},
			Rows:    rows,
		},
	}
	if len(fwRows) > 0 {
		sections = append(sections, exportSection{
			Title:   "Framework Scores",
			Headers: []string{"Framework", "Score (%)"},
			Rows:    fwRows,
		})
	}
	return sections
}

func unifiedHygiene(scope models.OrgScope) []exportSection {
	rows := [][]string{
		{"Avg Hygiene Score (%)", fmt.Sprintf("%.1f", models.GetDailyMetricAvg(scope, "avg_hygiene_score", 1))},
		{"Devices Compliant", fmtFloat(models.GetDailyMetricSum(scope, "devices_compliant", 1))},
		{"Devices Total", fmtFloat(models.GetDailyMetricSum(scope, "devices_total", 1))},
	}

	return []exportSection{
		{
			Title:   "Cyber Hygiene Summary",
			Headers: []string{"Metric", "Value"},
			Rows:    rows,
		},
	}
}

// ── Rendering ──

func unifiedExportPDF(reportType string, sections []exportSection, start, end time.Time) *bytes.Buffer {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Branded header bar
	pdf.SetFillColor(26, 115, 232) // Nivoxis blue
	pdf.Rect(0, 0, 210, 30, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 20)
	pdf.SetY(6)
	pdf.CellFormat(0, 12, unifiedReportTitle(reportType), "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 7, fmt.Sprintf("%s — %s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006")), "", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(10)

	for _, section := range sections {
		pdf.SetFont("Arial", "B", 13)
		pdf.CellFormat(0, 10, section.Title, "", 1, "", false, 0, "")
		pdf.Ln(2)

		if len(section.Headers) > 0 && len(section.Rows) > 0 {
			colW := 190.0 / float64(len(section.Headers))
			colWidths := make([]float64, len(section.Headers))
			for i := range colWidths {
				colWidths[i] = colW
			}
			unifiedPDFTable(pdf, section.Headers, section.Rows, colWidths)
		}
		pdf.Ln(6)

		// Add page if running out of space
		if pdf.GetY() > 260 {
			pdf.AddPage()
		}
	}

	// Footer
	pdf.Ln(8)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(0, 5, fmt.Sprintf("Generated by Nivoxis on %s", time.Now().UTC().Format("2006-01-02 15:04 UTC")), "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		log.Error(err)
	}
	return &buf
}

func unifiedPDFTable(pdf *gofpdf.Fpdf, headers []string, rows [][]string, colWidths []float64) {
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 8)
	for rowIdx, row := range rows {
		if rowIdx%2 == 0 {
			pdf.SetFillColor(245, 248, 252)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		for i, cell := range row {
			if i < len(colWidths) {
				pdf.CellFormat(colWidths[i], 6, truncateStr(cell, 40), "1", 0, "", true, 0, "")
			}
		}
		pdf.Ln(-1)
	}
}

func unifiedExportXLSX(reportType string, sections []exportSection) *bytes.Buffer {
	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1A73E8"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	firstSheet := true
	for _, section := range sections {
		sheetName := section.Title
		if len(sheetName) > 31 {
			sheetName = sheetName[:31]
		}
		if firstSheet {
			f.SetSheetName("Sheet1", sheetName)
			firstSheet = false
		} else {
			f.NewSheet(sheetName)
		}

		for i, h := range section.Headers {
			cell := unifiedCellRef(i+1, 1)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		for i, row := range section.Rows {
			for j, val := range row {
				f.SetCellValue(sheetName, unifiedCellRef(j+1, i+2), val)
			}
		}

		for _, c := range []string{"A", "B", "C", "D", "E", "F", "G"} {
			f.SetColWidth(sheetName, c, c, 22)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Error(err)
	}
	return &buf
}

func unifiedExportCSV(sections []exportSection) *bytes.Buffer {
	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)

	for _, section := range sections {
		cw.Write([]string{section.Title})
		if len(section.Headers) > 0 {
			cw.Write(section.Headers)
		}
		for _, row := range section.Rows {
			cw.Write(row)
		}
		cw.Write([]string{})
	}

	cw.Flush()
	return &buf
}

// ── Helpers ──

func unifiedCellRef(col, row int) string {
	colStr := ""
	for col > 0 {
		col--
		colStr = string(rune('A'+col%26)) + colStr
		col /= 26
	}
	return colStr + strconv.Itoa(row)
}

func unifiedReportTitle(rt string) string {
	titles := map[string]string{
		models.ReportTypeExecutiveSummary: "Executive Summary",
		models.ReportTypeCampaigns:        "Campaign Report",
		models.ReportTypeTraining:         "Training Report",
		models.ReportTypePhishingTickets:  "Phishing Tickets Report",
		models.ReportTypeEmailSecurity:    "Email Security Report",
		models.ReportTypeNetworkEvents:    "Network Events Report",
		models.ReportTypeROI:              "ROI Report",
		models.ReportTypeCompliance:       "Compliance Report",
		models.ReportTypeHygiene:          "Cyber Hygiene Report",
		models.ReportTypeRiskScores:       "Risk Scores Report",
	}
	if t, ok := titles[rt]; ok {
		return t
	}
	return "Report"
}

func fmtFloat(v float64) string {
	return strconv.Itoa(int(math.Round(v)))
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
