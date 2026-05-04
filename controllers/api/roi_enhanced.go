package api

import (
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
)

// ── Enhanced ROI Endpoints ──────────────────────────────────────
// Industry benchmarks, Monte Carlo analysis, historical reports,
// quarterly trends, and server-side PDF rendering.

// ROIBenchmarks handles GET (list) and POST (create/update) for /api/roi/benchmarks.
func (as *Server) ROIBenchmarks(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		benchmarks := models.GetBenchmarks(user.OrgId)
		JSONResponse(w, benchmarks, http.StatusOK)

	case http.MethodPost:
		var b models.ROIBenchmark
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		b.OrgId = user.OrgId
		if err := models.SaveBenchmark(&b); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving benchmark"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, b, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ROIBenchmarkItem handles DELETE for /api/roi/benchmarks/{id}.
func (as *Server) ROIBenchmarkItem(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := models.DeleteBenchmark(id, user.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Benchmark deleted"}, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ROIBenchmarkSeed handles POST /api/roi/benchmarks/seed — seeds default industry benchmarks.
func (as *Server) ROIBenchmarkSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	if err := models.SeedBenchmarks(user.OrgId); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error seeding benchmarks"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Industry benchmarks seeded"}, http.StatusOK)
}

// ROIBenchmarkCompare handles GET /api/roi/benchmarks/compare — returns org vs industry comparison.
func (as *Server) ROIBenchmarkCompare(w http.ResponseWriter, r *http.Request) {
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

	rpt, err := models.GenerateROIReport(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating comparison"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, rpt.Benchmarks, http.StatusOK)
}

// ROIMonteCarlo handles GET /api/roi/monte-carlo — runs simulation and returns confidence intervals.
func (as *Server) ROIMonteCarlo(w http.ResponseWriter, r *http.Request) {
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

	rpt, err := models.GenerateROIReport(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating ROI for simulation"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, rpt.MonteCarlo, http.StatusOK)
}

// ROIHistory handles GET /api/roi/history — returns all stored historical reports.
func (as *Server) ROIHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	records, err := models.GetROIReportHistory(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error loading history"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, records, http.StatusOK)
}

// ROIQuarterlyTrend handles GET /api/roi/trend — returns quarterly trend data for charting.
func (as *Server) ROIQuarterlyTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	points := models.GetROIQuarterlyTrend(user.OrgId)
	JSONResponse(w, points, http.StatusOK)
}

// ROIHistoryItem handles GET/DELETE for /api/roi/history/{id}.
func (as *Server) ROIHistoryItem(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		record, err := models.GetROIReportByID(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Report not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, record, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteROIReportRecord(id, user.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Historical report deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ROIGenerateAndSave handles POST /api/roi/generate-and-save — generates + persists.
func (as *Server) ROIGenerateAndSave(w http.ResponseWriter, r *http.Request) {
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

	rpt, err := models.GenerateROIReport(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating ROI report"}, http.StatusInternalServerError)
		return
	}

	// Persist the report to history
	if err := models.SaveROIReportRecord(rpt, user.Id); err != nil {
		log.Errorf("roi: failed to save report history: %v", err)
		// Non-fatal — still return the report
	}

	JSONResponse(w, rpt, http.StatusOK)
}

// ROIExportEnhancedPDF handles GET /api/roi/export-pdf — server-side PDF with
// benchmarks, Monte Carlo intervals, and quarterly trend chart rendered using gofpdf.
func (as *Server) ROIExportEnhancedPDF(w http.ResponseWriter, r *http.Request) {
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

	rpt, err := models.GenerateROIReport(user.OrgId, start, end)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error generating ROI report"}, http.StatusInternalServerError)
		return
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)

	// ── Page 1: Executive Summary ──
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 22)
	pdf.CellFormat(0, 16, "ROI Report", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 8, "Security Awareness Programme", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, rpt.PeriodLabel, "", 1, "C", false, 0, "")
	pdf.Ln(10)

	m := rpt.Metrics
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 12, fmt.Sprintf("ROI: %.0f%%  |  Cost Avoidance: $%.0f", m.ROIPercentage, m.CostAvoidance), "", 1, "C", false, 0, "")
	pdf.Ln(6)

	// Monte Carlo summary
	if rpt.MonteCarlo != nil {
		mc := rpt.MonteCarlo
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(0, 10, "Confidence Intervals (90%)", "", 1, "", false, 0, "")
		pdf.SetFont("Arial", "", 11)
		pdf.CellFormat(0, 7, fmt.Sprintf("Incidents Avoided: %.0f - %.0f (median: %.0f)",
			mc.IncidentsAvoided.Lower, mc.IncidentsAvoided.Upper, mc.IncidentsAvoided.Median), "", 1, "", false, 0, "")
		pdf.CellFormat(0, 7, fmt.Sprintf("Cost Avoidance: $%.0f - $%.0f (median: $%.0f)",
			mc.CostAvoidance.Lower, mc.CostAvoidance.Upper, mc.CostAvoidance.Median), "", 1, "", false, 0, "")
		pdf.CellFormat(0, 7, fmt.Sprintf("ROI: %.0f%% - %.0f%% (median: %.0f%%)",
			mc.ROIPercentage.Lower, mc.ROIPercentage.Upper, mc.ROIPercentage.Median), "", 1, "", false, 0, "")
		pdf.Ln(6)
	}

	// Key metrics table
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "Key Metrics", "", 1, "", false, 0, "")
	pdf.SetFillColor(52, 152, 219)
	pdf.SetTextColor(255, 255, 255)
	pdfTable(pdf, []string{"Metric", "Value"}, [][]string{
		{"Programme Cost", fmt.Sprintf("$%.0f", rpt.ProgramCost)},
		{"Cost Avoidance", fmt.Sprintf("$%.0f", m.CostAvoidance)},
		{"ROI (%)", fmt.Sprintf("%.0f%%", m.ROIPercentage)},
		{"Payback Period", fmt.Sprintf("%.1f months", m.PaybackPeriodMonths)},
		{"Incidents Avoided", strconv.Itoa(m.EstIncidentsAvoided)},
		{"Click Rate Reduction", fmt.Sprintf("%.1f pp", m.ClickRateReduction)},
		{"Breach Risk Reduction", fmt.Sprintf("%.1f%%", m.BreachRiskReduction)},
		{"Overall Risk Reduction", fmt.Sprintf("%.1f%%", m.OverallRiskReduction)},
	}, []float64{100, 80})

	// ── Page 2: Industry Benchmarks ──
	if len(rpt.Benchmarks) > 0 {
		pdf.AddPage()
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(0, 12, "Industry Benchmark Comparison", "", 1, "", false, 0, "")
		pdf.Ln(4)

		benchRows := [][]string{}
		for _, b := range rpt.Benchmarks {
			status := "Average"
			if b.Percentile == "top_quartile" {
				status = "Top Quartile"
			} else if b.Percentile == "below_average" {
				status = "Below Avg"
			}
			favorable := "No"
			if b.Favorable {
				favorable = "Yes"
			}
			benchRows = append(benchRows, []string{
				b.MetricLabel,
				fmt.Sprintf("%.1f", b.OrgValue),
				fmt.Sprintf("%.1f", b.IndustryAvg),
				fmt.Sprintf("%+.1f", b.Delta),
				status,
				favorable,
			})
		}

		pdf.SetFillColor(46, 204, 113)
		pdf.SetTextColor(255, 255, 255)
		pdfTable(pdf, []string{"Metric", "Your Value", "Industry Avg", "Delta", "Quartile", "Favorable"}, benchRows, []float64{50, 25, 28, 20, 28, 22})
	}

	// ── Page 3: Findings & Recommendations ──
	pdf.AddPage()
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 12, "Key Findings", "", 1, "", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 11)
	for i, f := range rpt.KeyFindings {
		pdf.MultiCell(0, 7, fmt.Sprintf("%d. %s", i+1, f), "", "", false)
		pdf.Ln(2)
	}
	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 12, "Recommendations", "", 1, "", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 11)
	for i, rec := range rpt.Recommendations {
		pdf.MultiCell(0, 7, fmt.Sprintf("%d. %s", i+1, rec), "", "", false)
		pdf.Ln(2)
	}

	// Footer
	pdf.Ln(12)
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated by Nivoxis Security Platform — %s", time.Now().Format("02 Jan 2006 15:04")), "", 1, "C", false, 0, "")

	filename := fmt.Sprintf("nivoxis-roi-report-enhanced-%s.pdf", time.Now().Format(roiDateFmt))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := pdf.Output(w); err != nil {
		log.Error(err)
	}
}
