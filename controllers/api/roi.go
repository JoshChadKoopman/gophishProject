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

// Shared constants for ROI exports.
const (
	roiDateFmt   = "2006-01-02"
	roiFmtPct1   = "%.1f%%"
	roiFmtDollar = "$%.0f"
	roiFmtNum    = "%d. %s"
	roiFmtAttach = "attachment; filename=\"%s\""
	roiHdrCType  = "Content-Type"
	roiHdrCDisp  = "Content-Disposition"
)

// ROIConfig handles GET/PUT for /api/roi/config.
func (as *Server) ROIConfig(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		cfg := models.GetROIConfig(user.OrgId)
		JSONResponse(w, cfg, http.StatusOK)
	case http.MethodPut:
		var cfg models.ROIConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		cfg.OrgId = user.OrgId
		if err := models.SaveROIConfig(&cfg); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving ROI config"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ROIGenerate handles POST /api/roi/generate.
func (as *Server) ROIGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	type genReq struct {
		PeriodStart string `json:"period_start"`
		PeriodEnd   string `json:"period_end"`
	}
	var req genReq
	json.NewDecoder(r.Body).Decode(&req)
	start, _ := time.Parse(roiDateFmt, req.PeriodStart)
	end, _ := time.Parse(roiDateFmt, req.PeriodEnd)
	if start.IsZero() {
		start = time.Now().AddDate(-1, 0, 0)
	}
	if end.IsZero() {
		end = time.Now()
	}
	report, err := models.GenerateROIReport(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating ROI report"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, report, http.StatusOK)
}

// ROIExport handles GET /api/roi/export?format=csv|xlsx|pdf
func (as *Server) ROIExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	start, _ := time.Parse(roiDateFmt, r.URL.Query().Get("start"))
	end, _ := time.Parse(roiDateFmt, r.URL.Query().Get("end"))
	if start.IsZero() {
		start = time.Now().AddDate(-1, 0, 0)
	}
	if end.IsZero() {
		end = time.Now()
	}
	report, err := models.GenerateROIReport(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating ROI report"}, http.StatusInternalServerError)
		return
	}
	switch r.URL.Query().Get("format") {
	case "xlsx":
		as.roiExportXLSX(w, report)
	case "csv":
		as.roiExportCSV(w, report)
	default:
		as.roiExportPDF(w, report)
	}
}

func (as *Server) roiExportPDF(w http.ResponseWriter, rpt *models.ROIReport) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 24)
	pdf.CellFormat(0, 20, "ROI Report", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 8, "Security Awareness Programme", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(0, 8, rpt.PeriodLabel, "", 1, "C", false, 0, "")
	pdf.Ln(12)
	m := rpt.Metrics
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 12, fmt.Sprintf("ROI: %.0f%%  |  Cost Avoidance: "+roiFmtDollar, m.ROIPercentage, m.CostAvoidance), "", 1, "C", false, 0, "")
	pdf.Ln(8)
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "Key Metrics", "", 1, "", false, 0, "")
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdfTable(pdf, []string{"Metric", "Value"}, [][]string{
		{"Programme Cost", fmt.Sprintf(roiFmtDollar, rpt.ProgramCost)},
		{"Cost Avoidance", fmt.Sprintf(roiFmtDollar, m.CostAvoidance)},
		{"ROI (%)", fmt.Sprintf("%.0f%%", m.ROIPercentage)},
		{"Payback Period", fmt.Sprintf("%.1f months", m.PaybackPeriodMonths)},
		{"Incidents Avoided", strconv.Itoa(m.EstIncidentsAvoided)},
		{"Click Rate Reduction", fmt.Sprintf(roiFmtPct1, m.ClickRateReduction)},
		{"Breach Risk Reduction", fmt.Sprintf(roiFmtPct1, m.BreachRiskReduction)},
		{"Overall Risk Reduction", fmt.Sprintf(roiFmtPct1, m.OverallRiskReduction)},
	}, []float64{100, 80})
	// Findings
	pdf.AddPage()
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 12, "Key Findings", "", 1, "", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 11)
	for i, f := range rpt.KeyFindings {
		pdf.MultiCell(0, 7, fmt.Sprintf(roiFmtNum, i+1, f), "", "", false)
		pdf.Ln(2)
	}
	pdf.Ln(8)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 12, "Recommendations", "", 1, "", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 11)
	for i, rec := range rpt.Recommendations {
		pdf.MultiCell(0, 7, fmt.Sprintf(roiFmtNum, i+1, rec), "", "", false)
		pdf.Ln(2)
	}
	filename := fmt.Sprintf("nivoxis-roi-report-%s.pdf", time.Now().Format(roiDateFmt))
	w.Header().Set(roiHdrCType, "application/pdf")
	w.Header().Set(roiHdrCDisp, fmt.Sprintf(roiFmtAttach, filename))
	if err := pdf.Output(w); err != nil {
		log.Error(err)
	}
}

func (as *Server) roiExportXLSX(w http.ResponseWriter, rpt *models.ROIReport) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "ROI Summary"
	f.SetSheetName("Sheet1", sheet)
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"3498DB"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellValue(sheet, "A1", "ROI Report — Security Awareness Programme")
	f.SetCellValue(sheet, "A2", rpt.PeriodLabel)
	f.MergeCell(sheet, "A1", "D1")
	f.MergeCell(sheet, "A2", "D2")
	row := 4
	m := rpt.Metrics
	ph := rpt.Phishing
	tr := rpt.Training
	rm := rpt.Remediation
	hy := rpt.Hygiene
	co := rpt.Compliance
	type sd struct {
		t string
		d [][]interface{}
	}
	secs := []sd{
		{"Key Metrics", [][]interface{}{
			{"Programme Cost", rpt.ProgramCost}, {"Cost Avoidance", m.CostAvoidance},
			{"ROI (%)", m.ROIPercentage}, {"Cost Per Employee", m.CostPerEmployee},
			{"Payback (months)", m.PaybackPeriodMonths}, {"Incidents Avoided", m.EstIncidentsAvoided},
			{"Click Rate Reduction (%)", m.ClickRateReduction}, {"Breach Risk Reduction (%)", m.BreachRiskReduction},
			{"Training Hours Saved", m.TrainingHoursSaved}, {"Training Cost Saved", m.TrainingCostSaved},
			{"Overall Risk Reduction (%)", m.OverallRiskReduction},
		}},
		{"Phishing", [][]interface{}{
			{"Total Simulations", ph.TotalSimulations}, {"Previous Click Rate (%)", ph.PreviousClickRate},
			{"Current Click Rate (%)", ph.CurrentClickRate}, {"Click Rate Reduction (pp)", ph.ClickRateReduction},
			{"Report Rate Increase (pp)", ph.ReportRateIncrease},
			{"Incidents Avoided", ph.IncidentsAvoided}, {"Cost Avoided", ph.CostAvoided},
		}},
		{"Training", [][]interface{}{
			{"Total Courses", tr.TotalCourses}, {"Completion Rate (%)", tr.CompletionRate},
			{"Avg Quiz Score (%)", tr.AvgQuizScore}, {"Certificates Issued", tr.CertificatesIssued},
			{"Productivity Saved (hrs)", tr.ProductivitySaved},
		}},
		{"Remediation", [][]interface{}{
			{"Paths Created", rm.PathsCreated}, {"Paths Completed", rm.PathsCompleted},
			{"Completion Rate (%)", rm.CompletionRate}, {"Critical Resolved", rm.CriticalResolved},
			{"Risk Reduction (%)", rm.RiskReduction},
		}},
		{"Cyber Hygiene", [][]interface{}{
			{"Devices Managed", hy.DevicesManaged}, {"Avg Hygiene Score (%)", hy.AvgHygieneScore},
			{"Fully Compliant (%)", hy.FullyCompliantPct}, {"Vulnerability Reduction (%)", hy.VulnerabilityReduction},
		}},
		{"Compliance", [][]interface{}{
			{"Frameworks Covered", co.FrameworksCovered}, {"Overall Score (%)", co.OverallScore},
			{"Audit Readiness (%)", co.AuditReadiness}, {"Penalty Risk Avoided", co.PenaltyRiskAvoided},
		}},
	}
	for _, sec := range secs {
		titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}})
		f.SetCellValue(sheet, cellRef(1, row), sec.t)
		f.SetCellStyle(sheet, cellRef(1, row), cellRef(1, row), titleStyle)
		row++
		f.SetCellValue(sheet, cellRef(1, row), "Metric")
		f.SetCellValue(sheet, cellRef(2, row), "Value")
		f.SetCellStyle(sheet, cellRef(1, row), cellRef(2, row), headerStyle)
		row++
		for _, dd := range sec.d {
			f.SetCellValue(sheet, cellRef(1, row), dd[0])
			f.SetCellValue(sheet, cellRef(2, row), dd[1])
			row++
		}
		row++
	}
	f.SetColWidth(sheet, "A", "A", 30)
	f.SetColWidth(sheet, "B", "B", 25)
	filename := fmt.Sprintf("nivoxis-roi-report-%s.xlsx", time.Now().Format(roiDateFmt))
	w.Header().Set(roiHdrCType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set(roiHdrCDisp, fmt.Sprintf(roiFmtAttach, filename))
	if err := f.Write(w); err != nil {
		log.Error(err)
	}
}

func (as *Server) roiExportCSV(w http.ResponseWriter, rpt *models.ROIReport) {
	filename := fmt.Sprintf("nivoxis-roi-report-%s.csv", time.Now().Format(roiDateFmt))
	w.Header().Set(roiHdrCType, "text/csv")
	w.Header().Set(roiHdrCDisp, fmt.Sprintf(roiFmtAttach, filename))
	cw := csv.NewWriter(w)
	defer cw.Flush()
	cw.Write([]string{"ROI Report — Security Awareness Programme"})
	cw.Write([]string{"Period", rpt.PeriodLabel})
	cw.Write([]string{})
	cw.Write([]string{"Section", "Metric", "Value"})
	m := rpt.Metrics
	ph := rpt.Phishing
	tr := rpt.Training
	rm := rpt.Remediation
	hy := rpt.Hygiene
	co := rpt.Compliance
	csvData := [][]string{
		{"Metrics", "Programme Cost", fmt.Sprintf("%.0f", rpt.ProgramCost)},
		{"Metrics", "Cost Avoidance", fmt.Sprintf("%.0f", m.CostAvoidance)},
		{"Metrics", "ROI (%)", fmt.Sprintf("%.1f", m.ROIPercentage)},
		{"Metrics", "Cost Per Employee", fmt.Sprintf("%.0f", m.CostPerEmployee)},
		{"Metrics", "Payback (months)", fmt.Sprintf("%.1f", m.PaybackPeriodMonths)},
		{"Metrics", "Incidents Avoided", strconv.Itoa(m.EstIncidentsAvoided)},
		{"Metrics", "Click Rate Reduction (%)", fmt.Sprintf("%.1f", m.ClickRateReduction)},
		{"Metrics", "Breach Risk Reduction (%)", fmt.Sprintf("%.1f", m.BreachRiskReduction)},
		{"Metrics", "Overall Risk Reduction (%)", fmt.Sprintf("%.1f", m.OverallRiskReduction)},
		{"Metrics", "Training Hours Saved", fmt.Sprintf("%.0f", m.TrainingHoursSaved)},
		{"Metrics", "Training Cost Saved", fmt.Sprintf("%.0f", m.TrainingCostSaved)},
		{"Phishing", "Total Simulations", strconv.FormatInt(ph.TotalSimulations, 10)},
		{"Phishing", "Previous Click Rate (%)", fmt.Sprintf("%.1f", ph.PreviousClickRate)},
		{"Phishing", "Current Click Rate (%)", fmt.Sprintf("%.1f", ph.CurrentClickRate)},
		{"Phishing", "Click Rate Reduction (pp)", fmt.Sprintf("%.1f", ph.ClickRateReduction)},
		{"Phishing", "Incidents Avoided", strconv.Itoa(ph.IncidentsAvoided)},
		{"Phishing", "Cost Avoided", fmt.Sprintf("%.0f", ph.CostAvoided)},
		{"Training", "Total Courses", strconv.FormatInt(tr.TotalCourses, 10)},
		{"Training", "Completion Rate (%)", fmt.Sprintf("%.1f", tr.CompletionRate)},
		{"Training", "Certificates Issued", strconv.FormatInt(tr.CertificatesIssued, 10)},
		{"Training", "Productivity Saved (hrs)", fmt.Sprintf("%.0f", tr.ProductivitySaved)},
		{"Remediation", "Paths Created", strconv.Itoa(rm.PathsCreated)},
		{"Remediation", "Paths Completed", strconv.Itoa(rm.PathsCompleted)},
		{"Remediation", "Completion Rate (%)", fmt.Sprintf("%.1f", rm.CompletionRate)},
		{"Remediation", "Risk Reduction (%)", fmt.Sprintf("%.1f", rm.RiskReduction)},
		{"Hygiene", "Devices Managed", strconv.Itoa(hy.DevicesManaged)},
		{"Hygiene", "Avg Hygiene Score (%)", fmt.Sprintf("%.1f", hy.AvgHygieneScore)},
		{"Hygiene", "Fully Compliant (%)", fmt.Sprintf("%.1f", hy.FullyCompliantPct)},
		{"Compliance", "Frameworks Covered", strconv.Itoa(co.FrameworksCovered)},
		{"Compliance", "Overall Score (%)", fmt.Sprintf("%.1f", co.OverallScore)},
		{"Compliance", "Penalty Risk Avoided", fmt.Sprintf("%.0f", co.PenaltyRiskAvoided)},
	}
	for _, row := range csvData {
		cw.Write(row)
	}
	cw.Write([]string{})
	cw.Write([]string{"Key Findings"})
	for i, finding := range rpt.KeyFindings {
		cw.Write([]string{strconv.Itoa(i + 1), sanitizeCSVField(finding)})
	}
	cw.Write([]string{})
	cw.Write([]string{"Recommendations"})
	for i, rec := range rpt.Recommendations {
		cw.Write([]string{strconv.Itoa(i + 1), sanitizeCSVField(rec)})
	}
}
