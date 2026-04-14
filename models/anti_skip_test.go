package models

import (
	"testing"
)

// =====================================================================
// UNIT TESTS — pure logic, no database
// =====================================================================

func TestDefaultAntiSkipPolicy(t *testing.T) {
	t.Run("returns correct defaults", func(t *testing.T) {
		policy := defaultAntiSkipPolicy(99)
		if policy.PresentationId != 99 {
			t.Errorf("Expected PresentationId=99, got %d", policy.PresentationId)
		}
		if policy.MinDwellSeconds != DefaultMinDwellSeconds {
			t.Errorf("Expected MinDwellSeconds=%d, got %d", DefaultMinDwellSeconds, policy.MinDwellSeconds)
		}
		if policy.EnforceSequential != DefaultEnforceSequential {
			t.Errorf("Expected EnforceSequential=%v, got %v", DefaultEnforceSequential, policy.EnforceSequential)
		}
		if policy.AllowBackNavigation != DefaultAllowBack {
			t.Errorf("Expected AllowBackNavigation=%v, got %v", DefaultAllowBack, policy.AllowBackNavigation)
		}
		if policy.RequireAcknowledge {
			t.Error("Expected RequireAcknowledge=false by default")
		}
		if policy.RequireScroll {
			t.Error("Expected RequireScroll=false by default")
		}
		if policy.MinScrollDepthPct != DefaultMinScrollDepth {
			t.Errorf("Expected MinScrollDepthPct=%d, got %d", DefaultMinScrollDepth, policy.MinScrollDepthPct)
		}
	})

	t.Run("different presentation IDs", func(t *testing.T) {
		for _, id := range []int64{0, 1, 999, 1000000} {
			p := defaultAntiSkipPolicy(id)
			if p.PresentationId != id {
				t.Errorf("Expected PresentationId=%d, got %d", id, p.PresentationId)
			}
		}
	})
}

func TestCheckEngagementRecordDwellTime(t *testing.T) {
	policy := AntiSkipPolicy{MinDwellSeconds: 10, RequireAcknowledge: false, RequireScroll: false}

	tests := []struct {
		name     string
		dwell    int
		wantFail bool
	}{
		{"zero dwell", 0, true},
		{"insufficient dwell", 5, true},
		{"exactly at threshold", 10, false},
		{"above threshold", 30, false},
		{"just below threshold", 9, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := PageEngagement{DwellSeconds: tt.dwell}
			reason := checkEngagementRecord(eng, policy)
			if tt.wantFail && reason == "" {
				t.Errorf("Expected failure for dwell=%d", tt.dwell)
			}
			if !tt.wantFail && reason != "" {
				t.Errorf("Expected pass for dwell=%d, got: %s", tt.dwell, reason)
			}
		})
	}
}

func TestCheckEngagementRecordAcknowledge(t *testing.T) {
	policy := AntiSkipPolicy{MinDwellSeconds: 0, RequireAcknowledge: true, RequireScroll: false}

	t.Run("not acknowledged", func(t *testing.T) {
		eng := PageEngagement{DwellSeconds: 100, Acknowledged: false}
		reason := checkEngagementRecord(eng, policy)
		if reason == "" {
			t.Error("Expected failure for missing acknowledgement")
		}
		if reason != ErrAckRequired.Error() {
			t.Errorf("Expected ErrAckRequired message, got: %s", reason)
		}
	})

	t.Run("acknowledged", func(t *testing.T) {
		eng := PageEngagement{DwellSeconds: 100, Acknowledged: true}
		reason := checkEngagementRecord(eng, policy)
		if reason != "" {
			t.Errorf("Expected pass with acknowledgement, got: %s", reason)
		}
	})

	t.Run("ack not required passes without ack", func(t *testing.T) {
		noAckPolicy := AntiSkipPolicy{MinDwellSeconds: 0, RequireAcknowledge: false}
		eng := PageEngagement{Acknowledged: false}
		reason := checkEngagementRecord(eng, noAckPolicy)
		if reason != "" {
			t.Errorf("Expected pass when ack not required, got: %s", reason)
		}
	})
}

func TestCheckEngagementRecordScrollDepth(t *testing.T) {
	policy := AntiSkipPolicy{MinDwellSeconds: 0, RequireAcknowledge: false, RequireScroll: true, MinScrollDepthPct: 80}

	tests := []struct {
		name     string
		scroll   int
		wantFail bool
	}{
		{"zero scroll", 0, true},
		{"insufficient scroll", 50, true},
		{"just below threshold", 79, true},
		{"at threshold", 80, false},
		{"above threshold", 95, false},
		{"full scroll", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := PageEngagement{DwellSeconds: 100, ScrollDepthPct: tt.scroll}
			reason := checkEngagementRecord(eng, policy)
			if tt.wantFail && reason == "" {
				t.Errorf("Expected failure for scroll=%d", tt.scroll)
			}
			if !tt.wantFail && reason != "" {
				t.Errorf("Expected pass for scroll=%d, got: %s", tt.scroll, reason)
			}
		})
	}

	t.Run("scroll not required passes at zero", func(t *testing.T) {
		noScrollPolicy := AntiSkipPolicy{MinDwellSeconds: 0, RequireScroll: false}
		eng := PageEngagement{ScrollDepthPct: 0}
		reason := checkEngagementRecord(eng, noScrollPolicy)
		if reason != "" {
			t.Errorf("Expected pass when scroll not required, got: %s", reason)
		}
	})
}

func TestCheckEngagementRecordAllRequirements(t *testing.T) {
	policy := AntiSkipPolicy{
		MinDwellSeconds:    15,
		RequireAcknowledge: true,
		RequireScroll:      true,
		MinScrollDepthPct:  90,
	}

	tests := []struct {
		name     string
		eng      PageEngagement
		wantFail bool
	}{
		{"all insufficient", PageEngagement{DwellSeconds: 5, ScrollDepthPct: 10, Acknowledged: false}, true},
		{"dwell ok, rest not", PageEngagement{DwellSeconds: 20, ScrollDepthPct: 10, Acknowledged: false}, true},
		{"dwell+ack ok, scroll not", PageEngagement{DwellSeconds: 20, ScrollDepthPct: 10, Acknowledged: true}, true},
		{"dwell+scroll ok, ack not", PageEngagement{DwellSeconds: 20, ScrollDepthPct: 95, Acknowledged: false}, true},
		{"all ok exactly at thresholds", PageEngagement{DwellSeconds: 15, ScrollDepthPct: 90, Acknowledged: true}, false},
		{"all ok above thresholds", PageEngagement{DwellSeconds: 60, ScrollDepthPct: 100, Acknowledged: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := checkEngagementRecord(tt.eng, policy)
			if tt.wantFail && reason == "" {
				t.Error("Expected failure, got pass")
			}
			if !tt.wantFail && reason != "" {
				t.Errorf("Expected pass, got: %s", reason)
			}
		})
	}
}

func TestCheckEngagementRecordNoDwellRequired(t *testing.T) {
	t.Run("zero dwell policy passes zero engagement", func(t *testing.T) {
		policy := AntiSkipPolicy{MinDwellSeconds: 0, RequireAcknowledge: false, RequireScroll: false}
		eng := PageEngagement{DwellSeconds: 0}
		reason := checkEngagementRecord(eng, policy)
		if reason != "" {
			t.Errorf("Expected pass with zero dwell requirement, got: %s", reason)
		}
	})
}

func TestCheckEngagementRecordPriorityOrder(t *testing.T) {
	t.Run("dwell failure takes priority over ack", func(t *testing.T) {
		policy := AntiSkipPolicy{MinDwellSeconds: 10, RequireAcknowledge: true}
		eng := PageEngagement{DwellSeconds: 5, Acknowledged: false}
		reason := checkEngagementRecord(eng, policy)
		if reason == "" {
			t.Error("Expected failure")
		}
		// Dwell check happens first
		if reason == ErrAckRequired.Error() {
			t.Error("Expected dwell failure to come before ack failure")
		}
	})

	t.Run("ack failure takes priority over scroll", func(t *testing.T) {
		policy := AntiSkipPolicy{MinDwellSeconds: 0, RequireAcknowledge: true, RequireScroll: true, MinScrollDepthPct: 80}
		eng := PageEngagement{DwellSeconds: 100, Acknowledged: false, ScrollDepthPct: 0}
		reason := checkEngagementRecord(eng, policy)
		if reason != ErrAckRequired.Error() {
			t.Errorf("Expected ack failure before scroll failure, got: %s", reason)
		}
	})
}

func TestPageAdvanceResultStruct(t *testing.T) {
	t.Run("allowed result", func(t *testing.T) {
		r := PageAdvanceResult{Allowed: true, NextPage: 2, PagesUnlocked: 3}
		if !r.Allowed {
			t.Error("Expected allowed=true")
		}
		if r.NextPage != 2 {
			t.Errorf("Expected NextPage=2, got %d", r.NextPage)
		}
	})

	t.Run("denied result has reason", func(t *testing.T) {
		r := PageAdvanceResult{Allowed: false, Reason: "test reason"}
		if r.Allowed {
			t.Error("Expected allowed=false")
		}
		if r.Reason != "test reason" {
			t.Errorf("Expected reason='test reason', got %q", r.Reason)
		}
	})
}

func TestCompletionGateResultAllPagesViewed(t *testing.T) {
	t.Run("all pages engaged", func(t *testing.T) {
		result := CompletionGateResult{
			TotalPages:   5,
			EngagedPages: 5,
			Allowed:      true,
		}
		if !result.Allowed {
			t.Error("Expected allowed when all pages engaged")
		}
		if len(result.MissingPages) != 0 {
			t.Errorf("Expected no missing pages, got %d", len(result.MissingPages))
		}
	})
}

func TestCompletionGateResultMissingPages(t *testing.T) {
	t.Run("pages missing", func(t *testing.T) {
		result := CompletionGateResult{
			TotalPages:   5,
			EngagedPages: 3,
			MissingPages: []int{2, 4},
			Allowed:      false,
			Reason:       "Insufficient engagement on 2 of 5 pages",
		}
		if result.Allowed {
			t.Error("Expected not allowed when pages are missing")
		}
		if len(result.MissingPages) != 2 {
			t.Errorf("Expected 2 missing pages, got %d", len(result.MissingPages))
		}
		if result.MissingPages[0] != 2 || result.MissingPages[1] != 4 {
			t.Errorf("Expected missing pages [2,4], got %v", result.MissingPages)
		}
	})
}

func TestPageEngagementUpdateStruct(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		update := PageEngagementUpdate{
			PageIndex:      0,
			DwellSeconds:   15,
			ScrollDepthPct: 100,
			Acknowledged:   true,
		}
		if update.DwellSeconds != 15 {
			t.Errorf("Expected DwellSeconds=15, got %d", update.DwellSeconds)
		}
		if update.ScrollDepthPct != 100 {
			t.Errorf("Expected ScrollDepthPct=100, got %d", update.ScrollDepthPct)
		}
		if !update.Acknowledged {
			t.Error("Expected Acknowledged=true")
		}
	})

	t.Run("boundary page index", func(t *testing.T) {
		update := PageEngagementUpdate{PageIndex: 0}
		if update.PageIndex != 0 {
			t.Errorf("Expected PageIndex=0, got %d", update.PageIndex)
		}
	})
}

func TestEngagementSummaryRow(t *testing.T) {
	t.Run("fields populated correctly", func(t *testing.T) {
		row := EngagementSummaryRow{
			UserId:            1,
			Username:          "testuser",
			Email:             "test@example.com",
			PagesEngaged:      5,
			TotalDwellSeconds: 120,
			AvgDwellSeconds:   24.0,
			AvgScrollDepth:    85.0,
		}
		if row.PagesEngaged != 5 {
			t.Errorf("Expected PagesEngaged=5, got %d", row.PagesEngaged)
		}
		if row.AvgDwellSeconds != 24.0 {
			t.Errorf("Expected AvgDwellSeconds=24.0, got %f", row.AvgDwellSeconds)
		}
		if row.TotalDwellSeconds != 120 {
			t.Errorf("Expected TotalDwellSeconds=120, got %d", row.TotalDwellSeconds)
		}
	})
}

func TestTableNames(t *testing.T) {
	t.Run("PageEngagement table name", func(t *testing.T) {
		pe := PageEngagement{}
		if pe.TableName() != "page_engagement" {
			t.Errorf("Expected table name 'page_engagement', got %q", pe.TableName())
		}
	})

	t.Run("AntiSkipPolicy table name", func(t *testing.T) {
		asp := AntiSkipPolicy{}
		if asp.TableName() != "anti_skip_policy" {
			t.Errorf("Expected table name 'anti_skip_policy', got %q", asp.TableName())
		}
	})
}

func TestErrorMessages(t *testing.T) {
	t.Run("error values are non-empty", func(t *testing.T) {
		errs := []error{ErrPageSkipped, ErrInsufficientDwell, ErrAckRequired, ErrScrollRequired, ErrNotAllPagesViewed}
		for _, e := range errs {
			if e.Error() == "" {
				t.Error("Expected non-empty error message")
			}
		}
	})
}

// =====================================================================
// FUNCTIONAL TESTS — DB-backed operations against in-memory SQLite
// =====================================================================

func TestSaveAndGetAntiSkipPolicy(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Policy Test", FileName: "p.pdf", FilePath: "/f"}
	if err := PostTrainingPresentation(tp); err != nil {
		t.Fatalf("failed to create presentation: %v", err)
	}

	t.Run("returns default when no custom policy", func(t *testing.T) {
		policy := GetAntiSkipPolicy(tp.Id)
		if policy.MinDwellSeconds != DefaultMinDwellSeconds {
			t.Errorf("Expected default MinDwellSeconds=%d, got %d", DefaultMinDwellSeconds, policy.MinDwellSeconds)
		}
		if policy.PresentationId != tp.Id {
			t.Errorf("Expected PresentationId=%d, got %d", tp.Id, policy.PresentationId)
		}
		if policy.EnforceSequential != DefaultEnforceSequential {
			t.Errorf("Expected default EnforceSequential=%v", DefaultEnforceSequential)
		}
	})

	t.Run("save and retrieve custom policy", func(t *testing.T) {
		custom := &AntiSkipPolicy{
			PresentationId:      tp.Id,
			MinDwellSeconds:     30,
			RequireAcknowledge:  true,
			RequireScroll:       true,
			MinScrollDepthPct:   90,
			EnforceSequential:   false,
			AllowBackNavigation: true,
		}
		if err := SaveAntiSkipPolicy(custom); err != nil {
			t.Fatalf("failed to save policy: %v", err)
		}
		if custom.Id == 0 {
			t.Fatal("Expected non-zero ID after save")
		}

		got := GetAntiSkipPolicy(tp.Id)
		if got.MinDwellSeconds != 30 {
			t.Errorf("Expected MinDwellSeconds=30, got %d", got.MinDwellSeconds)
		}
		if !got.RequireAcknowledge {
			t.Error("Expected RequireAcknowledge=true")
		}
		if !got.RequireScroll {
			t.Error("Expected RequireScroll=true")
		}
		if got.MinScrollDepthPct != 90 {
			t.Errorf("Expected MinScrollDepthPct=90, got %d", got.MinScrollDepthPct)
		}
		if got.EnforceSequential {
			t.Error("Expected EnforceSequential=false")
		}
	})

	t.Run("update existing policy", func(t *testing.T) {
		updated := &AntiSkipPolicy{
			PresentationId:  tp.Id,
			MinDwellSeconds: 60,
		}
		if err := SaveAntiSkipPolicy(updated); err != nil {
			t.Fatalf("failed to update policy: %v", err)
		}

		got := GetAntiSkipPolicy(tp.Id)
		if got.MinDwellSeconds != 60 {
			t.Errorf("Expected MinDwellSeconds=60 after update, got %d", got.MinDwellSeconds)
		}
	})

	t.Run("clamp negative dwell to zero", func(t *testing.T) {
		p := &AntiSkipPolicy{PresentationId: tp.Id, MinDwellSeconds: -5}
		if err := SaveAntiSkipPolicy(p); err != nil {
			t.Fatalf("failed to save: %v", err)
		}
		got := GetAntiSkipPolicy(tp.Id)
		if got.MinDwellSeconds != 0 {
			t.Errorf("Expected MinDwellSeconds clamped to 0, got %d", got.MinDwellSeconds)
		}
	})

	t.Run("clamp invalid scroll depth to default", func(t *testing.T) {
		p := &AntiSkipPolicy{PresentationId: tp.Id, MinScrollDepthPct: 150}
		if err := SaveAntiSkipPolicy(p); err != nil {
			t.Fatalf("failed to save: %v", err)
		}
		got := GetAntiSkipPolicy(tp.Id)
		if got.MinScrollDepthPct != DefaultMinScrollDepth {
			t.Errorf("Expected MinScrollDepthPct clamped to %d, got %d", DefaultMinScrollDepth, got.MinScrollDepthPct)
		}
	})
}

func TestDeleteAntiSkipPolicy(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Delete Policy", FileName: "d.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	t.Run("delete custom policy reverts to defaults", func(t *testing.T) {
		custom := &AntiSkipPolicy{PresentationId: tp.Id, MinDwellSeconds: 99}
		SaveAntiSkipPolicy(custom)

		// Verify custom is saved
		got := GetAntiSkipPolicy(tp.Id)
		if got.MinDwellSeconds != 99 {
			t.Fatalf("Expected 99, got %d", got.MinDwellSeconds)
		}

		// Delete
		if err := DeleteAntiSkipPolicy(tp.Id); err != nil {
			t.Fatalf("failed to delete policy: %v", err)
		}

		// Should be back to defaults
		got = GetAntiSkipPolicy(tp.Id)
		if got.MinDwellSeconds != DefaultMinDwellSeconds {
			t.Errorf("Expected default %d after delete, got %d", DefaultMinDwellSeconds, got.MinDwellSeconds)
		}
	})

	t.Run("delete non-existent policy no error", func(t *testing.T) {
		err := DeleteAntiSkipPolicy(999999)
		if err != nil {
			t.Errorf("Expected no error deleting non-existent policy, got: %v", err)
		}
	})
}

func TestRecordPageEngagement(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Engage Test", FileName: "e.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	var userId int64 = 1

	t.Run("create new engagement", func(t *testing.T) {
		update := PageEngagementUpdate{PageIndex: 0, DwellSeconds: 5, ScrollDepthPct: 30, Acknowledged: false}
		if err := RecordPageEngagement(userId, tp.Id, update); err != nil {
			t.Fatalf("failed to record engagement: %v", err)
		}

		records, err := GetPageEngagements(userId, tp.Id)
		if err != nil {
			t.Fatalf("failed to get engagements: %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("Expected 1 record, got %d", len(records))
		}
		if records[0].DwellSeconds != 5 {
			t.Errorf("Expected DwellSeconds=5, got %d", records[0].DwellSeconds)
		}
		if records[0].ScrollDepthPct != 30 {
			t.Errorf("Expected ScrollDepthPct=30, got %d", records[0].ScrollDepthPct)
		}
		if records[0].InteractionType != "timer" {
			t.Errorf("Expected InteractionType='timer', got %q", records[0].InteractionType)
		}
	})

	t.Run("accumulate dwell time", func(t *testing.T) {
		update := PageEngagementUpdate{PageIndex: 0, DwellSeconds: 10, ScrollDepthPct: 20}
		if err := RecordPageEngagement(userId, tp.Id, update); err != nil {
			t.Fatalf("failed to record engagement: %v", err)
		}

		records, _ := GetPageEngagements(userId, tp.Id)
		if len(records) != 1 {
			t.Fatalf("Expected still 1 record (upsert), got %d", len(records))
		}
		if records[0].DwellSeconds != 15 {
			t.Errorf("Expected accumulated DwellSeconds=15, got %d", records[0].DwellSeconds)
		}
	})

	t.Run("track max scroll depth", func(t *testing.T) {
		records, _ := GetPageEngagements(userId, tp.Id)
		if records[0].ScrollDepthPct != 30 {
			t.Errorf("Expected max ScrollDepthPct=30, got %d", records[0].ScrollDepthPct)
		}

		update := PageEngagementUpdate{PageIndex: 0, DwellSeconds: 0, ScrollDepthPct: 80}
		RecordPageEngagement(userId, tp.Id, update)
		records, _ = GetPageEngagements(userId, tp.Id)
		if records[0].ScrollDepthPct != 80 {
			t.Errorf("Expected max ScrollDepthPct=80, got %d", records[0].ScrollDepthPct)
		}
	})

	t.Run("acknowledge is sticky", func(t *testing.T) {
		update := PageEngagementUpdate{PageIndex: 0, DwellSeconds: 0, Acknowledged: true}
		RecordPageEngagement(userId, tp.Id, update)
		records, _ := GetPageEngagements(userId, tp.Id)
		if !records[0].Acknowledged {
			t.Error("Expected Acknowledged=true")
		}
		if records[0].InteractionType != "acknowledge" {
			t.Errorf("Expected InteractionType='acknowledge', got %q", records[0].InteractionType)
		}

		update = PageEngagementUpdate{PageIndex: 0, DwellSeconds: 1, Acknowledged: false}
		RecordPageEngagement(userId, tp.Id, update)
		records, _ = GetPageEngagements(userId, tp.Id)
		if !records[0].Acknowledged {
			t.Error("Expected Acknowledged to remain true (sticky)")
		}
	})

	t.Run("multiple pages tracked separately", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 1, DwellSeconds: 12})
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 2, DwellSeconds: 8})

		records, _ := GetPageEngagements(userId, tp.Id)
		if len(records) != 3 {
			t.Fatalf("Expected 3 records, got %d", len(records))
		}
		for i, r := range records {
			if r.PageIndex != i {
				t.Errorf("Expected record %d to have PageIndex=%d, got %d", i, i, r.PageIndex)
			}
		}
	})
}

func TestResetPageEngagement(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Reset Test", FileName: "r.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	var userId int64 = 1

	for i := 0; i < 3; i++ {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: i, DwellSeconds: 10})
	}

	records, _ := GetPageEngagements(userId, tp.Id)
	if len(records) != 3 {
		t.Fatalf("Expected 3 records before reset, got %d", len(records))
	}

	t.Run("reset clears all engagements", func(t *testing.T) {
		if err := ResetPageEngagement(userId, tp.Id); err != nil {
			t.Fatalf("failed to reset: %v", err)
		}
		records, _ = GetPageEngagements(userId, tp.Id)
		if len(records) != 0 {
			t.Errorf("Expected 0 records after reset, got %d", len(records))
		}
	})

	t.Run("reset does not affect other users", func(t *testing.T) {
		RecordPageEngagement(1, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 10})
		RecordPageEngagement(2, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 10})

		ResetPageEngagement(1, tp.Id)

		r1, _ := GetPageEngagements(1, tp.Id)
		r2, _ := GetPageEngagements(2, tp.Id)

		if len(r1) != 0 {
			t.Errorf("Expected 0 records for user 1 after reset, got %d", len(r1))
		}
		if len(r2) != 1 {
			t.Errorf("Expected 1 record for user 2 (unaffected), got %d", len(r2))
		}
	})
}

// =====================================================================
// INTEGRATION TESTS — end-to-end flows using DB-backed functions
// =====================================================================

func TestValidatePageAdvanceIntegration(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Advance Test", FileName: "a.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	var userId int64 = 1
	totalPages := 5

	policy := &AntiSkipPolicy{
		PresentationId:      tp.Id,
		MinDwellSeconds:     10,
		RequireAcknowledge:  true,
		RequireScroll:       false,
		EnforceSequential:   true,
		AllowBackNavigation: true,
	}
	SaveAntiSkipPolicy(policy)

	t.Run("denied without engagement on current page", func(t *testing.T) {
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if result.Allowed {
			t.Error("Expected denied without engagement")
		}
		if result.Reason == "" {
			t.Error("Expected a reason for denial")
		}
	})

	t.Run("denied with insufficient dwell", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 5, Acknowledged: true})
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if result.Allowed {
			t.Error("Expected denied with insufficient dwell")
		}
	})

	t.Run("denied with sufficient dwell but no ack", func(t *testing.T) {
		ResetPageEngagement(userId, tp.Id)
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 15, Acknowledged: false})
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if result.Allowed {
			t.Error("Expected denied without acknowledgment")
		}
	})

	t.Run("allowed with sufficient engagement", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 0, Acknowledged: true})
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if !result.Allowed {
			t.Errorf("Expected allowed with sufficient engagement, got reason: %s", result.Reason)
		}
		if result.NextPage != 1 {
			t.Errorf("Expected NextPage=1, got %d", result.NextPage)
		}
	})

	t.Run("denied for skip-forward", func(t *testing.T) {
		result := ValidatePageAdvance(userId, tp.Id, 0, 3, totalPages)
		if result.Allowed {
			t.Error("Expected denied for skip-forward")
		}
		if result.Reason != ErrPageSkipped.Error() {
			t.Errorf("Expected ErrPageSkipped, got: %s", result.Reason)
		}
	})

	t.Run("backward navigation allowed", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 1, DwellSeconds: 15, Acknowledged: true})
		result := ValidatePageAdvance(userId, tp.Id, 1, 0, totalPages)
		if !result.Allowed {
			t.Errorf("Expected backward navigation allowed, got reason: %s", result.Reason)
		}
	})

	t.Run("pages_unlocked reflects engagement progress", func(t *testing.T) {
		result := ValidatePageAdvance(userId, tp.Id, 1, 2, totalPages)
		if result.PagesUnlocked < 1 {
			t.Errorf("Expected PagesUnlocked >= 1, got %d", result.PagesUnlocked)
		}
	})

	t.Run("policy metadata returned in result", func(t *testing.T) {
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if result.MinDwell != 10 {
			t.Errorf("Expected MinDwell=10, got %d", result.MinDwell)
		}
		if !result.RequireAck {
			t.Error("Expected RequireAck=true")
		}
	})
}

func TestValidatePageAdvanceNonSequential(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "NonSeq Test", FileName: "ns.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	var userId int64 = 1

	policy := &AntiSkipPolicy{
		PresentationId:    tp.Id,
		MinDwellSeconds:   0,
		EnforceSequential: false,
	}
	SaveAntiSkipPolicy(policy)

	t.Run("skip forward allowed when not sequential", func(t *testing.T) {
		// Even with non-sequential, current page needs an engagement record
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 0})
		result := ValidatePageAdvance(userId, tp.Id, 0, 4, 5)
		if !result.Allowed {
			t.Errorf("Expected allowed for non-sequential skip, got reason: %s", result.Reason)
		}
	})

	t.Run("all pages unlocked when not sequential", func(t *testing.T) {
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, 5)
		if result.PagesUnlocked != 4 {
			t.Errorf("Expected PagesUnlocked=4 (all), got %d", result.PagesUnlocked)
		}
	})
}

func TestValidateCourseCompletionIntegration(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Complete Test", FileName: "c.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	var userId int64 = 1
	totalPages := 3

	policy := &AntiSkipPolicy{
		PresentationId:  tp.Id,
		MinDwellSeconds: 10,
	}
	SaveAntiSkipPolicy(policy)

	t.Run("denied with no engagements", func(t *testing.T) {
		result := ValidateCourseCompletion(userId, tp.Id, totalPages)
		if result.Allowed {
			t.Error("Expected denied with no engagements")
		}
		if len(result.MissingPages) != totalPages {
			t.Errorf("Expected %d missing pages, got %d", totalPages, len(result.MissingPages))
		}
		if result.EngagedPages != 0 {
			t.Errorf("Expected 0 engaged pages, got %d", result.EngagedPages)
		}
		if result.TotalPages != totalPages {
			t.Errorf("Expected TotalPages=%d, got %d", totalPages, result.TotalPages)
		}
	})

	t.Run("denied with partial engagement", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 15})
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 1, DwellSeconds: 12})

		result := ValidateCourseCompletion(userId, tp.Id, totalPages)
		if result.Allowed {
			t.Error("Expected denied with partial engagement")
		}
		if len(result.MissingPages) != 1 {
			t.Errorf("Expected 1 missing page, got %d", len(result.MissingPages))
		}
		if result.MissingPages[0] != 2 {
			t.Errorf("Expected missing page 2, got %v", result.MissingPages)
		}
		if result.EngagedPages != 2 {
			t.Errorf("Expected 2 engaged pages, got %d", result.EngagedPages)
		}
	})

	t.Run("denied with insufficient dwell on one page", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 2, DwellSeconds: 3})

		result := ValidateCourseCompletion(userId, tp.Id, totalPages)
		if result.Allowed {
			t.Error("Expected denied with insufficient dwell on page 2")
		}
		if len(result.MissingPages) != 1 {
			t.Errorf("Expected 1 missing page, got %d (pages: %v)", len(result.MissingPages), result.MissingPages)
		}
	})

	t.Run("allowed when all pages have sufficient engagement", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 2, DwellSeconds: 10})

		result := ValidateCourseCompletion(userId, tp.Id, totalPages)
		if !result.Allowed {
			t.Errorf("Expected allowed, got reason: %s (missing: %v)", result.Reason, result.MissingPages)
		}
		if result.EngagedPages != totalPages {
			t.Errorf("Expected %d engaged pages, got %d", totalPages, result.EngagedPages)
		}
	})
}

func TestFullCourseJourneyIntegration(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	// 1. Admin creates presentation
	tp := &TrainingPresentation{OrgId: 1, Name: "Full Journey", FileName: "fj.pdf", FilePath: "/f"}
	if err := PostTrainingPresentation(tp); err != nil {
		t.Fatalf("failed to create presentation: %v", err)
	}

	// 2. Admin sets anti-skip policy
	policy := &AntiSkipPolicy{
		PresentationId:      tp.Id,
		MinDwellSeconds:     5,
		RequireAcknowledge:  true,
		RequireScroll:       true,
		MinScrollDepthPct:   70,
		EnforceSequential:   true,
		AllowBackNavigation: true,
	}
	if err := SaveAntiSkipPolicy(policy); err != nil {
		t.Fatalf("failed to save policy: %v", err)
	}

	var userId int64 = 1
	totalPages := 3

	// 3. User starts course
	cp := &CourseProgress{UserId: userId, PresentationId: tp.Id, Status: "in_progress", TotalPages: totalPages}
	if err := SaveCourseProgress(cp); err != nil {
		t.Fatalf("failed to save progress: %v", err)
	}

	// 4. User reads page 0 — insufficient at first
	RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 2, ScrollDepthPct: 30})
	result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
	if result.Allowed {
		t.Error("Step 4: Expected denied (insufficient dwell)")
	}

	// 5. User keeps reading — enough dwell now, but no ack
	RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 5, ScrollDepthPct: 80})
	result = ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
	if result.Allowed {
		t.Error("Step 5: Expected denied (no acknowledgment)")
	}

	// 6. User acknowledges page 0
	RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 0, Acknowledged: true})
	result = ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
	if !result.Allowed {
		t.Errorf("Step 6: Expected allowed, got: %s", result.Reason)
	}

	// 7. User engages with page 1 fully
	RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 1, DwellSeconds: 10, ScrollDepthPct: 90, Acknowledged: true})
	result = ValidatePageAdvance(userId, tp.Id, 1, 2, totalPages)
	if !result.Allowed {
		t.Errorf("Step 7: Expected allowed, got: %s", result.Reason)
	}

	// 8. User goes back to page 0 (backward navigation)
	result = ValidatePageAdvance(userId, tp.Id, 2, 0, totalPages)
	if !result.Allowed {
		t.Errorf("Step 8: Expected backward nav allowed, got: %s", result.Reason)
	}

	// 9. User engages with page 2 fully
	RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 2, DwellSeconds: 8, ScrollDepthPct: 85, Acknowledged: true})

	// 10. Validate completion
	completion := ValidateCourseCompletion(userId, tp.Id, totalPages)
	if !completion.Allowed {
		t.Errorf("Step 10: Expected completion allowed, got: %s (missing: %v)", completion.Reason, completion.MissingPages)
	}
	if completion.EngagedPages != totalPages {
		t.Errorf("Step 10: Expected %d engaged, got %d", totalPages, completion.EngagedPages)
	}

	// 11. Save completion
	cp.Status = "complete"
	cp.CurrentPage = totalPages
	if err := SaveCourseProgress(cp); err != nil {
		t.Fatalf("failed to save completion: %v", err)
	}
	found, _ := GetCourseProgress(userId, tp.Id)
	if found.Status != "complete" {
		t.Errorf("Step 11: Expected status 'complete', got %q", found.Status)
	}
}

func TestCourseRestartIntegration(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Restart Test", FileName: "r.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	var userId int64 = 1
	totalPages := 3

	policy := &AntiSkipPolicy{PresentationId: tp.Id, MinDwellSeconds: 5}
	SaveAntiSkipPolicy(policy)

	for i := 0; i < totalPages; i++ {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: i, DwellSeconds: 10})
	}
	completion := ValidateCourseCompletion(userId, tp.Id, totalPages)
	if !completion.Allowed {
		t.Fatalf("Expected course completable, got: %s", completion.Reason)
	}

	if err := ResetPageEngagement(userId, tp.Id); err != nil {
		t.Fatalf("failed to reset engagement: %v", err)
	}

	t.Run("completion denied after reset", func(t *testing.T) {
		result := ValidateCourseCompletion(userId, tp.Id, totalPages)
		if result.Allowed {
			t.Error("Expected denied after reset")
		}
		if len(result.MissingPages) != totalPages {
			t.Errorf("Expected %d missing pages, got %d", totalPages, len(result.MissingPages))
		}
	})

	t.Run("advance denied after reset", func(t *testing.T) {
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if result.Allowed {
			t.Error("Expected advance denied after reset")
		}
	})
}

func TestMultiUserIsolation(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "MultiUser", FileName: "mu.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	policy := &AntiSkipPolicy{PresentationId: tp.Id, MinDwellSeconds: 10}
	SaveAntiSkipPolicy(policy)

	totalPages := 3

	for i := 0; i < totalPages; i++ {
		RecordPageEngagement(1, tp.Id, PageEngagementUpdate{PageIndex: i, DwellSeconds: 15})
	}

	t.Run("user 1 can complete", func(t *testing.T) {
		result := ValidateCourseCompletion(1, tp.Id, totalPages)
		if !result.Allowed {
			t.Errorf("Expected user 1 to complete, got: %s", result.Reason)
		}
	})

	t.Run("user 2 cannot complete", func(t *testing.T) {
		result := ValidateCourseCompletion(2, tp.Id, totalPages)
		if result.Allowed {
			t.Error("Expected user 2 denied (no engagement)")
		}
	})

	t.Run("user 2 cannot advance", func(t *testing.T) {
		result := ValidatePageAdvance(2, tp.Id, 0, 1, totalPages)
		if result.Allowed {
			t.Error("Expected user 2 advance denied")
		}
	})

	t.Run("user 1 can advance", func(t *testing.T) {
		result := ValidatePageAdvance(1, tp.Id, 0, 1, totalPages)
		if !result.Allowed {
			t.Errorf("Expected user 1 advance allowed, got: %s", result.Reason)
		}
	})
}

func TestDefaultPolicyIntegration(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Default Policy", FileName: "dp.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	var userId int64 = 1
	totalPages := 2

	t.Run("denied without any engagement (default dwell)", func(t *testing.T) {
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if result.Allowed {
			t.Error("Expected denied with default policy and no engagement")
		}
	})

	t.Run("allowed with sufficient default dwell", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: DefaultMinDwellSeconds})
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if !result.Allowed {
			t.Errorf("Expected allowed with default dwell met, got: %s", result.Reason)
		}
	})
}

func TestHighestUnlockedPageIntegration(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Unlock Test", FileName: "u.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	var userId int64 = 1
	totalPages := 5

	policy := &AntiSkipPolicy{
		PresentationId:    tp.Id,
		MinDwellSeconds:   5,
		EnforceSequential: true,
	}
	SaveAntiSkipPolicy(policy)

	t.Run("only page 0 unlocked initially", func(t *testing.T) {
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if result.PagesUnlocked != 0 {
			t.Errorf("Expected PagesUnlocked=0, got %d", result.PagesUnlocked)
		}
	})

	t.Run("page 1 unlocked after engaging page 0", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 0, DwellSeconds: 10})
		result := ValidatePageAdvance(userId, tp.Id, 0, 1, totalPages)
		if result.PagesUnlocked < 1 {
			t.Errorf("Expected PagesUnlocked >= 1, got %d", result.PagesUnlocked)
		}
	})

	t.Run("pages unlock progressively", func(t *testing.T) {
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 1, DwellSeconds: 10})
		RecordPageEngagement(userId, tp.Id, PageEngagementUpdate{PageIndex: 2, DwellSeconds: 10})

		result := ValidatePageAdvance(userId, tp.Id, 2, 3, totalPages)
		if result.PagesUnlocked < 3 {
			t.Errorf("Expected PagesUnlocked >= 3, got %d", result.PagesUnlocked)
		}
	})
}
