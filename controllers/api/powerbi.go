package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// OData-style wrapper for Power BI DirectQuery compatibility.
type odataResponse struct {
	Context string      `json:"@odata.context"`
	Value   interface{} `json:"value"`
	Count   int         `json:"@odata.count,omitempty"`
}

// PowerBIFeed handles GET /api/powerbi/{dataset} — OData-compatible JSON feed.
// Supported datasets: campaigns, results, risk-scores, training, groups, brs, compliance.
func (as *Server) PowerBIFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	scope := getOrgScope(r)
	dataset := r.URL.Query().Get("dataset")
	if dataset == "" {
		// List available datasets
		datasets := []map[string]string{
			{"name": "campaigns", "description": "Campaign overview metrics"},
			{"name": "results", "description": "Per-recipient campaign results"},
			{"name": "risk-scores", "description": "User risk scores"},
			{"name": "training", "description": "Training completion summary"},
			{"name": "groups", "description": "Group comparison metrics"},
			{"name": "brs", "description": "Behavioral risk scores"},
			{"name": "trend", "description": "Daily phishing trend data"},
			{"name": "compliance", "description": "Compliance framework posture"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(odataResponse{
			Context: "/api/powerbi/$metadata",
			Value:   datasets,
		})
		return
	}

	var data interface{}
	var count int
	var err error

	switch dataset {
	case "campaigns":
		overview, e := models.GetReportOverview(scope)
		if e != nil {
			err = e
			break
		}
		data = []models.ReportOverview{overview}
		count = 1

	case "results":
		data, err = fetchResultsFeed(scope)
		if d, ok := data.([]resultRow); ok {
			count = len(d)
		}

	case "risk-scores":
		scores, e := models.GetRiskScores(scope)
		if e != nil {
			err = e
			break
		}
		data = scores
		count = len(scores)

	case "training":
		summary, e := models.GetTrainingSummaryReport(scope)
		if e != nil {
			err = e
			break
		}
		data = []models.TrainingSummary{summary}
		count = 1

	case "groups":
		comparisons, e := models.GetGroupComparison(scope)
		if e != nil {
			err = e
			break
		}
		data = comparisons
		count = len(comparisons)

	case "brs":
		data, err = fetchBRSFeed(scope)
		if d, ok := data.([]brsRow); ok {
			count = len(d)
		}

	case "trend":
		days := 90
		if d := r.URL.Query().Get("days"); d != "" {
			if parsed, e := strconv.Atoi(d); e == nil && parsed > 0 {
				days = parsed
			}
		}
		points, e := models.GetReportTrend(scope, days)
		if e != nil {
			err = e
			break
		}
		data = points
		count = len(points)

	case "compliance":
		user := ctx.Get(r, "user").(models.User)
		dashboard, e := models.GetComplianceDashboard(user.OrgId)
		if e != nil {
			err = e
			break
		}
		data = dashboard.Frameworks
		count = len(dashboard.Frameworks)

	default:
		JSONResponse(w, models.Response{Success: false, Message: "Unknown dataset: " + dataset}, http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching data"}, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(odataResponse{
		Context: "/api/powerbi/" + dataset,
		Value:   data,
		Count:   count,
	})
}

// resultRow is a flat row for Power BI results feed.
type resultRow struct {
	CampaignName string `json:"campaign_name"`
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Position     string `json:"position"`
	Status       string `json:"status"`
	Reported     bool   `json:"reported"`
	SendDate     string `json:"send_date"`
	ModifiedDate string `json:"modified_date"`
}

func fetchResultsFeed(scope models.OrgScope) ([]resultRow, error) {
	db := models.GetDB()
	var rows []resultRow
	var err error
	if scope.IsSuperAdmin {
		err = db.Raw(`
			SELECT c.name as campaign_name, r.email, r.first_name, r.last_name,
				r.position, r.status, r.reported, r.send_date, r.modified_date
			FROM results r
			JOIN campaigns c ON r.campaign_id = c.id
			ORDER BY r.modified_date DESC
			LIMIT 10000
		`).Scan(&rows).Error
	} else {
		err = db.Raw(`
			SELECT c.name as campaign_name, r.email, r.first_name, r.last_name,
				r.position, r.status, r.reported, r.send_date, r.modified_date
			FROM results r
			JOIN campaigns c ON r.campaign_id = c.id
			WHERE c.org_id = ?
			ORDER BY r.modified_date DESC
			LIMIT 10000
		`, scope.OrgId).Scan(&rows).Error
	}
	return rows, err
}

// brsRow is a flat row for BRS feed.
type brsRow struct {
	Email            string  `json:"email"`
	Department       string  `json:"department"`
	SimulationScore  float64 `json:"simulation_score"`
	AcademyScore     float64 `json:"academy_score"`
	QuizScore        float64 `json:"quiz_score"`
	TrendScore       float64 `json:"trend_score"`
	ConsistencyScore float64 `json:"consistency_score"`
	CompositeScore   float64 `json:"composite_score"`
	Percentile       float64 `json:"percentile"`
	LastCalculated   string  `json:"last_calculated"`
}

func fetchBRSFeed(scope models.OrgScope) ([]brsRow, error) {
	db := models.GetDB()
	var rows []brsRow
	var err error
	if scope.IsSuperAdmin {
		err = db.Raw(`
			SELECT u.email, u.department, urs.simulation_score, urs.academy_score,
				urs.quiz_score, urs.trend_score, urs.consistency_score,
				urs.composite_score, urs.percentile, urs.last_calculated
			FROM user_risk_scores urs
			JOIN users u ON urs.user_id = u.id
			ORDER BY urs.composite_score DESC
		`).Scan(&rows).Error
	} else {
		err = db.Raw(`
			SELECT u.email, u.department, urs.simulation_score, urs.academy_score,
				urs.quiz_score, urs.trend_score, urs.consistency_score,
				urs.composite_score, urs.percentile, urs.last_calculated
			FROM user_risk_scores urs
			JOIN users u ON urs.user_id = u.id
			WHERE urs.org_id = ?
			ORDER BY urs.composite_score DESC
		`, scope.OrgId).Scan(&rows).Error
	}
	return rows, err
}

// ReportExportExcel handles GET /api/reports/export with format=xlsx.
// Extends the existing ReportExport to support Excel format.
func (as *Server) ReportExportExcel(w http.ResponseWriter, r *http.Request, scope models.OrgScope, reportType string) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Report"
	f.SetSheetName("Sheet1", sheet)

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"3498DB"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "2980B9", Style: 2},
		},
	})

	switch reportType {
	case "overview":
		as.xlsxOverview(f, sheet, headerStyle, scope)
	case "risk-scores":
		as.xlsxRiskScores(f, sheet, headerStyle, scope)
	case "training":
		as.xlsxTraining(f, sheet, headerStyle, scope)
	case "groups":
		as.xlsxGroups(f, sheet, headerStyle, scope)
	case "compliance":
		as.xlsxCompliance(f, sheet, headerStyle, scope)
	case "brs":
		as.xlsxBRS(f, sheet, headerStyle, scope)
	case "results":
		as.xlsxResults(f, sheet, headerStyle, scope)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Unknown report type"}, http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("nivoxis-report-%s-%s.xlsx", reportType, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := f.Write(w); err != nil {
		log.Error(err)
	}
}

func (as *Server) xlsxOverview(f *excelize.File, sheet string, style int, scope models.OrgScope) {
	overview, err := models.GetReportOverview(scope)
	if err != nil {
		return
	}
	headers := []string{"Metric", "Value"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
	}
	rows := [][]interface{}{
		{"Total Campaigns", overview.TotalCampaigns},
		{"Active Campaigns", overview.ActiveCampaigns},
		{"Total Recipients", overview.TotalRecipients},
		{"Emails Sent", overview.Stats.EmailsSent},
		{"Emails Opened", overview.Stats.OpenedEmail},
		{"Links Clicked", overview.Stats.ClickedLink},
		{"Data Submitted", overview.Stats.SubmittedData},
		{"Emails Reported", overview.Stats.EmailReported},
		{"Avg Click Rate (%)", overview.AvgClickRate},
		{"Avg Submit Rate (%)", overview.AvgSubmitRate},
		{"Avg Report Rate (%)", overview.AvgReportRate},
	}
	writeXlsxRows(f, sheet, 2, rows)
	f.SetColWidth(sheet, "A", "A", 25)
	f.SetColWidth(sheet, "B", "B", 15)
}

func (as *Server) xlsxRiskScores(f *excelize.File, sheet string, style int, scope models.OrgScope) {
	scores, err := models.GetRiskScores(scope)
	if err != nil {
		return
	}
	headers := []string{"Email", "First Name", "Last Name", "Total Emails", "Clicked", "Submitted", "Reported", "Risk Score"}
	writeXlsxHeaders(f, sheet, style, headers)

	for i, s := range scores {
		row := i + 2
		f.SetCellValue(sheet, cellRef(1, row), s.Email)
		f.SetCellValue(sheet, cellRef(2, row), s.FirstName)
		f.SetCellValue(sheet, cellRef(3, row), s.LastName)
		f.SetCellValue(sheet, cellRef(4, row), s.Total)
		f.SetCellValue(sheet, cellRef(5, row), s.Clicked)
		f.SetCellValue(sheet, cellRef(6, row), s.Submitted)
		f.SetCellValue(sheet, cellRef(7, row), s.Reported)
		f.SetCellValue(sheet, cellRef(8, row), s.RiskScore)
	}
	f.SetColWidth(sheet, "A", "A", 30)
}

func (as *Server) xlsxTraining(f *excelize.File, sheet string, style int, scope models.OrgScope) {
	summary, err := models.GetTrainingSummaryReport(scope)
	if err != nil {
		return
	}
	headers := []string{"Metric", "Value"}
	writeXlsxHeaders(f, sheet, style, headers)
	rows := [][]interface{}{
		{"Total Courses", summary.TotalCourses},
		{"Total Assignments", summary.TotalAssignments},
		{"Completed", summary.CompletedCount},
		{"In Progress", summary.InProgressCount},
		{"Not Started", summary.NotStartedCount},
		{"Overdue", summary.OverdueCount},
		{"Completion Rate (%)", summary.CompletionRate},
		{"Certificates Issued", summary.CertificatesIssued},
		{"Avg Quiz Score (%)", summary.AvgQuizScore},
	}
	writeXlsxRows(f, sheet, 2, rows)
	f.SetColWidth(sheet, "A", "A", 25)
}

func (as *Server) xlsxGroups(f *excelize.File, sheet string, style int, scope models.OrgScope) {
	comparisons, err := models.GetGroupComparison(scope)
	if err != nil {
		return
	}
	headers := []string{"Group", "Total", "Sent", "Opened", "Clicked", "Submitted", "Reported", "Click Rate (%)", "Submit Rate (%)"}
	writeXlsxHeaders(f, sheet, style, headers)

	for i, gc := range comparisons {
		row := i + 2
		f.SetCellValue(sheet, cellRef(1, row), gc.GroupName)
		f.SetCellValue(sheet, cellRef(2, row), gc.Stats.Total)
		f.SetCellValue(sheet, cellRef(3, row), gc.Stats.EmailsSent)
		f.SetCellValue(sheet, cellRef(4, row), gc.Stats.OpenedEmail)
		f.SetCellValue(sheet, cellRef(5, row), gc.Stats.ClickedLink)
		f.SetCellValue(sheet, cellRef(6, row), gc.Stats.SubmittedData)
		f.SetCellValue(sheet, cellRef(7, row), gc.Stats.EmailReported)
		f.SetCellValue(sheet, cellRef(8, row), gc.ClickRate)
		f.SetCellValue(sheet, cellRef(9, row), gc.SubmitRate)
	}
	f.SetColWidth(sheet, "A", "A", 25)
}

func (as *Server) xlsxCompliance(f *excelize.File, sheet string, style int, scope models.OrgScope) {
	user := models.User{OrgId: scope.OrgId}
	_ = user
	dashboard, err := models.GetComplianceDashboard(scope.OrgId)
	if err != nil {
		return
	}

	// Summary sheet
	headers := []string{"Framework", "Version", "Region", "Total Controls", "Compliant", "Partial", "Non-Compliant", "Not Assessed", "Score (%)"}
	writeXlsxHeaders(f, sheet, style, headers)

	for i, fs := range dashboard.Frameworks {
		row := i + 2
		f.SetCellValue(sheet, cellRef(1, row), fs.Framework.Name)
		f.SetCellValue(sheet, cellRef(2, row), fs.Framework.Version)
		f.SetCellValue(sheet, cellRef(3, row), fs.Framework.Region)
		f.SetCellValue(sheet, cellRef(4, row), fs.TotalControls)
		f.SetCellValue(sheet, cellRef(5, row), fs.Compliant)
		f.SetCellValue(sheet, cellRef(6, row), fs.Partial)
		f.SetCellValue(sheet, cellRef(7, row), fs.NonCompliant)
		f.SetCellValue(sheet, cellRef(8, row), fs.NotAssessed)
		f.SetCellValue(sheet, cellRef(9, row), fs.OverallScore)
	}
	f.SetColWidth(sheet, "A", "A", 15)
	f.SetColWidth(sheet, "B", "B", 12)

	// Controls detail sheet
	detailSheet := "Controls"
	f.NewSheet(detailSheet)
	detailHeaders := []string{"Framework", "Control Ref", "Title", "Category", "Status", "Score", "Evidence", "Last Assessed"}
	writeXlsxHeaders(f, detailSheet, style, detailHeaders)

	row := 2
	for _, fs := range dashboard.Frameworks {
		full, _ := models.GetFrameworkSummary(scope.OrgId, fs.Framework.Id, true)
		for _, c := range full.Controls {
			f.SetCellValue(detailSheet, cellRef(1, row), fs.Framework.Name)
			f.SetCellValue(detailSheet, cellRef(2, row), c.ControlRef)
			f.SetCellValue(detailSheet, cellRef(3, row), c.Title)
			f.SetCellValue(detailSheet, cellRef(4, row), c.Category)
			f.SetCellValue(detailSheet, cellRef(5, row), c.Status)
			f.SetCellValue(detailSheet, cellRef(6, row), c.Score)
			f.SetCellValue(detailSheet, cellRef(7, row), c.Evidence)
			f.SetCellValue(detailSheet, cellRef(8, row), c.LastAssessed)
			row++
		}
	}
	f.SetColWidth(detailSheet, "A", "A", 12)
	f.SetColWidth(detailSheet, "B", "B", 18)
	f.SetColWidth(detailSheet, "C", "C", 40)
	f.SetColWidth(detailSheet, "G", "G", 30)
}

func (as *Server) xlsxBRS(f *excelize.File, sheet string, style int, scope models.OrgScope) {
	rows, err := fetchBRSFeed(scope)
	if err != nil {
		return
	}
	headers := []string{"Email", "Department", "Simulation", "Academy", "Quiz", "Trend", "Consistency", "Composite", "Percentile"}
	writeXlsxHeaders(f, sheet, style, headers)

	for i, b := range rows {
		row := i + 2
		f.SetCellValue(sheet, cellRef(1, row), b.Email)
		f.SetCellValue(sheet, cellRef(2, row), b.Department)
		f.SetCellValue(sheet, cellRef(3, row), b.SimulationScore)
		f.SetCellValue(sheet, cellRef(4, row), b.AcademyScore)
		f.SetCellValue(sheet, cellRef(5, row), b.QuizScore)
		f.SetCellValue(sheet, cellRef(6, row), b.TrendScore)
		f.SetCellValue(sheet, cellRef(7, row), b.ConsistencyScore)
		f.SetCellValue(sheet, cellRef(8, row), b.CompositeScore)
		f.SetCellValue(sheet, cellRef(9, row), b.Percentile)
	}
	f.SetColWidth(sheet, "A", "A", 30)
	f.SetColWidth(sheet, "B", "B", 20)
}

func (as *Server) xlsxResults(f *excelize.File, sheet string, style int, scope models.OrgScope) {
	rows, err := fetchResultsFeed(scope)
	if err != nil {
		return
	}
	headers := []string{"Campaign", "Email", "First Name", "Last Name", "Position", "Status", "Reported", "Send Date", "Modified Date"}
	writeXlsxHeaders(f, sheet, style, headers)

	for i, r := range rows {
		row := i + 2
		f.SetCellValue(sheet, cellRef(1, row), r.CampaignName)
		f.SetCellValue(sheet, cellRef(2, row), r.Email)
		f.SetCellValue(sheet, cellRef(3, row), r.FirstName)
		f.SetCellValue(sheet, cellRef(4, row), r.LastName)
		f.SetCellValue(sheet, cellRef(5, row), r.Position)
		f.SetCellValue(sheet, cellRef(6, row), r.Status)
		f.SetCellValue(sheet, cellRef(7, row), r.Reported)
		f.SetCellValue(sheet, cellRef(8, row), r.SendDate)
		f.SetCellValue(sheet, cellRef(9, row), r.ModifiedDate)
	}
	f.SetColWidth(sheet, "A", "A", 25)
	f.SetColWidth(sheet, "B", "B", 30)
}

// cellRef returns a cell reference like "A1", "B3".
func cellRef(col, row int) string {
	ref, _ := excelize.CoordinatesToCellName(col, row)
	return ref
}

func writeXlsxHeaders(f *excelize.File, sheet string, style int, headers []string) {
	for i, h := range headers {
		cell := cellRef(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
	}
}

func writeXlsxRows(f *excelize.File, sheet string, startRow int, rows [][]interface{}) {
	for i, row := range rows {
		for j, val := range row {
			f.SetCellValue(sheet, cellRef(j+1, startRow+i), val)
		}
	}
}

// ComplianceExportCSV handles compliance report export as CSV.
func (as *Server) complianceExportCSV(w http.ResponseWriter, scope models.OrgScope) {
	dashboard, err := models.GetComplianceDashboard(scope.OrgId)
	if err != nil {
		log.Error(err)
		http.Error(w, "Error generating compliance report", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("nivoxis-compliance-report-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"Framework", "Control Ref", "Title", "Category", "Status", "Score", "Evidence", "Last Assessed"})

	for _, fs := range dashboard.Frameworks {
		full, _ := models.GetFrameworkSummary(scope.OrgId, fs.Framework.Id, true)
		for _, c := range full.Controls {
			writer.Write([]string{
				fs.Framework.Name,
				sanitizeCSVField(c.ControlRef),
				sanitizeCSVField(c.Title),
				sanitizeCSVField(c.Category),
				c.Status,
				fmt.Sprintf("%.1f", c.Score),
				sanitizeCSVField(c.Evidence),
				c.LastAssessed,
			})
		}
	}
}

// complianceExportPDF generates a PDF compliance report.
func (as *Server) complianceExportPDF(w http.ResponseWriter, scope models.OrgScope) {
	dashboard, err := models.GetComplianceDashboard(scope.OrgId)
	if err != nil {
		log.Error(err)
		http.Error(w, "Error generating compliance report", http.StatusInternalServerError)
		return
	}

	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(0, 15, "Nivoxis Compliance Report", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Overall Score: %.1f%%", dashboard.OverallScore), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, "Generated: "+time.Now().Format("January 2, 2006 15:04"), "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Framework summary table
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	headers := []string{"Framework", "Version", "Controls", "Compliant", "Partial", "Non-Compl.", "Score"}
	widths := []float64{50, 25, 25, 30, 25, 30, 25}
	pdfTable(pdf, headers, nil, widths)

	pdf.SetTextColor(0, 0, 0)
	for _, fs := range dashboard.Frameworks {
		row := []string{
			fs.Framework.Name,
			fs.Framework.Version,
			strconv.Itoa(fs.TotalControls),
			strconv.Itoa(fs.Compliant),
			strconv.Itoa(fs.Partial),
			strconv.Itoa(fs.NonCompliant),
			fmt.Sprintf("%.1f%%", fs.OverallScore),
		}
		for i, cell := range row {
			pdf.CellFormat(widths[i], 7, cell, "1", 0, "", false, 0, "")
		}
		pdf.Ln(-1)
	}

	// Detailed controls for each framework
	for _, fs := range dashboard.Frameworks {
		pdf.AddPage()
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(0, 12, fs.Framework.Name+" — Control Details", "", 1, "", false, 0, "")
		pdf.Ln(5)

		pdf.SetFillColor(52, 152, 219)
		pdf.SetTextColor(255, 255, 255)
		ctrlHeaders := []string{"Ref", "Title", "Category", "Status", "Score"}
		ctrlWidths := []float64{35, 100, 50, 35, 25}

		full, _ := models.GetFrameworkSummary(scope.OrgId, fs.Framework.Id, true)
		var ctrlRows [][]string
		for _, c := range full.Controls {
			ctrlRows = append(ctrlRows, []string{
				c.ControlRef,
				c.Title,
				c.Category,
				c.Status,
				fmt.Sprintf("%.0f", c.Score),
			})
		}
		pdfTable(pdf, ctrlHeaders, ctrlRows, ctrlWidths)
	}

	filename := fmt.Sprintf("nivoxis-compliance-report-%s.pdf", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := pdf.Output(w); err != nil {
		log.Error(err)
	}
}
