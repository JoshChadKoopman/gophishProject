package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Compliance Module Progress & Assignments (DB-backed) ────────
// Provides persistent tracking for compliance training module progress,
// assignments, and org-level analytics. Works alongside the in-memory
// BuiltInComplianceModules defined in compliance_training_modules.go.

// ComplianceModuleProgress tracks a user's progress through a compliance module.
type ComplianceModuleProgress struct {
	Id            int64     `json:"id" gorm:"primary_key"`
	UserId        int64     `json:"user_id" gorm:"column:user_id"`
	OrgId         int64     `json:"org_id" gorm:"column:org_id"`
	ModuleSlug    string    `json:"module_slug" gorm:"column:module_slug"`
	Status        string    `json:"status" gorm:"column:status"` // pending, in_progress, completed, failed
	CurrentPage   int       `json:"current_page" gorm:"column:current_page"`
	QuizScore     int       `json:"quiz_score" gorm:"column:quiz_score"`
	Passed        bool      `json:"passed" gorm:"column:passed"`
	AttemptsCount int       `json:"attempts_count" gorm:"column:attempts_count"`
	TimeSpentSecs int       `json:"time_spent_secs" gorm:"column:time_spent_secs"`
	StartedDate   time.Time `json:"started_date,omitempty" gorm:"column:started_date"`
	CompletedDate time.Time `json:"completed_date,omitempty" gorm:"column:completed_date"`
	CreatedDate   time.Time `json:"created_date" gorm:"column:created_date"`
}

// ComplianceModuleAssignment assigns a compliance module to a user or group.
type ComplianceModuleAssignment struct {
	Id          int64     `json:"id" gorm:"primary_key"`
	OrgId       int64     `json:"org_id" gorm:"column:org_id"`
	ModuleSlug  string    `json:"module_slug" gorm:"column:module_slug"`
	UserId      int64     `json:"user_id,omitempty" gorm:"column:user_id"`
	GroupId     int64     `json:"group_id,omitempty" gorm:"column:group_id"`
	AssignedBy  int64     `json:"assigned_by" gorm:"column:assigned_by"`
	DueDate     time.Time `json:"due_date" gorm:"column:due_date"`
	IsRequired  bool      `json:"is_required" gorm:"column:is_required;default:true"`
	CreatedDate time.Time `json:"created_date" gorm:"column:created_date"`
}

// ComplianceOrgStats holds org-level compliance training analytics.
type ComplianceOrgStats struct {
	TotalModulesAvailable int                          `json:"total_modules_available"`
	TotalAssignments      int                          `json:"total_assignments"`
	CompletedCount        int                          `json:"completed_count"`
	InProgressCount       int                          `json:"in_progress_count"`
	PassRate              float64                      `json:"pass_rate"`
	AvgQuizScore          float64                      `json:"avg_quiz_score"`
	ByFramework           map[string]FrameworkProgress  `json:"by_framework"`
}

// FrameworkProgress tracks per-framework training progress.
type FrameworkProgress struct {
	FrameworkSlug  string  `json:"framework_slug"`
	ModuleCount    int     `json:"module_count"`
	CompletedCount int     `json:"completed_count"`
	PassRate       float64 `json:"pass_rate"`
	AvgScore       float64 `json:"avg_score"`
}

// Compliance module progress status constants.
const (
	CompModStatusPending    = "pending"
	CompModStatusInProgress = "in_progress"
	CompModStatusCompleted  = "completed"
	CompModStatusFailed     = "failed"
)

// Shared query constants for compliance progress.
const (
	queryWhereUserModuleSlug = "user_id = ? AND module_slug = ?"
)

// Table names.
func (ComplianceModuleProgress) TableName() string   { return "compliance_module_progress" }
func (ComplianceModuleAssignment) TableName() string { return "compliance_module_assignments" }

// GetComplianceModuleProgressForUser returns a user's progress on a specific module.
func GetComplianceModuleProgressForUser(userId int64, moduleSlug string) (*ComplianceModuleProgress, error) {
	p := &ComplianceModuleProgress{}
	err := db.Where(queryWhereUserModuleSlug, userId, moduleSlug).First(p).Error
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetUserComplianceProgress returns all compliance module progress for a user.
func GetUserComplianceProgress(userId int64) ([]ComplianceModuleProgress, error) {
	progress := []ComplianceModuleProgress{}
	err := db.Where("user_id = ?", userId).Order("created_date desc").Find(&progress).Error
	return progress, err
}

// SaveComplianceModuleProgress upserts a user's module progress.
func SaveComplianceModuleProgress(p *ComplianceModuleProgress) error {
	existing := &ComplianceModuleProgress{}
	err := db.Where(queryWhereUserModuleSlug, p.UserId, p.ModuleSlug).First(existing).Error
	if err == nil {
		p.Id = existing.Id
		p.CreatedDate = existing.CreatedDate
	} else {
		p.CreatedDate = time.Now().UTC()
	}
	return db.Save(p).Error
}

// AssignComplianceModule creates a compliance module assignment.
func AssignComplianceModule(a *ComplianceModuleAssignment) error {
	a.CreatedDate = time.Now().UTC()
	return db.Save(a).Error
}

// GetOrgComplianceModuleAssignments returns compliance module assignments for an org.
func GetOrgComplianceModuleAssignments(orgId int64) ([]ComplianceModuleAssignment, error) {
	assignments := []ComplianceModuleAssignment{}
	err := db.Where("org_id = ?", orgId).Order("created_date desc").Find(&assignments).Error
	return assignments, err
}

// GetComplianceOrgStats builds org-level compliance training analytics.
func GetComplianceOrgStats(orgId int64) (*ComplianceOrgStats, error) {
	stats := &ComplianceOrgStats{
		TotalModulesAvailable: len(BuiltInComplianceModules),
		ByFramework:           make(map[string]FrameworkProgress),
	}

	// Count assignments
	var assignedCount int
	db.Model(&ComplianceModuleAssignment{}).Where("org_id = ?", orgId).Count(&assignedCount)
	stats.TotalAssignments = assignedCount

	// Count completed
	var completedCount int
	db.Model(&ComplianceModuleProgress{}).
		Where("org_id = ? AND status = ?", orgId, CompModStatusCompleted).
		Count(&completedCount)
	stats.CompletedCount = completedCount

	// Count in progress
	var inProgressCount int
	db.Model(&ComplianceModuleProgress{}).
		Where("org_id = ? AND status = ?", orgId, CompModStatusInProgress).
		Count(&inProgressCount)
	stats.InProgressCount = inProgressCount

	// Average score and pass rate
	type scoreRow struct {
		AvgScore float64
		PassRate float64
	}
	var sr scoreRow
	db.Model(&ComplianceModuleProgress{}).
		Select("COALESCE(AVG(quiz_score),0) as avg_score, "+
			"COALESCE(AVG(CASE WHEN passed=1 THEN 100.0 ELSE 0.0 END),0) as pass_rate").
		Where("org_id = ? AND status IN (?)", orgId, []string{CompModStatusCompleted, CompModStatusFailed}).
		Scan(&sr)
	stats.AvgQuizScore = sr.AvgScore
	stats.PassRate = sr.PassRate

	// Per-framework stats
	for fw, modules := range complianceModulesByFramework {
		fp := FrameworkProgress{
			FrameworkSlug: fw,
			ModuleCount:   len(modules),
		}
		slugs := make([]string, len(modules))
		for i, m := range modules {
			slugs[i] = m.Slug
		}
		var cc int
		db.Model(&ComplianceModuleProgress{}).
			Where("org_id = ? AND module_slug IN (?) AND status = ?", orgId, slugs, CompModStatusCompleted).
			Count(&cc)
		fp.CompletedCount = cc

		var fwSr scoreRow
		db.Model(&ComplianceModuleProgress{}).
			Select("COALESCE(AVG(quiz_score),0) as avg_score, "+
				"COALESCE(AVG(CASE WHEN passed=1 THEN 100.0 ELSE 0.0 END),0) as pass_rate").
			Where("org_id = ? AND module_slug IN (?) AND status IN (?)",
				orgId, slugs, []string{CompModStatusCompleted, CompModStatusFailed}).
			Scan(&fwSr)
		fp.AvgScore = fwSr.AvgScore
		fp.PassRate = fwSr.PassRate

		stats.ByFramework[fw] = fp
	}

	return stats, nil
}

// SeedComplianceModuleAssignmentsForOrg creates assignments for all built-in
// compliance modules matching the org's enabled frameworks.
func SeedComplianceModuleAssignmentsForOrg(orgId, assignedBy int64, dueDate time.Time) (int, error) {
	// Get enabled frameworks for this org
	frameworks, err := GetOrgFrameworks(orgId)
	if err != nil {
		return 0, err
	}
	enabledSlugs := make(map[string]bool)
	for _, f := range frameworks {
		enabledSlugs[f.Slug] = true
	}

	seeded := 0
	for _, m := range BuiltInComplianceModules {
		if !enabledSlugs[m.FrameworkSlug] {
			continue
		}
		// Check if already assigned
		var count int
		db.Model(&ComplianceModuleAssignment{}).
			Where("org_id = ? AND module_slug = ?", orgId, m.Slug).
			Count(&count)
		if count > 0 {
			continue
		}
		a := ComplianceModuleAssignment{
			OrgId:      orgId,
			ModuleSlug: m.Slug,
			AssignedBy: assignedBy,
			DueDate:    dueDate,
			IsRequired: true,
		}
		if err := AssignComplianceModule(&a); err != nil {
			log.Errorf("Failed to seed compliance module assignment %s for org %d: %v", m.Slug, orgId, err)
			continue
		}
		seeded++
	}
	return seeded, nil
}
