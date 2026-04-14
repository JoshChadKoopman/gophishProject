package models

import (
	"testing"
)

func TestBuiltInComplianceModulesNotEmpty(t *testing.T) {
	if len(BuiltInComplianceModules) == 0 {
		t.Fatal("BuiltInComplianceModules should not be empty")
	}
}

func TestComplianceModuleUniqueSlugs(t *testing.T) {
	seen := make(map[string]bool)
	for _, m := range BuiltInComplianceModules {
		if seen[m.Slug] {
			t.Errorf("duplicate compliance module slug: %s", m.Slug)
		}
		seen[m.Slug] = true
	}
}

func TestComplianceModuleRequiredFields(t *testing.T) {
	for _, m := range BuiltInComplianceModules {
		if m.Slug == "" {
			t.Error("module has empty slug")
		}
		if m.FrameworkSlug == "" {
			t.Errorf("module %s has empty framework slug", m.Slug)
		}
		if m.Title == "" {
			t.Errorf("module %s has empty title", m.Slug)
		}
		if m.Description == "" {
			t.Errorf("module %s has empty description", m.Slug)
		}
		if len(m.ControlRefs) == 0 {
			t.Errorf("module %s has no control references", m.Slug)
		}
		if m.EstimatedMinutes <= 0 {
			t.Errorf("module %s has invalid estimated minutes: %d", m.Slug, m.EstimatedMinutes)
		}
		if len(m.Pages) == 0 {
			t.Errorf("module %s has no pages", m.Slug)
		}
	}
}

func TestComplianceModulePagesHaveContent(t *testing.T) {
	for _, m := range BuiltInComplianceModules {
		for i, p := range m.Pages {
			if p.Title == "" {
				t.Errorf("module %s page %d has empty title", m.Slug, i)
			}
			if p.Body == "" {
				t.Errorf("module %s page %d has empty body", m.Slug, i)
			}
		}
	}
}

func TestComplianceModuleQuizzes(t *testing.T) {
	for _, m := range BuiltInComplianceModules {
		if m.Quiz == nil {
			continue // Quiz is optional
		}
		if m.Quiz.PassPercentage <= 0 || m.Quiz.PassPercentage > 100 {
			t.Errorf("module %s has invalid pass percentage: %d", m.Slug, m.Quiz.PassPercentage)
		}
		if len(m.Quiz.Questions) == 0 {
			t.Errorf("module %s has quiz with no questions", m.Slug)
		}
		for j, q := range m.Quiz.Questions {
			if q.QuestionText == "" {
				t.Errorf("module %s question %d has empty text", m.Slug, j)
			}
			if len(q.Options) < 2 {
				t.Errorf("module %s question %d has fewer than 2 options", m.Slug, j)
			}
			if q.CorrectOption < 0 || q.CorrectOption >= len(q.Options) {
				t.Errorf("module %s question %d has out-of-range correct option: %d", m.Slug, j, q.CorrectOption)
			}
		}
	}
}

func TestGetComplianceTrainingModules(t *testing.T) {
	modules := GetComplianceTrainingModules()
	if len(modules) != len(BuiltInComplianceModules) {
		t.Errorf("expected %d modules, got %d", len(BuiltInComplianceModules), len(modules))
	}
}

func TestGetComplianceTrainingModuleFound(t *testing.T) {
	slug := BuiltInComplianceModules[0].Slug
	m := GetComplianceTrainingModule(slug)
	if m == nil {
		t.Fatalf("expected to find module %s", slug)
	}
	if m.Slug != slug {
		t.Errorf("expected slug %s, got %s", slug, m.Slug)
	}
}

func TestGetComplianceTrainingModuleNotFound(t *testing.T) {
	m := GetComplianceTrainingModule("nonexistent-module-xyz")
	if m != nil {
		t.Error("expected nil for nonexistent module")
	}
}

func TestGetComplianceModulesForFramework(t *testing.T) {
	framework := BuiltInComplianceModules[0].FrameworkSlug
	modules := GetComplianceModulesForFramework(framework)
	if len(modules) == 0 {
		t.Errorf("expected modules for framework %s", framework)
	}
	for _, m := range modules {
		if m.FrameworkSlug != framework {
			t.Errorf("expected framework %s, got %s", framework, m.FrameworkSlug)
		}
	}
}

func TestGetComplianceModulesForFrameworkNotFound(t *testing.T) {
	modules := GetComplianceModulesForFramework("nonexistent-framework")
	if len(modules) != 0 {
		t.Errorf("expected 0 modules for nonexistent framework, got %d", len(modules))
	}
}

func TestGetComplianceModuleSummaries(t *testing.T) {
	summaries := GetComplianceModuleSummaries()
	if len(summaries) != len(BuiltInComplianceModules) {
		t.Errorf("expected %d summaries, got %d", len(BuiltInComplianceModules), len(summaries))
	}
	for i, s := range summaries {
		m := BuiltInComplianceModules[i]
		if s.Slug != m.Slug {
			t.Errorf("summary %d: expected slug %s, got %s", i, m.Slug, s.Slug)
		}
		if s.PageCount != len(m.Pages) {
			t.Errorf("summary %s: expected %d pages, got %d", s.Slug, len(m.Pages), s.PageCount)
		}
		if m.Quiz != nil {
			if !s.HasQuiz {
				t.Errorf("summary %s: expected HasQuiz true", s.Slug)
			}
			if s.QuestionCount != len(m.Quiz.Questions) {
				t.Errorf("summary %s: expected %d questions, got %d", s.Slug, len(m.Quiz.Questions), s.QuestionCount)
			}
		}
	}
}

func TestComplianceModuleFrameworkCoverage(t *testing.T) {
	// Verify we have modules for expected frameworks
	expectedFrameworks := []string{"nis2", "dora", "hipaa", "pci_dss", "nist_csf", "iso27001", "gdpr", "soc2"}
	for _, fw := range expectedFrameworks {
		modules := GetComplianceModulesForFramework(fw)
		if len(modules) == 0 {
			t.Errorf("no compliance modules found for framework %s", fw)
		}
	}
}
