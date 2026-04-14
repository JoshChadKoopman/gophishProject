package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/xuri/excelize/v2"
)

// ─── Cyber Hygiene Export ───

// HygieneExport handles GET /api/hygiene/export?format=csv|xlsx
// Exports the full org-level hygiene device list with check data.
func (as *Server) HygieneExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	devices, err := models.GetOrgDevicesEnriched(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching devices"}, http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "xlsx":
		as.hygieneExportXLSX(w, devices, user.OrgId)
	default:
		as.hygieneExportCSV(w, devices, user.OrgId)
	}
}

func (as *Server) hygieneExportCSV(w http.ResponseWriter, devices []models.HygieneAdminDeviceView, orgId int64) {
	filename := fmt.Sprintf("nivoxis-hygiene-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Summary row
	summary, _ := models.GetOrgHygieneEnrichedSummary(orgId)
	cw.Write([]string{"Cyber Hygiene Export"})
	cw.Write([]string{"Total Devices", strconv.Itoa(summary.TotalDevices)})
	cw.Write([]string{"Avg Score (%)", fmt.Sprintf("%.1f", summary.AvgScore)})
	cw.Write([]string{"Fully Compliant", strconv.Itoa(summary.FullyCompliant)})
	cw.Write([]string{"At Risk", strconv.Itoa(summary.AtRiskDevices)})
	cw.Write([]string{})

	// Device detail
	cw.Write([]string{
		"Device Name", "Type", "OS", "Hygiene Score",
		"User Name", "User Email", "Department",
		"OS Updated", "Antivirus", "Disk Encrypted",
		"Screen Lock", "Password Mgr", "VPN", "MFA",
	})

	for _, d := range devices {
		checkMap := make(map[string]string)
		for _, c := range d.Checks {
			checkMap[c.CheckType] = c.Status
		}
		cw.Write([]string{
			sanitizeCSVField(d.Name),
			d.DeviceType,
			d.OS,
			strconv.Itoa(d.HygieneScore),
			sanitizeCSVField(d.UserName),
			sanitizeCSVField(d.UserEmail),
			sanitizeCSVField(d.Department),
			checkMap[models.HygieneCheckOSUpdated],
			checkMap[models.HygieneCheckAntivirusActive],
			checkMap[models.HygieneCheckDiskEncrypted],
			checkMap[models.HygieneCheckScreenLock],
			checkMap[models.HygieneCheckPasswordManager],
			checkMap[models.HygieneCheckVPNEnabled],
			checkMap[models.HygieneCheckMFAEnabled],
		})
	}
}

func (as *Server) hygieneExportXLSX(w http.ResponseWriter, devices []models.HygieneAdminDeviceView, orgId int64) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Devices"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"27AE60"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	passStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D5F5E3"}},
	})
	failStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FADBD8"}},
	})

	headers := []string{
		"Device Name", "Type", "OS", "Score",
		"User", "Email", "Department",
		"OS Updated", "Antivirus", "Disk Encrypted",
		"Screen Lock", "Password Mgr", "VPN", "MFA",
	}
	for i, h := range headers {
		f.SetCellValue(sheet, cellRef(i+1, 1), h)
		f.SetCellStyle(sheet, cellRef(i+1, 1), cellRef(i+1, 1), headerStyle)
	}

	checkTypes := []string{
		models.HygieneCheckOSUpdated, models.HygieneCheckAntivirusActive,
		models.HygieneCheckDiskEncrypted, models.HygieneCheckScreenLock,
		models.HygieneCheckPasswordManager, models.HygieneCheckVPNEnabled,
		models.HygieneCheckMFAEnabled,
	}

	for i, d := range devices {
		row := i + 2
		f.SetCellValue(sheet, cellRef(1, row), d.Name)
		f.SetCellValue(sheet, cellRef(2, row), d.DeviceType)
		f.SetCellValue(sheet, cellRef(3, row), d.OS)
		f.SetCellValue(sheet, cellRef(4, row), d.HygieneScore)
		f.SetCellValue(sheet, cellRef(5, row), d.UserName)
		f.SetCellValue(sheet, cellRef(6, row), d.UserEmail)
		f.SetCellValue(sheet, cellRef(7, row), d.Department)

		checkMap := make(map[string]string)
		for _, c := range d.Checks {
			checkMap[c.CheckType] = c.Status
		}
		for j, ct := range checkTypes {
			col := 8 + j
			status := checkMap[ct]
			f.SetCellValue(sheet, cellRef(col, row), status)
			if status == models.HygieneStatusPass {
				f.SetCellStyle(sheet, cellRef(col, row), cellRef(col, row), passStyle)
			} else if status == models.HygieneStatusFail {
				f.SetCellStyle(sheet, cellRef(col, row), cellRef(col, row), failStyle)
			}
		}
	}

	// Summary sheet
	sumSheet := "Summary"
	f.NewSheet(sumSheet)
	summary, _ := models.GetOrgHygieneEnrichedSummary(orgId)
	sumData := [][]interface{}{
		{"Total Devices", summary.TotalDevices},
		{"Avg Score (%)", summary.AvgScore},
		{"Fully Compliant", summary.FullyCompliant},
		{"At Risk (<50%)", summary.AtRiskDevices},
		{"Profiles Configured", summary.ProfileCount},
	}
	for i, d := range sumData {
		f.SetCellValue(sumSheet, cellRef(1, i+1), d[0])
		f.SetCellValue(sumSheet, cellRef(2, i+1), d[1])
	}
	f.SetColWidth(sumSheet, "A", "A", 25)
	f.SetColWidth(sheet, "A", "A", 20)
	for _, c := range []string{"B", "C", "D", "E", "F", "G"} {
		f.SetColWidth(sheet, c, c, 15)
	}

	filename := fmt.Sprintf("nivoxis-hygiene-%s.xlsx", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := f.Write(w); err != nil {
		log.Error(err)
	}
}

// ─── Remediation Export ───

// RemediationExport handles GET /api/remediation/export?format=csv|xlsx
func (as *Server) RemediationExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	paths, err := models.GetRemediationPaths(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching remediation paths"}, http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "xlsx":
		as.remediationExportXLSX(w, paths, user.OrgId)
	default:
		as.remediationExportCSV(w, paths, user.OrgId)
	}
}

func (as *Server) remediationExportCSV(w http.ResponseWriter, paths []models.RemediationPath, orgId int64) {
	filename := fmt.Sprintf("nivoxis-remediation-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Summary
	summary, _ := models.GetRemediationSummary(orgId)
	cw.Write([]string{"Remediation Paths Export"})
	cw.Write([]string{"Total Paths", strconv.Itoa(summary.TotalPaths)})
	cw.Write([]string{"Active", strconv.Itoa(summary.ActivePaths)})
	cw.Write([]string{"Completed", strconv.Itoa(summary.CompletedPaths)})
	cw.Write([]string{"Critical Risk", strconv.Itoa(summary.CriticalCount)})
	cw.Write([]string{"Avg Completion (%)", fmt.Sprintf("%.1f", summary.AvgCompletion)})
	cw.Write([]string{})

	// Path detail
	cw.Write([]string{
		"Path ID", "Name", "User", "Email", "Risk Level", "Status",
		"Fail Count", "Total Courses", "Completed", "Due Date", "Created",
	})

	for _, p := range paths {
		cw.Write([]string{
			strconv.FormatInt(p.Id, 10),
			sanitizeCSVField(p.Name),
			sanitizeCSVField(p.UserName),
			sanitizeCSVField(p.UserEmail),
			p.RiskLevel,
			p.Status,
			strconv.Itoa(p.FailCount),
			strconv.Itoa(p.TotalCourses),
			strconv.Itoa(p.CompletedCount),
			p.DueDate.Format("2006-01-02"),
			p.CreatedDate.Format("2006-01-02"),
		})
	}

	// Steps detail
	cw.Write([]string{})
	cw.Write([]string{"Steps Detail"})
	cw.Write([]string{"Path ID", "Step Order", "Course", "Required", "Status", "Completed"})
	for _, p := range paths {
		for _, s := range p.Steps {
			completed := ""
			if !s.CompletedDate.IsZero() {
				completed = s.CompletedDate.Format("2006-01-02")
			}
			cw.Write([]string{
				strconv.FormatInt(p.Id, 10),
				strconv.Itoa(s.SortOrder),
				sanitizeCSVField(s.CourseName),
				strconv.FormatBool(s.Required),
				s.Status,
				completed,
			})
		}
	}
}

func (as *Server) remediationExportXLSX(w http.ResponseWriter, paths []models.RemediationPath, orgId int64) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Remediation Paths"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E74C3C"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	critStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FADBD8"}},
	})
	compStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D5F5E3"}},
	})

	headers := []string{
		"ID", "Name", "User", "Email", "Risk Level", "Status",
		"Fail Count", "Total Courses", "Completed", "Due Date", "Created",
	}
	for i, h := range headers {
		f.SetCellValue(sheet, cellRef(i+1, 1), h)
		f.SetCellStyle(sheet, cellRef(i+1, 1), cellRef(i+1, 1), headerStyle)
	}

	for i, p := range paths {
		row := i + 2
		f.SetCellValue(sheet, cellRef(1, row), p.Id)
		f.SetCellValue(sheet, cellRef(2, row), p.Name)
		f.SetCellValue(sheet, cellRef(3, row), p.UserName)
		f.SetCellValue(sheet, cellRef(4, row), p.UserEmail)
		f.SetCellValue(sheet, cellRef(5, row), p.RiskLevel)
		f.SetCellValue(sheet, cellRef(6, row), p.Status)
		f.SetCellValue(sheet, cellRef(7, row), p.FailCount)
		f.SetCellValue(sheet, cellRef(8, row), p.TotalCourses)
		f.SetCellValue(sheet, cellRef(9, row), p.CompletedCount)
		f.SetCellValue(sheet, cellRef(10, row), p.DueDate.Format("2006-01-02"))
		f.SetCellValue(sheet, cellRef(11, row), p.CreatedDate.Format("2006-01-02"))

		// Color-code by risk or status
		if p.RiskLevel == models.RiskLevelCritical {
			for c := 1; c <= 11; c++ {
				f.SetCellStyle(sheet, cellRef(c, row), cellRef(c, row), critStyle)
			}
		} else if p.Status == models.RemediationStatusCompleted {
			for c := 1; c <= 11; c++ {
				f.SetCellStyle(sheet, cellRef(c, row), cellRef(c, row), compStyle)
			}
		}
	}

	// Steps sheet
	stepSheet := "Steps"
	f.NewSheet(stepSheet)
	stepHeaders := []string{"Path ID", "Path Name", "Step Order", "Course", "Required", "Status", "Completed"}
	for i, h := range stepHeaders {
		f.SetCellValue(stepSheet, cellRef(i+1, 1), h)
		f.SetCellStyle(stepSheet, cellRef(i+1, 1), cellRef(i+1, 1), headerStyle)
	}
	stepRow := 2
	for _, p := range paths {
		for _, s := range p.Steps {
			completed := ""
			if !s.CompletedDate.IsZero() {
				completed = s.CompletedDate.Format("2006-01-02")
			}
			f.SetCellValue(stepSheet, cellRef(1, stepRow), p.Id)
			f.SetCellValue(stepSheet, cellRef(2, stepRow), p.Name)
			f.SetCellValue(stepSheet, cellRef(3, stepRow), s.SortOrder)
			f.SetCellValue(stepSheet, cellRef(4, stepRow), s.CourseName)
			f.SetCellValue(stepSheet, cellRef(5, stepRow), s.Required)
			f.SetCellValue(stepSheet, cellRef(6, stepRow), s.Status)
			f.SetCellValue(stepSheet, cellRef(7, stepRow), completed)
			stepRow++
		}
	}

	// Summary sheet
	sumSheet := "Summary"
	f.NewSheet(sumSheet)
	summary, _ := models.GetRemediationSummary(orgId)
	sumRows := [][]interface{}{
		{"Total Paths", summary.TotalPaths},
		{"Active", summary.ActivePaths},
		{"Completed", summary.CompletedPaths},
		{"Cancelled", summary.CancelledPaths},
		{"Expired", summary.ExpiredPaths},
		{"Critical Count", summary.CriticalCount},
		{"High Count", summary.HighCount},
		{"Avg Completion (%)", summary.AvgCompletion},
	}
	for i, d := range sumRows {
		f.SetCellValue(sumSheet, cellRef(1, i+1), d[0])
		f.SetCellValue(sumSheet, cellRef(2, i+1), d[1])
	}

	f.SetColWidth(sheet, "A", "A", 8)
	f.SetColWidth(sheet, "B", "B", 35)
	f.SetColWidth(sheet, "C", "D", 25)
	f.SetColWidth(stepSheet, "A", "A", 8)
	f.SetColWidth(stepSheet, "B", "B", 35)
	f.SetColWidth(stepSheet, "D", "D", 30)
	f.SetColWidth(sumSheet, "A", "A", 20)

	filename := fmt.Sprintf("nivoxis-remediation-%s.xlsx", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if err := f.Write(w); err != nil {
		log.Error(err)
	}
}
