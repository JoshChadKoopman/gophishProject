package models

import (
	"errors"
	"fmt"
	"time"

	log "github.com/gophish/gophish/logger"
)

// RemediationPath represents a targeted training path for repeat phishing offenders.
type RemediationPath struct {
	Id              int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId           int64     `json:"org_id" gorm:"column:org_id"`
	UserId          int64     `json:"user_id" gorm:"column:user_id"`
	UserEmail       string    `json:"user_email" gorm:"column:user_email"`
	EscalationEvent int64     `json:"escalation_event_id,omitempty" gorm:"column:escalation_event_id"`
	Name            string    `json:"name" gorm:"column:name"`
	Description     string    `json:"description,omitempty" gorm:"column:description;type:text"`
	FailCount       int       `json:"fail_count" gorm:"column:fail_count"`
	RiskLevel       string    `json:"risk_level" gorm:"column:risk_level"`
	Status          string    `json:"status" gorm:"column:status"`
	TotalCourses    int       `json:"total_courses" gorm:"column:total_courses"`
	CompletedCount  int       `json:"completed_count" gorm:"column:completed_count"`
	DueDate         time.Time `json:"due_date" gorm:"column:due_date"`
	CompletedDate   time.Time `json:"completed_date,omitempty" gorm:"column:completed_date"`
	CreatedDate     time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate    time.Time `json:"modified_date" gorm:"column:modified_date"`

	Steps    []RemediationStep `json:"steps,omitempty" gorm:"-"`
	UserName string            `json:"user_name,omitempty" gorm:"-"`
}

func (RemediationPath) TableName() string { return "remediation_paths" }

// RemediationStep is a single course in a remediation path.
type RemediationStep struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	PathId         int64     `json:"path_id" gorm:"column:path_id"`
	PresentationId int64     `json:"presentation_id" gorm:"column:presentation_id"`
	SortOrder      int       `json:"sort_order" gorm:"column:sort_order"`
	Required       bool      `json:"required" gorm:"column:required"`
	Status         string    `json:"status" gorm:"column:status"`
	CompletedDate  time.Time `json:"completed_date,omitempty" gorm:"column:completed_date"`
	CourseName     string    `json:"course_name,omitempty" gorm:"-"`
}

func (RemediationStep) TableName() string { return "remediation_steps" }

const (
	RiskLevelLow      = "low"
	RiskLevelMedium   = "medium"
	RiskLevelHigh     = "high"
	RiskLevelCritical = "critical"
)

const (
	RemediationStatusActive    = "active"
	RemediationStatusCompleted = "completed"
	RemediationStatusExpired   = "expired"
	RemediationStatusCancelled = "cancelled"
)

const (
	StepStatusPending   = "pending"
	StepStatusCompleted = "completed"
	StepStatusSkipped   = "skipped"
)

const (
	remQWhereId         = "id = ?"
	remQWhereOrgStatus  = "org_id = ? AND status = ?"
	remQWherePathId     = "path_id = ?"
	remQWherePathPres   = "path_id = ? AND presentation_id = ?"
	remQWherePathStepSt = "path_id = ? AND status = ?"
	remQWhereOrgRisk    = "org_id = ? AND risk_level = ?"
)

var (
	ErrRemediationNotFound    = errors.New("remediation path not found")
	ErrRemediationStepMissing = errors.New("remediation step not found")
)

var validRiskLevels = map[string]bool{
	RiskLevelLow: true, RiskLevelMedium: true,
	RiskLevelHigh: true, RiskLevelCritical: true,
}

// DetermineRiskLevel maps a fail count to a risk level.
func DetermineRiskLevel(failCount int) string {
	switch {
	case failCount >= 8:
		return RiskLevelCritical
	case failCount >= 5:
		return RiskLevelHigh
	case failCount >= 3:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// GetRemediationPaths returns all paths for an org.
func GetRemediationPaths(orgId int64) ([]RemediationPath, error) {
	paths := []RemediationPath{}
	err := db.Where("org_id = ?", orgId).Order("created_date desc").Find(&paths).Error
	if err != nil {
		return paths, err
	}
	for i := range paths {
		hydrateRemediationPath(&paths[i])
	}
	return paths, nil
}

// GetRemediationPath returns a single path by ID, scoped to org.
func GetRemediationPath(id, orgId int64) (RemediationPath, error) {
	p := RemediationPath{}
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&p).Error
	if err != nil {
		return p, ErrRemediationNotFound
	}
	hydrateRemediationPath(&p)
	return p, nil
}

// GetRemediationPathsForUser returns all paths for a user.
func GetRemediationPathsForUser(userId, orgId int64) ([]RemediationPath, error) {
	paths := []RemediationPath{}
	err := db.Where("user_id = ? AND org_id = ?", userId, orgId).
		Order("created_date desc").Find(&paths).Error
	if err != nil {
		return paths, err
	}
	for i := range paths {
		hydrateRemediationPath(&paths[i])
	}
	return paths, nil
}

// PostRemediationPath creates a new remediation path with its steps.
func PostRemediationPath(p *RemediationPath, courseIds []int64) error {
	if p.Name == "" {
		return errors.New("remediation path name is required")
	}
	if len(courseIds) == 0 {
		return errors.New("at least one course is required")
	}
	p.Status = RemediationStatusActive
	p.TotalCourses = len(courseIds)
	p.CompletedCount = 0
	p.CreatedDate = time.Now().UTC()
	p.ModifiedDate = time.Now().UTC()
	if p.RiskLevel == "" {
		p.RiskLevel = DetermineRiskLevel(p.FailCount)
	}
	if !validRiskLevels[p.RiskLevel] {
		p.RiskLevel = RiskLevelLow
	}
	if err := db.Save(p).Error; err != nil {
		return err
	}
	for i, cid := range courseIds {
		step := RemediationStep{
			PathId: p.Id, PresentationId: cid,
			SortOrder: i + 1, Required: true, Status: StepStatusPending,
		}
		if err := db.Save(&step).Error; err != nil {
			log.Errorf("create remediation step: %v", err)
		}
	}
	for _, cid := range courseIds {
		a := &CourseAssignment{
			UserId: p.UserId, PresentationId: cid,
			AssignedBy: 0, Priority: mapRiskToPriority(p.RiskLevel), DueDate: p.DueDate,
		}
		if err := PostAssignment(a); err != nil && err != ErrAssignmentExists {
			log.Errorf("auto-assign remediation course: %v", err)
		}
	}
	return nil
}

// CompleteRemediationStep marks a step as complete and advances the path.
func CompleteRemediationStep(pathId, presentationId int64) error {
	step := RemediationStep{}
	err := db.Where(remQWherePathPres, pathId, presentationId).First(&step).Error
	if err != nil {
		return ErrRemediationStepMissing
	}
	if step.Status == StepStatusCompleted {
		return nil
	}
	now := time.Now().UTC()
	db.Model(&RemediationStep{}).Where(remQWhereId, step.Id).Updates(map[string]interface{}{
		"status": StepStatusCompleted, "completed_date": now,
	})
	return recalcRemediationPath(pathId)
}

// CancelRemediationPath marks a path and all pending steps as cancelled.
func CancelRemediationPath(id, orgId int64) error {
	p, err := GetRemediationPath(id, orgId)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	db.Model(&RemediationPath{}).Where(remQWhereId, p.Id).Updates(map[string]interface{}{
		"status": RemediationStatusCancelled, "modified_date": now,
	})
	db.Model(&RemediationStep{}).Where(remQWherePathStepSt, p.Id, StepStatusPending).Updates(map[string]interface{}{
		"status": StepStatusSkipped,
	})
	return nil
}

// CreateRemediationFromEscalation creates a remediation path from an escalation event.
func CreateRemediationFromEscalation(orgId int64, event EscalationEvent, courseIds []int64) (*RemediationPath, error) {
	if len(courseIds) == 0 {
		return nil, errors.New("no courses specified for remediation")
	}
	riskLevel := DetermineRiskLevel(event.FailCount)
	dueDays := 14
	if riskLevel == RiskLevelCritical {
		dueDays = 7
	}
	path := &RemediationPath{
		OrgId: orgId, UserId: event.UserId, UserEmail: event.UserEmail,
		EscalationEvent: event.Id,
		Name:            fmt.Sprintf("Remediation — %s (Level %d)", event.UserEmail, event.Level),
		Description:     fmt.Sprintf("Targeted remediation for repeat offender with %d failures.", event.FailCount),
		FailCount:       event.FailCount, RiskLevel: riskLevel,
		DueDate: time.Now().UTC().AddDate(0, 0, dueDays),
	}
	if err := PostRemediationPath(path, courseIds); err != nil {
		return nil, err
	}
	return path, nil
}

// EvaluateAndCreateRemediations runs escalation evaluation and auto-creates
// remediation paths for new mandatory_training events.
func EvaluateAndCreateRemediations(orgId int64) ([]RemediationPath, error) {
	events, err := EvaluateAndEscalate(orgId)
	if err != nil {
		return nil, err
	}
	var paths []RemediationPath
	for _, event := range events {
		if event.Action != EscalationActionTraining {
			continue
		}
		policy, pErr := GetEscalationPolicy(event.PolicyId, orgId)
		if pErr != nil || policy.AssignTrainingId <= 0 {
			continue
		}
		path, cErr := CreateRemediationFromEscalation(orgId, event, []int64{policy.AssignTrainingId})
		if cErr != nil {
			log.Errorf("create remediation from escalation: %v", cErr)
			continue
		}
		paths = append(paths, *path)
	}
	return paths, nil
}

// RemediationSummary provides aggregate stats for the admin dashboard.
type RemediationSummary struct {
	TotalPaths     int     `json:"total_paths"`
	ActivePaths    int     `json:"active_paths"`
	CompletedPaths int     `json:"completed_paths"`
	CancelledPaths int     `json:"cancelled_paths"`
	ExpiredPaths   int     `json:"expired_paths"`
	AvgCompletion  float64 `json:"avg_completion_pct"`
	CriticalCount  int     `json:"critical_count"`
	HighCount      int     `json:"high_count"`
}

// GetRemediationSummary returns aggregate statistics.
func GetRemediationSummary(orgId int64) (RemediationSummary, error) {
	s := RemediationSummary{}
	db.Table("remediation_paths").Where("org_id = ?", orgId).Count(&s.TotalPaths)
	db.Table("remediation_paths").Where(remQWhereOrgStatus, orgId, RemediationStatusActive).Count(&s.ActivePaths)
	db.Table("remediation_paths").Where(remQWhereOrgStatus, orgId, RemediationStatusCompleted).Count(&s.CompletedPaths)
	db.Table("remediation_paths").Where(remQWhereOrgStatus, orgId, RemediationStatusCancelled).Count(&s.CancelledPaths)
	db.Table("remediation_paths").Where(remQWhereOrgStatus, orgId, RemediationStatusExpired).Count(&s.ExpiredPaths)
	db.Table("remediation_paths").Where(remQWhereOrgRisk, orgId, RiskLevelCritical).Count(&s.CriticalCount)
	db.Table("remediation_paths").Where(remQWhereOrgRisk, orgId, RiskLevelHigh).Count(&s.HighCount)
	row := db.Table("remediation_paths").Where("org_id = ? AND total_courses > 0", orgId).
		Select("COALESCE(AVG(completed_count * 100.0 / total_courses), 0)").Row()
	row.Scan(&s.AvgCompletion)
	return s, nil
}

// MarkExpiredRemediationPaths transitions active paths past their due date to expired.
func MarkExpiredRemediationPaths(orgId int64) (int, error) {
	now := time.Now().UTC()
	result := db.Model(&RemediationPath{}).
		Where("org_id = ? AND due_date < ? AND due_date != ? AND status = ?",
			orgId, now, time.Time{}, RemediationStatusActive).
		Updates(map[string]interface{}{"status": RemediationStatusExpired, "modified_date": now})
	return int(result.RowsAffected), result.Error
}

func hydrateRemediationPath(p *RemediationPath) {
	steps := []RemediationStep{}
	db.Where(remQWherePathId, p.Id).Order("sort_order asc").Find(&steps)
	for i := range steps {
		tp := TrainingPresentation{}
		if db.Where(remQWhereId, steps[i].PresentationId).First(&tp).Error == nil {
			steps[i].CourseName = tp.Name
		}
	}
	p.Steps = steps
	if p.UserId > 0 {
		u, err := GetUser(p.UserId)
		if err == nil {
			p.UserName = u.FirstName + " " + u.LastName
		}
	}
}

func recalcRemediationPath(pathId int64) error {
	var completed, total int
	db.Table("remediation_steps").Where(remQWherePathId, pathId).Count(&total)
	db.Table("remediation_steps").Where(remQWherePathStepSt, pathId, StepStatusCompleted).Count(&completed)
	updates := map[string]interface{}{
		"completed_count": completed, "total_courses": total, "modified_date": time.Now().UTC(),
	}
	if total > 0 && completed >= total {
		updates["status"] = RemediationStatusCompleted
		updates["completed_date"] = time.Now().UTC()
	}
	return db.Model(&RemediationPath{}).Where(remQWhereId, pathId).Updates(updates).Error
}

func mapRiskToPriority(risk string) string {
	switch risk {
	case RiskLevelCritical:
		return AssignmentPriorityCritical
	case RiskLevelHigh:
		return AssignmentPriorityHigh
	case RiskLevelMedium:
		return AssignmentPriorityNormal
	default:
		return AssignmentPriorityLow
	}
}
