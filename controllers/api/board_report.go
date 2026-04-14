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
	"github.com/gorilla/mux"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// Shared constants for board report exports.
const (
	brDateFmt        = "2006-01-02"
	brFmtPct1        = "%.1f%%"
	brFmtAttach      = "attachment; filename=\"%s\""
	brHdrCType       = "Content-Type"
	brHdrCDisp       = "Content-Disposition"
	brLblTotalCamp   = "Total Campaigns"
	brLblTotalRecip  = "Total Recipients"
	brLblTotalCourse = "Total Courses"
	brLblOverdueAsgn = "Overdue Assignments"
	brLblCertsIssued = "Certificates Issued"
	brLblHighRisk    = "High Risk Users"
	brLblMedRisk     = "Medium Risk Users"
	brLblLowRisk     = "Low Risk Users"
	brLblTotalPaths  = "Total Paths"
	brLblTotalDev    = "Total Devices"
	brLblFullyCompl  = "Fully Compliant"
	brLblAtRiskDev   = "At Risk Devices"
)

// BoardReports handles GET (list) and POST (create) for /api/board-reports/.
func (as *Server) BoardReports(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		reports, err := models.GetBoardReports(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching board reports"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, reports, http.StatusOK)

	case http.MethodPost:
		br := models.BoardReport{}
		if err := json.NewDecoder(r.Body).Decode(&br); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		br.OrgId = user.OrgId
		br.CreatedBy = user.Id
		if err := models.PostBoardReport(&br); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, br, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// BoardReport handles GET, PUT, DELETE for /api/board-reports/{id}.
func (as *Server) BoardReport(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	switch r.Method {
	case http.MethodGet:
		br, err := models.GetBoardReport(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
			return
		}
		// Attach the live snapshot
		snap, err := models.GenerateBoardReportSnapshot(user.OrgId, br.PeriodStart, br.PeriodEnd)
		if err != nil {
			log.Error(err)
		}
		br.Snapshot = snap
		JSONResponse(w, br, http.StatusOK)

	case http.MethodPut:
		br, err := models.GetBoardReport(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
			return
		}
		update := models.BoardReport{}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		br.Title = update.Title
		br.PeriodStart = update.PeriodStart
		br.PeriodEnd = update.PeriodEnd
		br.Status = update.Status
		if err := models.PutBoardReport(&br); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, br, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteBoardReport(id, user.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Report deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// BoardReportGenerate handles POST /api/board-reports/generate
// to produce a live snapshot without persisting a report record.
func (as *Server) BoardReportGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	type generateReq struct {
		PeriodStart string `json:"period_start"`
		PeriodEnd   string `json:"period_end"`
	}
	var req generateReq
	json.NewDecoder(r.Body).Decode(&req)

	start, _ := time.Parse("2006-01-02", req.PeriodStart)
	end, _ := time.Parse("2006-01-02", req.PeriodEnd)
	if start.IsZero() {
		start = time.Now().AddDate(0, -3, 0)
	}
	if end.IsZero() {
		end = time.Now()
	}

	snap, err := models.GenerateBoardReportSnapshot(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating report"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, snap, http.StatusOK)
}

// BoardReportEnhanced handles POST /api/board-reports/enhanced
// Generates a full board report with AI narrative, period comparison,
// and ROI integration — designed for CISO / board-level consumption.
func (as *Server) BoardReportEnhanced(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	type enhancedReq struct {
		PeriodStart string `json:"period_start"`
		PeriodEnd   string `json:"period_end"`
	}
	var req enhancedReq
	json.NewDecoder(r.Body).Decode(&req)

	start, _ := time.Parse("2006-01-02", req.PeriodStart)
	end, _ := time.Parse("2006-01-02", req.PeriodEnd)
	if start.IsZero() {
		start = time.Now().AddDate(0, -3, 0)
	}
	if end.IsZero() {
		end = time.Now()
	}

	enhanced, err := models.GenerateEnhancedBoardReport(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating enhanced board report"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, enhanced, http.StatusOK)
}

// BoardReportNarrative handles POST /api/board-reports/{id}/narrative
// Generates an AI narrative for an existing board report.
func (as *Server) BoardReportNarrative(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	br, err := models.GetBoardReport(id, user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}

	enhanced, err := models.GenerateEnhancedBoardReport(user.OrgId, br.PeriodStart, br.PeriodEnd)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating narrative"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, enhanced.Narrative, http.StatusOK)
}

// BoardReportExportPDF handles GET /api/board-reports/{id}/export?format=pdf
func (as *Server) BoardReportExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	br, err := models.GetBoardReport(id, user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}

	snap, err := models.GenerateBoardReportSnapshot(user.OrgId, br.PeriodStart, br.PeriodEnd)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating snapshot"}, http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "xlsx":
		as.boardReportExportXLSX(w, br, snap)
	case "csv":
		as.boardReportExportCSV(w, br, snap)
	default:
		as.boardReportExportPDF(w, br, snap)
	}
}

// ─── PDF Export ───

func (as *Server) boardReportExportPDF(w http.ResponseWriter, br models.BoardReport, snap *models.BoardReportSnapshot) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Title page
	pdf.SetFont("Arial", "B", 24)
	pdf.CellFormat(0, 20, "Board Report", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 14)
	pdf.CellFormat(0, 10, br.Title, "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(0, 8, snap.PeriodLabel, "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, "Generated: "+time.Now().Format("January 2, 2006 15:04"), "", 1, "C", false, 0, "")
	pdf.Ln(12)

	// Security Posture Score
	pdf.SetFont("Arial", "B", 16)
	trendEmoji := "→"
	if snap.RiskTrend == "improving" {
		trendEmoji = "↑"
	} else if snap.RiskTrend == "declining" {
		trendEmoji = "↓"
	}
	pdf.CellFormat(0, 12, fmt.Sprintf("Security Posture Score: %.0f/100  %s %s",
		snap.SecurityPostureScore, trendEmoji, snap.RiskTrend), "", 1, "C", false, 0, "")
	pdf.Ln(8)

	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)

	// Phishing section
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 10, "1. Phishing Simulation Results", "", 1, "", false, 0, "")
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdfTable(pdf, []string{"Metric", "Value"}, [][]string{
		{"Total Campaigns", strconv.FormatInt(snap.Phishing.TotalCampaigns, 10)},
		{"Total Recipients", strconv.FormatInt(snap.Phishing.TotalRecipients, 10)},
		{"Avg Click Rate", fmt.Sprintf("%.1f%%", snap.Phishing.AvgClickRate)},
		{"Avg Submit Rate", fmt.Sprintf("%.1f%%", snap.Phishing.AvgSubmitRate)},
		{"Avg Report Rate", fmt.Sprintf("%.1f%%", snap.Phishing.AvgReportRate)},
	}, []float64{100, 80})
	pdf.Ln(6)

	// Training section
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "2. Training & Awareness", "", 1, "", false, 0, "")
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdfTable(pdf, []string{"Metric", "Value"}, [][]string{
		{"Completion Rate", fmt.Sprintf("%.1f%%", snap.Training.CompletionRate)},
		{"Total Courses", strconv.FormatInt(snap.Training.TotalCourses, 10)},
		{"Overdue Assignments", strconv.FormatInt(snap.Training.OverdueCount, 10)},
		{"Avg Quiz Score", fmt.Sprintf("%.1f%%", snap.Training.AvgQuizScore)},
		{"Certificates Issued", strconv.FormatInt(snap.Training.CertificatesIssued, 10)},
	}, []float64{100, 80})
	pdf.Ln(6)

	// Risk section
	pdf.AddPage()
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "3. Risk Assessment", "", 1, "", false, 0, "")
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdfTable(pdf, []string{"Category", "Count"}, [][]string{
		{"High Risk Users", strconv.Itoa(snap.Risk.HighRiskUsers)},
		{"Medium Risk Users", strconv.Itoa(snap.Risk.MediumRiskUsers)},
		{"Low Risk Users", strconv.Itoa(snap.Risk.LowRiskUsers)},
		{"Average Risk Score", fmt.Sprintf("%.1f", snap.Risk.AvgRiskScore)},
	}, []float64{100, 80})
	pdf.Ln(6)

	// Compliance section
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "4. Compliance Posture", "", 1, "", false, 0, "")
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdfTable(pdf, []string{"Metric", "Value"}, [][]string{
		{"Frameworks", strconv.Itoa(snap.Compliance.FrameworkCount)},
		{"Overall Score", fmt.Sprintf("%.1f%%", snap.Compliance.OverallScore)},
		{"Compliant Controls", strconv.Itoa(snap.Compliance.Compliant)},
		{"Partial Controls", strconv.Itoa(snap.Compliance.Partial)},
		{"Non-Compliant Controls", strconv.Itoa(snap.Compliance.NonCompliant)},
	}, []float64{100, 80})
	pdf.Ln(6)

	// Remediation section
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "5. Remediation Progress", "", 1, "", false, 0, "")
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdfTable(pdf, []string{"Metric", "Value"}, [][]string{
		{"Total Paths", strconv.Itoa(snap.Remediation.TotalPaths)},
		{"Active", strconv.Itoa(snap.Remediation.ActivePaths)},
		{"Completed", strconv.Itoa(snap.Remediation.CompletedPaths)},
		{"Critical", strconv.Itoa(snap.Remediation.CriticalCount)},
		{"Avg Completion", fmt.Sprintf("%.0f%%", snap.Remediation.AvgCompletion)},
	}, []float64{100, 80})
	pdf.Ln(6)

	// Hygiene section
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "6. Cyber Hygiene", "", 1, "", false, 0, "")
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdfTable(pdf, []string{"Metric", "Value"}, [][]string{
		{"Total Devices", strconv.Itoa(snap.Hygiene.TotalDevices)},
		{"Avg Hygiene Score", fmt.Sprintf("%.0f%%", snap.Hygiene.AvgScore)},
		{"Fully Compliant", strconv.Itoa(snap.Hygiene.FullyCompliant)},
		{"At Risk Devices", strconv.Itoa(snap.Hygiene.AtRiskDevices)},
	}, []float64{100, 80})

	// Recommendations page
	pdf.AddPage()
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 12, "Key Recommendations", "", 1, "", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 11)
	for i, rec := range snap.Recommendations {
		pdf.MultiCell(0, 7, fmt.Sprintf("%d. %s", i+1, rec), "", "", false)
		pdf.Ln(2)
	}

	filename := fmt.Sprintf("nivoxis-board-report-%s.pdf", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := pdf.Output(w); err != nil {
		log.Error(err)
	}
}

// ─── XLSX Export ───

func (as *Server) boardReportExportXLSX(w http.ResponseWriter, br models.BoardReport, snap *models.BoardReportSnapshot) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Executive Summary"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"3498DB"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// Title
	f.SetCellValue(sheet, "A1", "Board Report: "+br.Title)
	f.SetCellValue(sheet, "A2", snap.PeriodLabel)
	f.SetCellValue(sheet, "A3", fmt.Sprintf("Security Posture Score: %.0f/100 (%s)", snap.SecurityPostureScore, snap.RiskTrend))
	f.MergeCell(sheet, "A1", "D1")
	f.MergeCell(sheet, "A2", "D2")
	f.MergeCell(sheet, "A3", "D3")

	row := 5
	sections := []struct {
		Title string
		Data  [][]interface{}
	}{
		{"Phishing Simulations", [][]interface{}{
			{"Total Campaigns", snap.Phishing.TotalCampaigns},
			{"Total Recipients", snap.Phishing.TotalRecipients},
			{"Avg Click Rate (%)", snap.Phishing.AvgClickRate},
			{"Avg Submit Rate (%)", snap.Phishing.AvgSubmitRate},
			{"Avg Report Rate (%)", snap.Phishing.AvgReportRate},
		}},
		{"Training & Awareness", [][]interface{}{
			{"Completion Rate (%)", snap.Training.CompletionRate},
			{"Total Courses", snap.Training.TotalCourses},
			{"Overdue Assignments", snap.Training.OverdueCount},
			{"Avg Quiz Score (%)", snap.Training.AvgQuizScore},
			{"Certificates Issued", snap.Training.CertificatesIssued},
		}},
		{"Risk Assessment", [][]interface{}{
			{"High Risk Users", snap.Risk.HighRiskUsers},
			{"Medium Risk Users", snap.Risk.MediumRiskUsers},
			{"Low Risk Users", snap.Risk.LowRiskUsers},
			{"Avg Risk Score", snap.Risk.AvgRiskScore},
		}},
		{"Compliance", [][]interface{}{
			{"Frameworks", snap.Compliance.FrameworkCount},
			{"Overall Score (%)", snap.Compliance.OverallScore},
			{"Compliant Controls", snap.Compliance.Compliant},
			{"Partial Controls", snap.Compliance.Partial},
			{"Non-Compliant Controls", snap.Compliance.NonCompliant},
		}},
		{"Remediation", [][]interface{}{
			{"Total Paths", snap.Remediation.TotalPaths},
			{"Active", snap.Remediation.ActivePaths},
			{"Completed", snap.Remediation.CompletedPaths},
			{"Critical", snap.Remediation.CriticalCount},
			{"Avg Completion (%)", snap.Remediation.AvgCompletion},
		}},
		{"Cyber Hygiene", [][]interface{}{
			{"Total Devices", snap.Hygiene.TotalDevices},
			{"Avg Hygiene Score (%)", snap.Hygiene.AvgScore},
			{"Fully Compliant", snap.Hygiene.FullyCompliant},
			{"At Risk Devices", snap.Hygiene.AtRiskDevices},
		}},
	}

	for _, sec := range sections {
		f.SetCellValue(sheet, cellRef(1, row), sec.Title)
		titleStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 12},
		})
		f.SetCellStyle(sheet, cellRef(1, row), cellRef(1, row), titleStyle)
		row++

		// Headers
		f.SetCellValue(sheet, cellRef(1, row), "Metric")
		f.SetCellValue(sheet, cellRef(2, row), "Value")
		f.SetCellStyle(sheet, cellRef(1, row), cellRef(2, row), headerStyle)
		row++

		for _, d := range sec.Data {
			f.SetCellValue(sheet, cellRef(1, row), d[0])
			f.SetCellValue(sheet, cellRef(2, row), d[1])
			row++
		}
		row++ // blank row between sections
	}

	// Recommendations sheet
	recSheet := "Recommendations"
	f.NewSheet(recSheet)
	f.SetCellValue(recSheet, "A1", "Key Recommendations")
	recStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}})
	f.SetCellStyle(recSheet, "A1", "A1", recStyle)
	for i, rec := range snap.Recommendations {
		f.SetCellValue(recSheet, cellRef(1, i+2), fmt.Sprintf("%d. %s", i+1, rec))
	}
	f.SetColWidth(recSheet, "A", "A", 80)
	f.SetColWidth(sheet, "A", "A", 30)
	f.SetColWidth(sheet, "B", "B", 20)

	filename := fmt.Sprintf("nivoxis-board-report-%s.xlsx", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := f.Write(w); err != nil {
		log.Error(err)
	}
}

// ─── CSV Export ───

func (as *Server) boardReportExportCSV(w http.ResponseWriter, br models.BoardReport, snap *models.BoardReportSnapshot) {
	filename := fmt.Sprintf("nivoxis-board-report-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"Board Report: " + br.Title})
	writer.Write([]string{"Period", snap.PeriodLabel})
	writer.Write([]string{"Security Posture Score", fmt.Sprintf("%.0f", snap.SecurityPostureScore)})
	writer.Write([]string{"Risk Trend", snap.RiskTrend})
	writer.Write([]string{})

	writer.Write([]string{"Section", "Metric", "Value"})

	csvRows := [][]string{
		{"Phishing", "Total Campaigns", strconv.FormatInt(snap.Phishing.TotalCampaigns, 10)},
		{"Phishing", "Total Recipients", strconv.FormatInt(snap.Phishing.TotalRecipients, 10)},
		{"Phishing", "Avg Click Rate (%)", fmt.Sprintf("%.2f", snap.Phishing.AvgClickRate)},
		{"Phishing", "Avg Submit Rate (%)", fmt.Sprintf("%.2f", snap.Phishing.AvgSubmitRate)},
		{"Phishing", "Avg Report Rate (%)", fmt.Sprintf("%.2f", snap.Phishing.AvgReportRate)},
		{"Training", "Completion Rate (%)", fmt.Sprintf("%.2f", snap.Training.CompletionRate)},
		{"Training", "Total Courses", strconv.FormatInt(snap.Training.TotalCourses, 10)},
		{"Training", "Overdue Assignments", strconv.FormatInt(snap.Training.OverdueCount, 10)},
		{"Training", "Avg Quiz Score (%)", fmt.Sprintf("%.2f", snap.Training.AvgQuizScore)},
		{"Training", "Certificates Issued", strconv.FormatInt(snap.Training.CertificatesIssued, 10)},
		{"Risk", "High Risk Users", strconv.Itoa(snap.Risk.HighRiskUsers)},
		{"Risk", "Medium Risk Users", strconv.Itoa(snap.Risk.MediumRiskUsers)},
		{"Risk", "Low Risk Users", strconv.Itoa(snap.Risk.LowRiskUsers)},
		{"Risk", "Avg Risk Score", fmt.Sprintf("%.2f", snap.Risk.AvgRiskScore)},
		{"Compliance", "Frameworks", strconv.Itoa(snap.Compliance.FrameworkCount)},
		{"Compliance", "Overall Score (%)", fmt.Sprintf("%.2f", snap.Compliance.OverallScore)},
		{"Compliance", "Compliant", strconv.Itoa(snap.Compliance.Compliant)},
		{"Compliance", "Partial", strconv.Itoa(snap.Compliance.Partial)},
		{"Compliance", "Non-Compliant", strconv.Itoa(snap.Compliance.NonCompliant)},
		{"Remediation", "Total Paths", strconv.Itoa(snap.Remediation.TotalPaths)},
		{"Remediation", "Active", strconv.Itoa(snap.Remediation.ActivePaths)},
		{"Remediation", "Completed", strconv.Itoa(snap.Remediation.CompletedPaths)},
		{"Remediation", "Critical", strconv.Itoa(snap.Remediation.CriticalCount)},
		{"Remediation", "Avg Completion (%)", fmt.Sprintf("%.0f", snap.Remediation.AvgCompletion)},
		{"Hygiene", "Total Devices", strconv.Itoa(snap.Hygiene.TotalDevices)},
		{"Hygiene", "Avg Score (%)", fmt.Sprintf("%.0f", snap.Hygiene.AvgScore)},
		{"Hygiene", "Fully Compliant", strconv.Itoa(snap.Hygiene.FullyCompliant)},
		{"Hygiene", "At Risk Devices", strconv.Itoa(snap.Hygiene.AtRiskDevices)},
	}
	for _, row := range csvRows {
		writer.Write(row)
	}

	writer.Write([]string{})
	writer.Write([]string{"Recommendations"})
	for i, rec := range snap.Recommendations {
		writer.Write([]string{strconv.Itoa(i + 1), sanitizeCSVField(rec)})
	}
}
