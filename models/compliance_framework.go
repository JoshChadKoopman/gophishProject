package models

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ComplianceFramework represents a regulatory framework (e.g., NIS2, DORA).
type ComplianceFramework struct {
	Id          int64     `json:"id" gorm:"column:id; primary_key:yes"`
	Slug        string    `json:"slug" gorm:"column:slug"`
	Name        string    `json:"name" gorm:"column:name"`
	Version     string    `json:"version" gorm:"column:version"`
	Description string    `json:"description" gorm:"column:description"`
	Region      string    `json:"region" gorm:"column:region"`
	IsActive    bool      `json:"is_active" gorm:"column:is_active"`
	CreatedDate time.Time `json:"created_date" gorm:"column:created_date"`
}

// ComplianceControl represents a specific requirement within a framework.
type ComplianceControl struct {
	Id              int64  `json:"id" gorm:"column:id; primary_key:yes"`
	FrameworkId     int64  `json:"framework_id" gorm:"column:framework_id"`
	ControlRef      string `json:"control_ref" gorm:"column:control_ref"`
	Title           string `json:"title" gorm:"column:title"`
	Description     string `json:"description" gorm:"column:description"`
	Category        string `json:"category" gorm:"column:category"`
	EvidenceType    string `json:"evidence_type" gorm:"column:evidence_type"`
	EvidenceCritera string `json:"evidence_criteria" gorm:"column:evidence_criteria"`
	SortOrder       int    `json:"sort_order" gorm:"column:sort_order"`

	// Populated at query time
	Status       string  `json:"status,omitempty" gorm:"-"`
	Score        float64 `json:"score,omitempty" gorm:"-"`
	Evidence     string  `json:"evidence,omitempty" gorm:"-"`
	LastAssessed string  `json:"last_assessed,omitempty" gorm:"-"`
}

// OrgComplianceMapping links an organization to frameworks they must comply with.
type OrgComplianceMapping struct {
	Id          int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId       int64     `json:"org_id" gorm:"column:org_id"`
	FrameworkId int64     `json:"framework_id" gorm:"column:framework_id"`
	EnabledDate time.Time `json:"enabled_date" gorm:"column:enabled_date"`
	IsActive    bool      `json:"is_active" gorm:"column:is_active"`
}

// ComplianceAssessment stores point-in-time compliance evaluation results.
type ComplianceAssessment struct {
	Id            int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId         int64     `json:"org_id" gorm:"column:org_id"`
	FrameworkId   int64     `json:"framework_id" gorm:"column:framework_id"`
	ControlId     int64     `json:"control_id" gorm:"column:control_id"`
	Status        string    `json:"status" gorm:"column:status"`
	Score         float64   `json:"score" gorm:"column:score"`
	Evidence      string    `json:"evidence" gorm:"column:evidence;type:text"`
	AssessedDate  time.Time `json:"assessed_date" gorm:"column:assessed_date"`
	AssessedBy    int64     `json:"assessed_by" gorm:"column:assessed_by"`
	Notes         string    `json:"notes" gorm:"column:notes;type:text"`
}

// Compliance status constants.
const (
	ComplianceStatusCompliant    = "compliant"
	ComplianceStatusPartial      = "partial"
	ComplianceStatusNonCompliant = "non_compliant"
	ComplianceStatusNotAssessed  = "not_assessed"
)

const dateFormatISO = "2006-01-02"

// Evidence type constants define how compliance is verified.
const (
	EvidenceTypeSimulationRate = "simulation_rate"
	EvidenceTypeTrainingRate   = "training_rate"
	EvidenceTypeQuizPassRate   = "quiz_pass_rate"
	EvidenceTypeBRSScore       = "brs_score"
	EvidenceTypeReportRate     = "report_rate"
	EvidenceTypeCertification  = "certification"
	EvidenceTypeManual         = "manual"
)

// FrameworkSummary is the high-level compliance posture for an org + framework.
type FrameworkSummary struct {
	Framework        ComplianceFramework `json:"framework"`
	TotalControls    int                 `json:"total_controls"`
	Compliant        int                 `json:"compliant"`
	Partial          int                 `json:"partial"`
	NonCompliant     int                 `json:"non_compliant"`
	NotAssessed      int                 `json:"not_assessed"`
	OverallScore     float64             `json:"overall_score"`
	Controls         []ComplianceControl `json:"controls,omitempty"`
	LastAssessedDate string              `json:"last_assessed_date"`
}

// ComplianceDashboard is the full compliance posture across all enabled frameworks.
type ComplianceDashboard struct {
	Frameworks    []FrameworkSummary `json:"frameworks"`
	OverallScore  float64            `json:"overall_score"`
	TotalControls int                `json:"total_controls"`
	Compliant     int                `json:"compliant"`
}

// EvidenceCriteria is the parsed form of evidence_criteria JSON.
type EvidenceCriteria struct {
	Metric    string  `json:"metric"`
	Threshold float64 `json:"threshold"`
	Operator  string  `json:"operator"` // "gte", "lte", "eq"
}

// GetComplianceFrameworks returns all available frameworks.
func GetComplianceFrameworks() ([]ComplianceFramework, error) {
	frameworks := []ComplianceFramework{}
	err := db.Where("is_active = 1").Order("name asc").Find(&frameworks).Error
	return frameworks, err
}

// GetComplianceFramework returns a single framework by ID.
func GetComplianceFramework(id int64) (ComplianceFramework, error) {
	f := ComplianceFramework{}
	err := db.Where("id = ?", id).First(&f).Error
	return f, err
}

// GetFrameworkControls returns all controls for a framework.
func GetFrameworkControls(frameworkId int64) ([]ComplianceControl, error) {
	controls := []ComplianceControl{}
	err := db.Where("framework_id = ?", frameworkId).Order("sort_order asc").Find(&controls).Error
	return controls, err
}

// GetOrgFrameworks returns frameworks enabled for an organization.
func GetOrgFrameworks(orgId int64) ([]ComplianceFramework, error) {
	frameworks := []ComplianceFramework{}
	err := db.Raw(`
		SELECT cf.* FROM compliance_frameworks cf
		JOIN org_compliance_mappings ocm ON cf.id = ocm.framework_id
		WHERE ocm.org_id = ? AND ocm.is_active = 1 AND cf.is_active = 1
		ORDER BY cf.name ASC
	`, orgId).Scan(&frameworks).Error
	return frameworks, err
}

// EnableOrgFramework enables a compliance framework for an organization.
func EnableOrgFramework(orgId, frameworkId int64) error {
	existing := OrgComplianceMapping{}
	err := db.Where("org_id = ? AND framework_id = ?", orgId, frameworkId).First(&existing).Error
	if err == nil {
		// Already exists, re-enable
		return db.Table("org_compliance_mappings").Where("id = ?", existing.Id).Updates(map[string]interface{}{
			"is_active":    true,
			"enabled_date": time.Now().UTC(),
		}).Error
	}
	mapping := OrgComplianceMapping{
		OrgId:       orgId,
		FrameworkId: frameworkId,
		EnabledDate: time.Now().UTC(),
		IsActive:    true,
	}
	return db.Save(&mapping).Error
}

// DisableOrgFramework disables a compliance framework for an organization.
func DisableOrgFramework(orgId, frameworkId int64) error {
	return db.Table("org_compliance_mappings").
		Where("org_id = ? AND framework_id = ?", orgId, frameworkId).
		Update("is_active", false).Error
}

// SaveComplianceAssessment saves a manual or automated compliance assessment for a control.
func SaveComplianceAssessment(a *ComplianceAssessment) error {
	a.AssessedDate = time.Now().UTC()
	return db.Save(a).Error
}

// GetLatestAssessment returns the most recent assessment for a control in an org.
func GetLatestAssessment(orgId, controlId int64) (ComplianceAssessment, error) {
	a := ComplianceAssessment{}
	err := db.Where("org_id = ? AND control_id = ?", orgId, controlId).
		Order("assessed_date DESC").First(&a).Error
	return a, err
}

// GetFrameworkAssessments returns all latest assessments for a framework in an org.
func GetFrameworkAssessments(orgId, frameworkId int64) ([]ComplianceAssessment, error) {
	assessments := []ComplianceAssessment{}
	err := db.Raw(`
		SELECT ca.* FROM compliance_assessments ca
		INNER JOIN (
			SELECT control_id, MAX(assessed_date) as max_date
			FROM compliance_assessments
			WHERE org_id = ? AND framework_id = ?
			GROUP BY control_id
		) latest ON ca.control_id = latest.control_id AND ca.assessed_date = latest.max_date
		WHERE ca.org_id = ? AND ca.framework_id = ?
	`, orgId, frameworkId, orgId, frameworkId).Scan(&assessments).Error
	return assessments, err
}

// GetComplianceDashboard returns the full compliance posture for an org.
func GetComplianceDashboard(orgId int64) (ComplianceDashboard, error) {
	dashboard := ComplianceDashboard{}
	frameworks, err := GetOrgFrameworks(orgId)
	if err != nil {
		return dashboard, err
	}

	var totalScore float64
	for _, f := range frameworks {
		summary, err := GetFrameworkSummary(orgId, f.Id, false)
		if err != nil {
			log.Error(err)
			continue
		}
		dashboard.Frameworks = append(dashboard.Frameworks, summary)
		dashboard.TotalControls += summary.TotalControls
		dashboard.Compliant += summary.Compliant
		totalScore += summary.OverallScore
	}

	if len(dashboard.Frameworks) > 0 {
		dashboard.OverallScore = math.Round(totalScore*100/float64(len(dashboard.Frameworks))) / 100
	}

	return dashboard, nil
}

// GetFrameworkSummary returns the compliance summary for a specific framework.
func GetFrameworkSummary(orgId, frameworkId int64, includeControls bool) (FrameworkSummary, error) {
	summary := FrameworkSummary{}

	f, err := GetComplianceFramework(frameworkId)
	if err != nil {
		return summary, err
	}
	summary.Framework = f

	controls, err := GetFrameworkControls(frameworkId)
	if err != nil {
		return summary, err
	}
	summary.TotalControls = len(controls)

	// Get latest assessments
	assessments, _ := GetFrameworkAssessments(orgId, frameworkId)
	assessmentMap := make(map[int64]ComplianceAssessment)
	for _, a := range assessments {
		assessmentMap[a.ControlId] = a
	}

	var totalScore float64
	for i := range controls {
		if a, ok := assessmentMap[controls[i].Id]; ok {
			controls[i].Status = a.Status
			controls[i].Score = a.Score
			controls[i].Evidence = a.Evidence
			controls[i].LastAssessed = a.AssessedDate.Format(dateFormatISO)
			if a.AssessedDate.Format(dateFormatISO) > summary.LastAssessedDate {
				summary.LastAssessedDate = a.AssessedDate.Format(dateFormatISO)
			}
			switch a.Status {
			case ComplianceStatusCompliant:
				summary.Compliant++
				totalScore += 100
			case ComplianceStatusPartial:
				summary.Partial++
				totalScore += a.Score
			case ComplianceStatusNonCompliant:
				summary.NonCompliant++
			default:
				summary.NotAssessed++
			}
		} else {
			controls[i].Status = ComplianceStatusNotAssessed
			summary.NotAssessed++
		}
	}

	if summary.TotalControls > 0 {
		summary.OverallScore = math.Round(totalScore*100/float64(summary.TotalControls)) / 100
	}

	if includeControls {
		summary.Controls = controls
	}

	return summary, nil
}

// evidenceResult holds the output of a metric collection.
type evidenceResult struct {
	Value       float64
	Description string
}

// collectEvidence fetches the metric value and description for a control's evidence type.
func collectEvidence(orgId int64, evidenceType string, criteria EvidenceCriteria) (evidenceResult, error) {
	scope := OrgScope{OrgId: orgId}

	switch evidenceType {
	case EvidenceTypeSimulationRate:
		overview, err := GetReportOverview(scope)
		if err != nil {
			return evidenceResult{}, err
		}
		return collectSimulationMetric(overview, criteria.Metric), nil

	case EvidenceTypeTrainingRate:
		summary, err := GetTrainingSummaryReport(scope)
		if err != nil {
			return evidenceResult{}, err
		}
		return evidenceResult{
			Value:       summary.CompletionRate,
			Description: formatF("Training completion: %.1f%% (%d/%d)", summary.CompletionRate, summary.CompletedCount, summary.TotalAssignments),
		}, nil

	case EvidenceTypeQuizPassRate:
		summary, err := GetTrainingSummaryReport(scope)
		if err != nil {
			return evidenceResult{}, err
		}
		return evidenceResult{Value: summary.AvgQuizScore, Description: formatF("Avg quiz score: %.1f%%", summary.AvgQuizScore)}, nil

	case EvidenceTypeBRSScore:
		var avgScore float64
		if err := db.Raw(`SELECT COALESCE(AVG(composite_score), 0) FROM user_risk_scores WHERE org_id = ?`, orgId).Row().Scan(&avgScore); err != nil {
			return evidenceResult{}, err
		}
		return evidenceResult{Value: avgScore, Description: formatF("Org avg BRS: %.1f", avgScore)}, nil

	case EvidenceTypeReportRate:
		overview, err := GetReportOverview(scope)
		if err != nil {
			return evidenceResult{}, err
		}
		return evidenceResult{Value: overview.AvgReportRate, Description: formatF("Phishing report rate: %.1f%%", overview.AvgReportRate)}, nil

	case EvidenceTypeCertification:
		return collectCertEvidence(orgId), nil
	}

	return evidenceResult{}, nil
}

func collectSimulationMetric(overview ReportOverview, metric string) evidenceResult {
	switch metric {
	case "click_rate":
		return evidenceResult{Value: overview.AvgClickRate, Description: formatF("Avg click rate: %.1f%%", overview.AvgClickRate)}
	case "report_rate":
		return evidenceResult{Value: overview.AvgReportRate, Description: formatF("Avg report rate: %.1f%%", overview.AvgReportRate)}
	case "campaigns_run":
		return evidenceResult{Value: float64(overview.TotalCampaigns), Description: formatF("Total campaigns: %d", overview.TotalCampaigns)}
	}
	return evidenceResult{}
}

func collectCertEvidence(orgId int64) evidenceResult {
	var certCount, totalUsers int
	db.Raw(`SELECT COUNT(DISTINCT user_id) FROM user_compliance_certs WHERE certification_id IN (
		SELECT id FROM compliance_certifications WHERE org_id = ? OR org_id = 0
	)`, orgId).Row().Scan(&certCount)
	db.Raw(`SELECT COUNT(*) FROM users WHERE org_id = ?`, orgId).Row().Scan(&totalUsers)
	var pct float64
	if totalUsers > 0 {
		pct = float64(certCount) * 100 / float64(totalUsers)
	}
	return evidenceResult{Value: pct, Description: formatF("Certified users: %d/%d (%.1f%%)", certCount, totalUsers, pct)}
}

// deriveComplianceStatus determines the compliance status and score from a metric value.
func deriveComplianceStatus(value float64, criteria EvidenceCriteria) (string, float64) {
	if evaluateCriteria(value, criteria) {
		return ComplianceStatusCompliant, 100
	}
	if value <= 0 {
		return ComplianceStatusNonCompliant, 0
	}
	score := 0.0
	if criteria.Threshold > 0 {
		if criteria.Operator == "lte" {
			if value <= criteria.Threshold*2 {
				score = math.Max(0, (1-(value-criteria.Threshold)/criteria.Threshold)*100)
			}
		} else {
			score = math.Min(99, value/criteria.Threshold*100)
		}
	}
	return ComplianceStatusPartial, math.Round(score*100) / 100
}

// AutoAssessControl evaluates a control's compliance status based on platform data.
func AutoAssessControl(orgId int64, control ComplianceControl) (*ComplianceAssessment, error) {
	if control.EvidenceType == EvidenceTypeManual {
		return nil, nil
	}

	criteria := EvidenceCriteria{}
	if err := json.Unmarshal([]byte(control.EvidenceCritera), &criteria); err != nil {
		return nil, err
	}

	ev, err := collectEvidence(orgId, control.EvidenceType, criteria)
	if err != nil {
		return nil, err
	}

	status, score := deriveComplianceStatus(ev.Value, criteria)

	assessment := &ComplianceAssessment{
		OrgId:        orgId,
		FrameworkId:  control.FrameworkId,
		ControlId:    control.Id,
		Status:       status,
		Score:        score,
		Evidence:     ev.Description,
		AssessedDate: time.Now().UTC(),
		AssessedBy:   0,
	}

	return assessment, db.Save(assessment).Error
}

// AutoAssessFramework runs auto-assessment on all auto-assessable controls in a framework.
func AutoAssessFramework(orgId, frameworkId int64) (FrameworkSummary, error) {
	controls, err := GetFrameworkControls(frameworkId)
	if err != nil {
		return FrameworkSummary{}, err
	}

	for _, c := range controls {
		if c.EvidenceType != EvidenceTypeManual {
			if _, err := AutoAssessControl(orgId, c); err != nil {
				log.Errorf("auto-assess control %d: %v", c.Id, err)
			}
		}
	}

	return GetFrameworkSummary(orgId, frameworkId, true)
}

func evaluateCriteria(value float64, c EvidenceCriteria) bool {
	switch c.Operator {
	case "gte":
		return value >= c.Threshold
	case "lte":
		return value <= c.Threshold
	case "eq":
		return value == c.Threshold
	case "gt":
		return value > c.Threshold
	case "lt":
		return value < c.Threshold
	default:
		return value >= c.Threshold
	}
}

func formatF(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
