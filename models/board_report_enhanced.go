package models

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Board Report Narrative Generation & Enhanced Features ───────
// Adds: AI-generated executive summary, period-over-period deltas,
// department-level risk heatmap, and approval workflow with audit trail.

// ── Approval Workflow ───────────────────────────────────────────
// Status progression: draft → review → approved → published
// Each transition is recorded in board_report_approvals for governance.

const (
	BoardReportStatusReview   = "review"
	BoardReportStatusApproved = "approved"
	// BoardReportStatusDraft and BoardReportStatusPublished already exist in board_report.go
)

// ValidBoardReportTransitions defines the allowed status transitions.
var ValidBoardReportTransitions = map[string][]string{
	BoardReportStatusDraft:     {BoardReportStatusReview},
	BoardReportStatusReview:    {BoardReportStatusApproved, BoardReportStatusDraft},
	BoardReportStatusApproved:  {BoardReportStatusPublished, BoardReportStatusDraft},
	BoardReportStatusPublished: {BoardReportStatusDraft},
}

// BoardReportApproval records a status transition for audit trail / governance.
type BoardReportApproval struct {
	Id          int64     `json:"id" gorm:"primary_key"`
	OrgId       int64     `json:"org_id" gorm:"column:org_id"`
	ReportId    int64     `json:"report_id" gorm:"column:report_id;index"`
	FromStatus  string    `json:"from_status" gorm:"column:from_status;size:20"`
	ToStatus    string    `json:"to_status" gorm:"column:to_status;size:20"`
	UserId      int64     `json:"user_id" gorm:"column:user_id"`
	Username    string    `json:"username" gorm:"column:username;size:200"`
	Comment     string    `json:"comment" gorm:"column:comment;type:text"`
	CreatedDate time.Time `json:"created_date" gorm:"column:created_date"`
}

func (BoardReportApproval) TableName() string { return "board_report_approvals" }

// TransitionBoardReportStatus validates and applies a status transition,
// recording it in the approval audit trail.
func TransitionBoardReportStatus(reportId, orgId int64, newStatus string, user User, comment string) error {
	br, err := GetBoardReport(reportId, orgId)
	if err != nil {
		return err
	}

	// Validate the transition
	allowed, ok := ValidBoardReportTransitions[br.Status]
	if !ok {
		return fmt.Errorf("current status %q does not support transitions", br.Status)
	}
	valid := false
	for _, s := range allowed {
		if s == newStatus {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("cannot transition from %q to %q", br.Status, newStatus)
	}

	// Record the approval
	approval := BoardReportApproval{
		OrgId:       orgId,
		ReportId:    reportId,
		FromStatus:  br.Status,
		ToStatus:    newStatus,
		UserId:      user.Id,
		Username:    user.Username,
		Comment:     comment,
		CreatedDate: time.Now().UTC(),
	}
	if err := db.Create(&approval).Error; err != nil {
		return fmt.Errorf("failed to record approval: %w", err)
	}

	// Update the report status
	br.Status = newStatus
	br.ModifiedDate = time.Now().UTC()
	return db.Save(&br).Error
}

// GetBoardReportApprovals returns the audit trail for a board report.
func GetBoardReportApprovals(reportId, orgId int64) ([]BoardReportApproval, error) {
	var approvals []BoardReportApproval
	err := db.Where("report_id = ? AND org_id = ?", reportId, orgId).
		Order("created_date DESC").
		Find(&approvals).Error
	return approvals, err
}

// ── AI Narrative Content ────────────────────────────────────────
// Stored per-report so admins can review/edit before publishing.

// BoardReportNarrativeContent stores the editable AI-generated narrative.
type BoardReportNarrativeContent struct {
	Id            int64     `json:"id" gorm:"primary_key"`
	OrgId         int64     `json:"org_id" gorm:"column:org_id;index"`
	ReportId      int64     `json:"report_id" gorm:"column:report_id;uniqueIndex"`
	AIGenerated   bool      `json:"ai_generated" gorm:"column:ai_generated"`
	Paragraph1    string    `json:"paragraph1" gorm:"column:paragraph1;type:text"` // Executive overview
	Paragraph2    string    `json:"paragraph2" gorm:"column:paragraph2;type:text"` // Performance comparison
	Paragraph3    string    `json:"paragraph3" gorm:"column:paragraph3;type:text"` // Forward outlook
	FullNarrative string    `json:"full_narrative" gorm:"column:full_narrative;type:text"`
	EditedBy      int64     `json:"edited_by" gorm:"column:edited_by"`
	CreatedDate   time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate  time.Time `json:"modified_date" gorm:"column:modified_date"`
}

func (BoardReportNarrativeContent) TableName() string { return "board_report_narratives" }

// SaveBoardReportNarrative upserts a narrative for a report.
func SaveBoardReportNarrative(n *BoardReportNarrativeContent) error {
	n.ModifiedDate = time.Now().UTC()
	if n.CreatedDate.IsZero() {
		n.CreatedDate = time.Now().UTC()
	}

	var existing BoardReportNarrativeContent
	err := db.Where("report_id = ?", n.ReportId).First(&existing).Error
	if err == nil {
		n.Id = existing.Id
		n.CreatedDate = existing.CreatedDate
	}
	return db.Save(n).Error
}

// GetBoardReportNarrative retrieves the stored narrative for a report.
func GetBoardReportNarrative(reportId int64) (*BoardReportNarrativeContent, error) {
	var n BoardReportNarrativeContent
	err := db.Where("report_id = ?", reportId).First(&n).Error
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// ── Period-Over-Period Deltas ───────────────────────────────────

// PeriodDelta captures a metric's change between two periods.
type PeriodDelta struct {
	Metric       string  `json:"metric"`
	Label        string  `json:"label"`
	CurrentValue float64 `json:"current_value"`
	PriorValue   float64 `json:"prior_value"`
	AbsChange    float64 `json:"abs_change"`    // absolute change
	PctChange    float64 `json:"pct_change"`    // percentage change
	Direction    string  `json:"direction"`     // "up", "down", "flat"
	Favorable    bool    `json:"favorable"`     // whether this direction is good
	Arrow        string  `json:"arrow"`         // "↑", "↓", "→"
	DisplayDelta string  `json:"display_delta"` // e.g. "↑12% training completion"
}

// ComputePeriodDeltas compares two snapshots and produces human-readable deltas.
func ComputePeriodDeltas(current, prior *BoardReportSnapshot) []PeriodDelta {
	if current == nil || prior == nil {
		return nil
	}

	type metricDef struct {
		key    string
		label  string
		curVal float64
		priVal float64
		upGood bool   // is "up" favorable?
		unit   string // "%" or "pp" or "pts"
	}

	defs := []metricDef{
		{"click_rate", "Phishing Click Rate", current.Phishing.AvgClickRate, prior.Phishing.AvgClickRate, false, "pp"},
		{"submit_rate", "Data Submission Rate", current.Phishing.AvgSubmitRate, prior.Phishing.AvgSubmitRate, false, "pp"},
		{"report_rate", "Phishing Report Rate", current.Phishing.AvgReportRate, prior.Phishing.AvgReportRate, true, "pp"},
		{"training_completion", "Training Completion", current.Training.CompletionRate, prior.Training.CompletionRate, true, "%"},
		{"quiz_score", "Avg Quiz Score", current.Training.AvgQuizScore, prior.Training.AvgQuizScore, true, "%"},
		{"compliance_score", "Compliance Score", current.Compliance.OverallScore, prior.Compliance.OverallScore, true, "%"},
		{"hygiene_score", "Hygiene Score", current.Hygiene.AvgScore, prior.Hygiene.AvgScore, true, "%"},
		{"risk_score", "Avg Risk Score", current.Risk.AvgRiskScore, prior.Risk.AvgRiskScore, false, "pts"},
		{"posture_score", "Security Posture Score", current.SecurityPostureScore, prior.SecurityPostureScore, true, "pts"},
	}

	deltas := make([]PeriodDelta, 0, len(defs))
	for _, d := range defs {
		absChange := d.curVal - d.priVal
		pctChange := 0.0
		if d.priVal != 0 {
			pctChange = math.Round(absChange/d.priVal*1000) / 10 // 1 decimal
		}

		direction := "flat"
		arrow := "→"
		if absChange > 0.5 {
			direction = "up"
			arrow = "↑"
		} else if absChange < -0.5 {
			direction = "down"
			arrow = "↓"
		}

		favorable := false
		if direction == "up" && d.upGood {
			favorable = true
		} else if direction == "down" && !d.upGood {
			favorable = true
		} else if direction == "flat" {
			favorable = true
		}

		displayDelta := fmt.Sprintf("%s%.1f%s %s", arrow, math.Abs(absChange), d.unit, d.label)

		deltas = append(deltas, PeriodDelta{
			Metric:       d.key,
			Label:        d.label,
			CurrentValue: d.curVal,
			PriorValue:   d.priVal,
			AbsChange:    math.Round(absChange*10) / 10,
			PctChange:    pctChange,
			Direction:    direction,
			Favorable:    favorable,
			Arrow:        arrow,
			DisplayDelta: displayDelta,
		})
	}
	return deltas
}

// ── Department Risk Heatmap ─────────────────────────────────────
// Rows = departments, Columns = risk factors (click rate, training
// completion, hygiene score, risk score, compliance score).

// DeptHeatmapCell represents a single cell in the heatmap.
type DeptHeatmapCell struct {
	Value float64 `json:"value"`
	Level string  `json:"level"` // "low", "medium", "high", "critical"
	Color string  `json:"color"` // hex colour for rendering
}

// DeptHeatmapRow represents one department's row in the heatmap.
type DeptHeatmapRow struct {
	Department string                     `json:"department"`
	UserCount  int                        `json:"user_count"`
	Cells      map[string]DeptHeatmapCell `json:"cells"`
}

// DeptHeatmapColumn describes a heatmap column.
type DeptHeatmapColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// DeptHeatmap is the full heatmap payload.
type DeptHeatmap struct {
	Columns []DeptHeatmapColumn `json:"columns"`
	Rows    []DeptHeatmapRow    `json:"rows"`
}

// heatmapColumns defines the risk factor columns.
var heatmapColumns = []DeptHeatmapColumn{
	{Key: "click_rate", Label: "Click Rate"},
	{Key: "training_completion", Label: "Training Completion"},
	{Key: "hygiene_score", Label: "Hygiene Score"},
	{Key: "risk_score", Label: "Risk Score (BRS)"},
	{Key: "report_rate", Label: "Report Rate"},
}

// GenerateDeptHeatmap builds the department-level risk heatmap.
func GenerateDeptHeatmap(orgId int64) (*DeptHeatmap, error) {
	scope := OrgScope{OrgId: orgId}

	// 1. Gather department-level BRS scores
	deptBRS, _ := GetDepartmentBRS(scope)
	deptScores := map[string]DepartmentRiskScore{}
	for _, d := range deptBRS {
		deptScores[d.Department] = d
	}

	// 2. Get per-department phishing stats
	type deptPhish struct {
		Department    string
		UserCount     int
		AvgClickRate  float64
		AvgReportRate float64
	}
	var deptPhishing []deptPhish
	db.Raw(`
		SELECT u.department,
			COUNT(DISTINCT u.id) as user_count,
			COALESCE(AVG(CASE WHEN r.status IN ('Clicked Link','Submitted Data') THEN 100.0 ELSE 0 END), 0) as avg_click_rate,
			COALESCE(AVG(CASE WHEN r.reported = 1 THEN 100.0 ELSE 0 END), 0) as avg_report_rate
		FROM users u
		LEFT JOIN results r ON r.email = u.email
		LEFT JOIN campaigns c ON r.campaign_id = c.id AND c.org_id = ?
		WHERE u.org_id = ? AND u.department != '' AND u.department IS NOT NULL
		GROUP BY u.department
		ORDER BY u.department
	`, orgId, orgId).Scan(&deptPhishing)

	// 3. Get per-department training completion
	type deptTrain struct {
		Department     string
		CompletionRate float64
	}
	var deptTraining []deptTrain
	db.Raw(`
		SELECT u.department,
			COALESCE(
				100.0 * SUM(CASE WHEN ta.status = 'completed' THEN 1 ELSE 0 END) /
				NULLIF(COUNT(ta.id), 0),
			0) as completion_rate
		FROM users u
		LEFT JOIN training_assignments ta ON ta.user_id = u.id
		WHERE u.org_id = ? AND u.department != '' AND u.department IS NOT NULL
		GROUP BY u.department
	`, orgId).Scan(&deptTraining)
	trainMap := map[string]float64{}
	for _, dt := range deptTraining {
		trainMap[dt.Department] = math.Round(dt.CompletionRate*10) / 10
	}

	// 4. Get per-department hygiene scores
	type deptHyg struct {
		Department string
		AvgScore   float64
	}
	var deptHygiene []deptHyg
	db.Raw(`
		SELECT u.department,
			COALESCE(AVG(chd.score), 0) as avg_score
		FROM users u
		LEFT JOIN cyber_hygiene_devices chd ON chd.user_id = u.id AND chd.org_id = ?
		WHERE u.org_id = ? AND u.department != '' AND u.department IS NOT NULL
		GROUP BY u.department
	`, orgId, orgId).Scan(&deptHygiene)
	hygMap := map[string]float64{}
	for _, dh := range deptHygiene {
		hygMap[dh.Department] = math.Round(dh.AvgScore*10) / 10
	}

	// 5. Build the heatmap
	heatmap := &DeptHeatmap{Columns: heatmapColumns}

	// Collect all departments
	allDepts := map[string]int{}
	for _, dp := range deptPhishing {
		allDepts[dp.Department] = dp.UserCount
	}
	// Add departments from BRS that may not be in phishing results
	for dept, ds := range deptScores {
		if _, ok := allDepts[dept]; !ok {
			allDepts[dept] = ds.UserCount
		}
	}

	for dept, userCount := range allDepts {
		row := DeptHeatmapRow{
			Department: dept,
			UserCount:  userCount,
			Cells:      make(map[string]DeptHeatmapCell),
		}

		// Click rate (lower is better)
		clickRate := 0.0
		reportRate := 0.0
		for _, dp := range deptPhishing {
			if dp.Department == dept {
				clickRate = math.Round(dp.AvgClickRate*10) / 10
				reportRate = math.Round(dp.AvgReportRate*10) / 10
				break
			}
		}
		row.Cells["click_rate"] = heatCell(clickRate, true)    // lower is better
		row.Cells["report_rate"] = heatCell(reportRate, false) // higher is better

		// Training completion (higher is better)
		row.Cells["training_completion"] = heatCell(trainMap[dept], false)

		// Hygiene score (higher is better)
		row.Cells["hygiene_score"] = heatCell(hygMap[dept], false)

		// BRS risk score (lower is better = less risky)
		brsScore := 0.0
		if ds, ok := deptScores[dept]; ok {
			brsScore = math.Round(ds.CompositeScore*10) / 10
		}
		row.Cells["risk_score"] = heatCell(brsScore, true) // lower is better

		heatmap.Rows = append(heatmap.Rows, row)
	}

	return heatmap, nil
}

// heatCell classifies a value into a heatmap level with colour.
// lowerIsBetter inverts the thresholds.
func heatCell(value float64, lowerIsBetter bool) DeptHeatmapCell {
	// Normalise: for "lower is better", high values are bad
	level := "low"
	color := "#2ecc71" // green

	if lowerIsBetter {
		switch {
		case value >= 40:
			level = "critical"
			color = "#e74c3c"
		case value >= 25:
			level = "high"
			color = "#e67e22"
		case value >= 10:
			level = "medium"
			color = "#f1c40f"
		default:
			level = "low"
			color = "#2ecc71"
		}
	} else {
		// Higher is better (training completion, report rate, hygiene)
		switch {
		case value >= 80:
			level = "low"
			color = "#2ecc71"
		case value >= 60:
			level = "medium"
			color = "#f1c40f"
		case value >= 40:
			level = "high"
			color = "#e67e22"
		default:
			level = "critical"
			color = "#e74c3c"
		}
	}

	return DeptHeatmapCell{
		Value: value,
		Level: level,
		Color: color,
	}
}

// ── AI Narrative Prompt Builder ─────────────────────────────────
// Builds a structured prompt from metrics for the LLM to generate a
// 3-paragraph board-ready executive summary.

// BuildBoardNarrativePrompt creates the system and user prompts for
// AI-generated executive narrative.
func BuildBoardNarrativePrompt(snap *BoardReportSnapshot, deltas []PeriodDelta, heatmap *DeptHeatmap) (systemPrompt, userPrompt string) {
	systemPrompt = `You are a cybersecurity executive report writer for a CISO/board audience.
Generate a professional, data-driven executive summary in EXACTLY 3 paragraphs:

PARAGRAPH 1 — "Security Posture Overview": Summarise the overall security posture score, 
risk trend, and headline performance metrics. Use specific numbers from the data provided.

PARAGRAPH 2 — "Period-Over-Period Performance": Highlight the most significant changes 
between this period and last period. Use the delta data provided. Include specific numbers 
with directional arrows (↑/↓). Mention departments that need attention from the heatmap.

PARAGRAPH 3 — "Forward Outlook & Recommendations": Based on the metrics and trends, 
provide 2-3 actionable recommendations for the board. Be specific and tie each to a metric.

Rules:
- Use professional, concise business language suitable for a board presentation
- Include specific numbers, not vague statements
- Format percentages as "X%" and currency as "$XK" or "$XM"
- Each paragraph should be 3-5 sentences
- Output ONLY the 3 paragraphs, separated by blank lines. No headers or labels.
- Do NOT use markdown formatting, keep it as plain text`

	// Build the data payload for the user prompt
	var b []byte
	data := map[string]interface{}{
		"security_posture_score": snap.SecurityPostureScore,
		"risk_trend":             snap.RiskTrend,
		"phishing": map[string]interface{}{
			"campaigns":   snap.Phishing.TotalCampaigns,
			"recipients":  snap.Phishing.TotalRecipients,
			"click_rate":  snap.Phishing.AvgClickRate,
			"submit_rate": snap.Phishing.AvgSubmitRate,
			"report_rate": snap.Phishing.AvgReportRate,
		},
		"training": map[string]interface{}{
			"completion_rate": snap.Training.CompletionRate,
			"quiz_score":      snap.Training.AvgQuizScore,
			"overdue":         snap.Training.OverdueCount,
			"certificates":    snap.Training.CertificatesIssued,
		},
		"risk": map[string]interface{}{
			"high_risk_users": snap.Risk.HighRiskUsers,
			"avg_risk_score":  snap.Risk.AvgRiskScore,
		},
		"compliance": map[string]interface{}{
			"score":      snap.Compliance.OverallScore,
			"frameworks": snap.Compliance.FrameworkCount,
		},
		"hygiene": map[string]interface{}{
			"avg_score":       snap.Hygiene.AvgScore,
			"at_risk_devices": snap.Hygiene.AtRiskDevices,
		},
		"recommendations": snap.Recommendations,
	}

	// Add deltas
	if len(deltas) > 0 {
		deltaList := make([]map[string]interface{}, 0, len(deltas))
		for _, d := range deltas {
			deltaList = append(deltaList, map[string]interface{}{
				"metric":    d.Label,
				"current":   d.CurrentValue,
				"prior":     d.PriorValue,
				"change":    d.DisplayDelta,
				"favorable": d.Favorable,
			})
		}
		data["period_deltas"] = deltaList
	}

	// Add top-level heatmap concerns
	if heatmap != nil && len(heatmap.Rows) > 0 {
		concerns := []string{}
		for _, row := range heatmap.Rows {
			for _, col := range heatmap.Columns {
				if cell, ok := row.Cells[col.Key]; ok && (cell.Level == "critical" || cell.Level == "high") {
					concerns = append(concerns, fmt.Sprintf("%s: %s = %.1f (%s)", row.Department, col.Label, cell.Value, cell.Level))
				}
			}
		}
		if len(concerns) > 0 {
			data["department_concerns"] = concerns
		}
	}

	b, _ = json.MarshalIndent(data, "", "  ")
	userPrompt = "Generate a 3-paragraph executive summary from the following security metrics:\n\n" + string(b)

	return
}

// BuildDeterministicNarrative generates a 3-paragraph narrative without AI,
// using the same data. This is the fallback when AI is not configured.
func BuildDeterministicNarrative(snap *BoardReportSnapshot, deltas []PeriodDelta, heatmap *DeptHeatmap) *BoardReportNarrativeContent {
	n := &BoardReportNarrativeContent{
		AIGenerated:  false,
		CreatedDate:  time.Now().UTC(),
		ModifiedDate: time.Now().UTC(),
	}

	// Paragraph 1: Security Posture Overview
	p1 := fmt.Sprintf("The organisation's security posture score stands at %.0f/100 (%s), with an overall risk trend assessed as %s. ",
		snap.SecurityPostureScore, postureLabel(snap.SecurityPostureScore), snap.RiskTrend)
	p1 += fmt.Sprintf("Across %d phishing simulation campaigns reaching %d recipients, the average click rate was %.1f%% and the report rate was %.1f%%. ",
		snap.Phishing.TotalCampaigns, snap.Phishing.TotalRecipients, snap.Phishing.AvgClickRate, snap.Phishing.AvgReportRate)
	p1 += fmt.Sprintf("Training completion stands at %.0f%% with an average quiz score of %.0f%%, and compliance posture is %.0f%%.",
		snap.Training.CompletionRate, snap.Training.AvgQuizScore, snap.Compliance.OverallScore)
	n.Paragraph1 = p1

	// Paragraph 2: Period Comparison
	p2 := "Compared to the prior period, "
	significantDeltas := []string{}
	for _, d := range deltas {
		if d.Direction != "flat" && math.Abs(d.AbsChange) > 1 {
			significantDeltas = append(significantDeltas, d.DisplayDelta)
		}
	}
	if len(significantDeltas) > 0 {
		p2 += "the following key changes were observed: "
		for i, sd := range significantDeltas {
			if i > 0 && i < len(significantDeltas)-1 {
				p2 += ", "
			} else if i > 0 {
				p2 += ", and "
			}
			p2 += sd
		}
		p2 += ". "
	} else {
		p2 += "metrics remained broadly stable with no significant changes. "
	}
	// Add heatmap concerns
	if heatmap != nil {
		criticalDepts := []string{}
		for _, row := range heatmap.Rows {
			critCount := 0
			for _, cell := range row.Cells {
				if cell.Level == "critical" || cell.Level == "high" {
					critCount++
				}
			}
			if critCount >= 2 {
				criticalDepts = append(criticalDepts, row.Department)
			}
		}
		if len(criticalDepts) > 0 {
			p2 += fmt.Sprintf("Department-level analysis identifies %d department(s) requiring focused attention: ", len(criticalDepts))
			for i, d := range criticalDepts {
				if i > 0 {
					p2 += ", "
				}
				p2 += d
			}
			p2 += "."
		}
	}
	n.Paragraph2 = p2

	// Paragraph 3: Forward Outlook
	p3 := "Based on the current data, the following actions are recommended: "
	for i, rec := range snap.Recommendations {
		if i >= 3 {
			break // Top 3 for board summary
		}
		if i > 0 {
			p3 += " "
		}
		p3 += fmt.Sprintf("(%d) %s", i+1, rec)
	}
	if snap.SecurityPostureScore >= 70 {
		p3 += " The organisation is well-positioned to withstand common social engineering attacks; maintaining momentum through regular simulations and continuous training is key."
	} else {
		p3 += " Targeted interventions for the identified high-risk departments and improved training engagement will be critical for strengthening the organisation's human firewall."
	}
	n.Paragraph3 = p3

	n.FullNarrative = n.Paragraph1 + "\n\n" + n.Paragraph2 + "\n\n" + n.Paragraph3
	return n
}

// ── Full Enhanced Board Report Payload ──────────────────────────

// FullBoardReportPayload is the complete response for the enhanced board report
// detail endpoint — includes snapshot, narrative, deltas, heatmap, and audit trail.
type FullBoardReportPayload struct {
	BoardReport
	Snapshot   *BoardReportSnapshot         `json:"snapshot,omitempty"`
	Narrative  *BoardReportNarrativeContent `json:"narrative,omitempty"`
	Deltas     []PeriodDelta                `json:"deltas,omitempty"`
	Heatmap    *DeptHeatmap                 `json:"heatmap,omitempty"`
	Approvals  []BoardReportApproval        `json:"approvals,omitempty"`
	ROISummary *ROIMetrics                  `json:"roi_summary,omitempty"`
}

// BuildFullBoardReportPayload assembles all enhanced board report data.
func BuildFullBoardReportPayload(br BoardReport, orgId int64) (*FullBoardReportPayload, error) {
	payload := &FullBoardReportPayload{BoardReport: br}

	// Current snapshot
	snap, err := GenerateBoardReportSnapshot(orgId, br.PeriodStart, br.PeriodEnd)
	if err != nil {
		log.Error(err)
		return payload, err
	}
	payload.Snapshot = snap

	// Prior period snapshot for deltas
	duration := br.PeriodEnd.Sub(br.PeriodStart)
	priorStart := br.PeriodStart.Add(-duration)
	priorSnap, priorErr := GenerateBoardReportSnapshot(orgId, priorStart, br.PeriodStart)
	if priorErr == nil && priorSnap != nil {
		payload.Deltas = ComputePeriodDeltas(snap, priorSnap)
	}

	// Department heatmap
	heatmap, _ := GenerateDeptHeatmap(orgId)
	payload.Heatmap = heatmap

	// Stored narrative
	narrative, narErr := GetBoardReportNarrative(br.Id)
	if narErr == nil {
		payload.Narrative = narrative
	}

	// Approval trail
	approvals, _ := GetBoardReportApprovals(br.Id, orgId)
	payload.Approvals = approvals

	// ROI integration
	roiReport, roiErr := GenerateROIReport(orgId, br.PeriodStart, br.PeriodEnd)
	if roiErr == nil && roiReport != nil {
		payload.ROISummary = &roiReport.Metrics
	}

	return payload, nil
}
