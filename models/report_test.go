package models

import (
	"testing"
)

func TestReportOverviewDefaults(t *testing.T) {
	o := ReportOverview{}
	if o.TotalCampaigns != 0 || o.ActiveCampaigns != 0 || o.TotalRecipients != 0 {
		t.Error("default ReportOverview should have zero campaign counts")
	}
	if o.AvgClickRate != 0 || o.AvgSubmitRate != 0 || o.AvgReportRate != 0 {
		t.Error("default ReportOverview should have zero rates")
	}
}

func TestTrendPointDefaults(t *testing.T) {
	tp := TrendPoint{}
	if tp.Sent != 0 || tp.Opened != 0 || tp.Clicked != 0 || tp.SubmittedData != 0 || tp.Reported != 0 {
		t.Error("default TrendPoint should have zero counts")
	}
	if tp.ClickRate != 0 {
		t.Error("default TrendPoint should have zero click rate")
	}
	if tp.Date != "" {
		t.Error("default TrendPoint should have empty date")
	}
}

func TestUserRiskScoreDefaults(t *testing.T) {
	s := UserRiskScore{}
	if s.RiskScore != 0 || s.Total != 0 || s.Clicked != 0 || s.Submitted != 0 || s.Reported != 0 {
		t.Error("default UserRiskScore should have zero values")
	}
	if s.Email != "" || s.FirstName != "" || s.LastName != "" {
		t.Error("default UserRiskScore should have empty strings")
	}
}

func TestTrainingSummaryDefaults(t *testing.T) {
	s := TrainingSummary{}
	if s.TotalCourses != 0 || s.TotalAssignments != 0 || s.CompletedCount != 0 {
		t.Error("default TrainingSummary should have zero counts")
	}
	if s.CompletionRate != 0 || s.AvgQuizScore != 0 {
		t.Error("default TrainingSummary should have zero rates")
	}
	if s.CertificatesIssued != 0 || s.OverdueCount != 0 {
		t.Error("default TrainingSummary should have zero certificates and overdue")
	}
}

func TestGroupComparisonDefaults(t *testing.T) {
	gc := GroupComparison{}
	if gc.GroupId != 0 || gc.GroupName != "" {
		t.Error("default GroupComparison should have zero id and empty name")
	}
	if gc.ClickRate != 0 || gc.SubmitRate != 0 {
		t.Error("default GroupComparison should have zero rates")
	}
}
