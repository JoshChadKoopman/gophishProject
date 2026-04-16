package models

import (
	"testing"
)

func TestScheduledReportValidate_Valid(t *testing.T) {
	sr := ScheduledReport{
		Name:       "Weekly Summary",
		ReportType: ReportTypeCampaigns,
		Format:     "pdf",
		Frequency:  FrequencyWeekly,
		Hour:       8,
		Minute:     0,
		DayOfWeek:  1,
		DayOfMonth: 1,
		Recipients: "admin@example.com",
		Timezone:   "UTC",
	}
	if msg := sr.Validate(); msg != "" {
		t.Errorf("Expected valid, got: %s", msg)
	}
}

func TestScheduledReportValidate_MissingName(t *testing.T) {
	sr := ScheduledReport{
		ReportType: ReportTypeCampaigns,
		Format:     "pdf",
		Frequency:  FrequencyWeekly,
		Recipients: "a@b.com",
		DayOfWeek:  1,
		DayOfMonth: 1,
	}
	if msg := sr.Validate(); msg != "name is required" {
		t.Errorf("Expected 'name is required', got: %s", msg)
	}
}

func TestScheduledReportValidate_InvalidType(t *testing.T) {
	sr := ScheduledReport{
		Name:       "Test",
		ReportType: "invalid_type",
		Format:     "pdf",
		Frequency:  FrequencyWeekly,
		Recipients: "a@b.com",
		DayOfWeek:  1,
		DayOfMonth: 1,
	}
	if msg := sr.Validate(); msg != "invalid report_type" {
		t.Errorf("Expected 'invalid report_type', got: %s", msg)
	}
}

func TestScheduledReportValidate_InvalidFormat(t *testing.T) {
	sr := ScheduledReport{
		Name:       "Test",
		ReportType: ReportTypeROI,
		Format:     "html",
		Frequency:  FrequencyMonthly,
		Recipients: "a@b.com",
		DayOfWeek:  1,
		DayOfMonth: 1,
	}
	if msg := sr.Validate(); msg != "invalid format (must be pdf, xlsx, or csv)" {
		t.Errorf("Expected format error, got: %s", msg)
	}
}

func TestScheduledReportValidate_InvalidHour(t *testing.T) {
	sr := ScheduledReport{
		Name:       "Test",
		ReportType: ReportTypeTraining,
		Format:     "csv",
		Frequency:  FrequencyDaily,
		Hour:       25,
		Recipients: "a@b.com",
		DayOfWeek:  1,
		DayOfMonth: 1,
	}
	if msg := sr.Validate(); msg != "hour must be 0-23" {
		t.Errorf("Expected hour error, got: %s", msg)
	}
}

func TestScheduledReportRecipientList(t *testing.T) {
	sr := ScheduledReport{
		Recipients: " admin@test.com , bob@test.com , , carol@test.com ",
	}
	list := sr.RecipientList()
	if len(list) != 3 {
		t.Errorf("Expected 3 recipients, got %d: %v", len(list), list)
	}
	if list[0] != "admin@test.com" {
		t.Errorf("Expected admin@test.com, got %s", list[0])
	}
}

func TestScheduledReportFiltersRoundTrip(t *testing.T) {
	sr := ScheduledReport{}
	f := ScheduledReportFilters{
		PeriodDays:  30,
		GroupIds:    []int64{1, 2, 3},
		CampaignIds: []int64{10},
	}
	sr.SetFilters(f)
	parsed := sr.ParseFilters()
	if parsed.PeriodDays != 30 {
		t.Errorf("Expected PeriodDays=30, got %d", parsed.PeriodDays)
	}
	if len(parsed.GroupIds) != 3 {
		t.Errorf("Expected 3 GroupIds, got %d", len(parsed.GroupIds))
	}
}

func TestValidReportTypesMap(t *testing.T) {
	expected := []string{
		ReportTypeExecutiveSummary, ReportTypeCampaigns, ReportTypeTraining,
		ReportTypePhishingTickets, ReportTypeEmailSecurity, ReportTypeNetworkEvents,
		ReportTypeROI, ReportTypeCompliance, ReportTypeHygiene, ReportTypeRiskScores,
	}
	for _, rt := range expected {
		if !ValidReportTypes[rt] {
			t.Errorf("Expected %s to be valid", rt)
		}
	}
	if ValidReportTypes["nonexistent"] {
		t.Error("Expected 'nonexistent' to be invalid")
	}
}

func TestValidFrequenciesMap(t *testing.T) {
	for _, f := range []string{FrequencyDaily, FrequencyWeekly, FrequencyBiweekly, FrequencyMonthly, FrequencyQuarterly} {
		if !ValidFrequencies[f] {
			t.Errorf("Expected %s to be valid frequency", f)
		}
	}
}
