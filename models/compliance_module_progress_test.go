package models

import (
	"testing"
)

func TestCompModStatusConstants(t *testing.T) {
	if CompModStatusPending != "pending" {
		t.Errorf("expected 'pending', got %q", CompModStatusPending)
	}
	if CompModStatusInProgress != "in_progress" {
		t.Errorf("expected 'in_progress', got %q", CompModStatusInProgress)
	}
	if CompModStatusCompleted != "completed" {
		t.Errorf("expected 'completed', got %q", CompModStatusCompleted)
	}
	if CompModStatusFailed != "failed" {
		t.Errorf("expected 'failed', got %q", CompModStatusFailed)
	}
}

func TestComplianceModuleProgressTableName(t *testing.T) {
	p := ComplianceModuleProgress{}
	if p.TableName() != "compliance_module_progress" {
		t.Errorf("expected 'compliance_module_progress', got %q", p.TableName())
	}
}

func TestComplianceModuleAssignmentTableName(t *testing.T) {
	a := ComplianceModuleAssignment{}
	if a.TableName() != "compliance_module_assignments" {
		t.Errorf("expected 'compliance_module_assignments', got %q", a.TableName())
	}
}

func TestComplianceModuleProgressStruct(t *testing.T) {
	p := ComplianceModuleProgress{
		UserId:     1,
		OrgId:      1,
		ModuleSlug: "nis2-awareness-obligations",
		Status:     CompModStatusCompleted,
		QuizScore:  90,
		Passed:     true,
	}
	if p.ModuleSlug != "nis2-awareness-obligations" {
		t.Errorf("expected slug 'nis2-awareness-obligations', got %q", p.ModuleSlug)
	}
	if !p.Passed {
		t.Error("expected Passed to be true")
	}
}

func TestComplianceModuleAssignmentStruct(t *testing.T) {
	a := ComplianceModuleAssignment{
		OrgId:      1,
		ModuleSlug: "gdpr-data-protection",
		UserId:     42,
		IsRequired: true,
	}
	if a.ModuleSlug != "gdpr-data-protection" {
		t.Errorf("expected slug 'gdpr-data-protection', got %q", a.ModuleSlug)
	}
	if !a.IsRequired {
		t.Error("expected IsRequired to be true")
	}
}

func TestComplianceOrgStatsStruct(t *testing.T) {
	s := ComplianceOrgStats{
		TotalModulesAvailable: 10,
		TotalAssignments:      50,
		CompletedCount:        30,
		PassRate:              85.5,
		ByFramework:           make(map[string]FrameworkProgress),
	}
	if s.TotalModulesAvailable != 10 {
		t.Errorf("expected 10 total modules, got %d", s.TotalModulesAvailable)
	}
	if s.PassRate != 85.5 {
		t.Errorf("expected pass rate 85.5, got %f", s.PassRate)
	}
}

func TestFrameworkProgressStruct(t *testing.T) {
	fp := FrameworkProgress{
		FrameworkSlug:  "nis2",
		ModuleCount:    3,
		CompletedCount: 2,
		PassRate:       90.0,
		AvgScore:       88.5,
	}
	if fp.FrameworkSlug != "nis2" {
		t.Errorf("expected 'nis2', got %q", fp.FrameworkSlug)
	}
	if fp.ModuleCount != 3 {
		t.Errorf("expected 3, got %d", fp.ModuleCount)
	}
}

func TestQueryWhereUserModuleSlug(t *testing.T) {
	if queryWhereUserModuleSlug != "user_id = ? AND module_slug = ?" {
		t.Errorf("unexpected query constant: %q", queryWhereUserModuleSlug)
	}
}
