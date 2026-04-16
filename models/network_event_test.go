package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// setupNetworkEventTest initialises an in-memory DB for network event tests.
func setupNetworkEventTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM network_events")
	db.Exec("DELETE FROM network_event_notes")
	db.Exec("DELETE FROM network_event_rules")
	db.Exec("DELETE FROM network_incidents")
	db.Exec("DELETE FROM playbook_execution_logs")
	return func() {
		db.Exec("DELETE FROM network_events")
		db.Exec("DELETE FROM network_event_notes")
		db.Exec("DELETE FROM network_event_rules")
		db.Exec("DELETE FROM network_incidents")
		db.Exec("DELETE FROM playbook_execution_logs")
	}
}

// ---- TableName checks ----

func TestNetworkEventTableName(t *testing.T) {
	e := NetworkEvent{}
	if e.TableName() != "network_events" {
		t.Errorf("expected 'network_events', got '%s'", e.TableName())
	}
}

func TestNetworkEventNoteTableName(t *testing.T) {
	n := NetworkEventNote{}
	if n.TableName() != "network_event_notes" {
		t.Errorf("expected 'network_event_notes', got '%s'", n.TableName())
	}
}

func TestNetworkEventRuleTableName(t *testing.T) {
	r := NetworkEventRule{}
	if r.TableName() != "network_event_rules" {
		t.Errorf("expected 'network_event_rules', got '%s'", r.TableName())
	}
}

// ---- Constants ----

func TestNetworkEventConstants(t *testing.T) {
	if NetworkEventSourceSIEM != "siem" {
		t.Error("unexpected NetworkEventSourceSIEM")
	}
	if NetworkEventTypeLoginAnomaly != "login_anomaly" {
		t.Error("unexpected NetworkEventTypeLoginAnomaly")
	}
	if NetworkEventSeverityCritical != "critical" {
		t.Error("unexpected NetworkEventSeverityCritical")
	}
	if NetworkEventStatusNew != "new" {
		t.Error("unexpected NetworkEventStatusNew")
	}
	if NetworkEventStatusResolved != "resolved" {
		t.Error("unexpected NetworkEventStatusResolved")
	}
}

// ---- PostNetworkEvent ----

func TestPostNetworkEvent(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	e := &NetworkEvent{
		OrgId:     1,
		Source:    NetworkEventSourceSIEM,
		EventType: NetworkEventTypeLoginAnomaly,
		Severity:  NetworkEventSeverityHigh,
		Title:     "Unusual login from new location",
		UserEmail: "user@test.com",
	}
	err := PostNetworkEvent(e)
	if err != nil {
		t.Fatalf("PostNetworkEvent: %v", err)
	}
	if e.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if e.Status != NetworkEventStatusNew {
		t.Fatalf("expected status 'new', got '%s'", e.Status)
	}
	if e.CreatedDate.IsZero() {
		t.Fatal("expected non-zero created_date")
	}
}

func TestPostNetworkEvent_DefaultEventDate(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	e := &NetworkEvent{OrgId: 1, Source: "custom", Title: "Test"}
	PostNetworkEvent(e)
	if e.EventDate.IsZero() {
		t.Fatal("event_date should be set when not provided")
	}
}

// ---- GetNetworkEvent ----

func TestGetNetworkEvent(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	e := &NetworkEvent{
		OrgId: 1, Source: NetworkEventSourceFirewall,
		EventType: NetworkEventTypeNetworkIntrusion, Title: "Fetch me",
	}
	PostNetworkEvent(e)

	fetched, err := GetNetworkEvent(e.Id, 1)
	if err != nil {
		t.Fatalf("GetNetworkEvent: %v", err)
	}
	if fetched.Title != "Fetch me" {
		t.Fatalf("wrong title: %s", fetched.Title)
	}
}

func TestGetNetworkEvent_NotFound(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	_, err := GetNetworkEvent(999, 1)
	if err != ErrNetworkEventNotFound {
		t.Fatalf("expected ErrNetworkEventNotFound, got %v", err)
	}
}

func TestGetNetworkEvent_OrgScoped(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	e := &NetworkEvent{OrgId: 1, Source: "custom", Title: "Org 1 event"}
	PostNetworkEvent(e)

	_, err := GetNetworkEvent(e.Id, 999)
	if err != ErrNetworkEventNotFound {
		t.Fatal("should not find event from another org")
	}
}

// ---- GetNetworkEvents (filtered) ----

func TestGetNetworkEvents_BySource(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", Title: "SIEM 1"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "endpoint", Title: "Endpoint 1"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", Title: "SIEM 2"})

	events, err := GetNetworkEvents(1, NetworkEventFilter{Source: "siem"})
	if err != nil {
		t.Fatalf("GetNetworkEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 SIEM events, got %d", len(events))
	}
}

func TestGetNetworkEvents_BySeverity(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "custom", Severity: "critical", Title: "Critical"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "custom", Severity: "low", Title: "Low"})

	events, _ := GetNetworkEvents(1, NetworkEventFilter{Severity: "critical"})
	if len(events) != 1 {
		t.Fatalf("expected 1 critical event, got %d", len(events))
	}
}

func TestGetNetworkEvents_WithLimit(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	for i := 0; i < 5; i++ {
		PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "custom", Title: "Event"})
	}

	events, _ := GetNetworkEvents(1, NetworkEventFilter{Limit: 2})
	if len(events) != 2 {
		t.Fatalf("expected 2 events with limit=2, got %d", len(events))
	}
}

// ---- UpdateNetworkEventStatus ----

func TestUpdateNetworkEventStatus(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	e := &NetworkEvent{OrgId: 1, Source: "custom", Title: "Status test"}
	PostNetworkEvent(e)

	err := UpdateNetworkEventStatus(e.Id, 1, NetworkEventStatusInvestigating, 42)
	if err != nil {
		t.Fatalf("UpdateNetworkEventStatus: %v", err)
	}
	updated, _ := GetNetworkEvent(e.Id, 1)
	if updated.Status != NetworkEventStatusInvestigating {
		t.Fatalf("expected 'investigating', got '%s'", updated.Status)
	}
}

func TestUpdateNetworkEventStatus_Resolved(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	e := &NetworkEvent{OrgId: 1, Source: "custom", Title: "Resolve test"}
	PostNetworkEvent(e)

	err := UpdateNetworkEventStatus(e.Id, 1, NetworkEventStatusResolved, 42)
	if err != nil {
		t.Fatalf("UpdateNetworkEventStatus: %v", err)
	}
	updated, _ := GetNetworkEvent(e.Id, 1)
	if updated.Status != NetworkEventStatusResolved {
		t.Fatalf("expected 'resolved', got '%s'", updated.Status)
	}
	if updated.ResolvedBy != 42 {
		t.Fatalf("expected resolved_by=42, got %d", updated.ResolvedBy)
	}
	if updated.ResolvedDate.IsZero() {
		t.Fatal("expected non-zero resolved_date")
	}
}

func TestUpdateNetworkEventStatus_NotFound(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	err := UpdateNetworkEventStatus(999, 1, "resolved", 1)
	if err != ErrNetworkEventNotFound {
		t.Fatalf("expected ErrNetworkEventNotFound, got %v", err)
	}
}

// ---- Notes ----

func TestAddNetworkEventNote(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	e := &NetworkEvent{OrgId: 1, Source: "custom", Title: "Note test"}
	PostNetworkEvent(e)

	note := &NetworkEventNote{EventId: e.Id, UserId: 1, Content: "Investigation started"}
	err := AddNetworkEventNote(note)
	if err != nil {
		t.Fatalf("AddNetworkEventNote: %v", err)
	}
	if note.Id == 0 {
		t.Fatal("expected non-zero note ID")
	}

	// Verify hydration
	fetched, _ := GetNetworkEvent(e.Id, 1)
	if len(fetched.Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(fetched.Notes))
	}
	if fetched.Notes[0].Content != "Investigation started" {
		t.Fatalf("wrong note content: %s", fetched.Notes[0].Content)
	}
}

// ---- BulkIngest ----

func TestBulkIngestNetworkEvents(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	events := []NetworkEvent{
		{Source: "siem", EventType: "login_anomaly", Title: "Event 1", Severity: "high"},
		{Source: "endpoint", EventType: "malware_detected", Title: "Event 2", Severity: "critical"},
		{Source: "firewall", EventType: "network_intrusion", Title: "Event 3", Severity: "medium"},
	}
	count, err := BulkIngestNetworkEvents(1, events)
	if err != nil {
		t.Fatalf("BulkIngestNetworkEvents: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 ingested, got %d", count)
	}

	all, _ := GetNetworkEvents(1, NetworkEventFilter{})
	if len(all) != 3 {
		t.Fatalf("expected 3 events in DB, got %d", len(all))
	}
}

// ---- Rules ----

func TestNetworkEventRuleCRUD(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	// Create
	rule := &NetworkEventRule{
		OrgId: 1, Name: "Escalate SIEM alerts",
		SourceMatch: "siem", AutoSeverity: "critical", Enabled: true,
	}
	err := PostNetworkEventRule(rule)
	if err != nil {
		t.Fatalf("PostNetworkEventRule: %v", err)
	}
	if rule.Id == 0 {
		t.Fatal("expected non-zero rule ID")
	}

	// List
	rules, err := GetNetworkEventRules(1)
	if err != nil {
		t.Fatalf("GetNetworkEventRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	// Update
	rule.Name = "Updated rule"
	err = PutNetworkEventRule(rule)
	if err != nil {
		t.Fatalf("PutNetworkEventRule: %v", err)
	}
	rules, _ = GetNetworkEventRules(1)
	if rules[0].Name != "Updated rule" {
		t.Fatalf("expected updated name, got '%s'", rules[0].Name)
	}

	// Delete
	err = DeleteNetworkEventRule(rule.Id, 1)
	if err != nil {
		t.Fatalf("DeleteNetworkEventRule: %v", err)
	}
	rules, _ = GetNetworkEventRules(1)
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(rules))
	}
}

func TestDeleteNetworkEventRule_OrgScoped(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	rule := &NetworkEventRule{OrgId: 1, Name: "Org 1 rule", Enabled: true}
	PostNetworkEventRule(rule)

	err := DeleteNetworkEventRule(rule.Id, 999)
	if err == nil {
		t.Fatal("should not delete rule from another org")
	}
}

// ---- applyNetworkEventRules ----

func TestApplyNetworkEventRules_AutoSeverity(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEventRule(&NetworkEventRule{
		OrgId: 1, Name: "SIEM to critical",
		SourceMatch: "siem", AutoSeverity: "critical", Enabled: true,
	})

	e := &NetworkEvent{OrgId: 1, Source: "siem", Severity: "info", Title: "Test"}
	PostNetworkEvent(e)

	if e.Severity != "critical" {
		t.Fatalf("expected severity overridden to 'critical', got '%s'", e.Severity)
	}
}

func TestApplyNetworkEventRules_AutoAssign(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEventRule(&NetworkEventRule{
		OrgId: 1, Name: "Assign endpoint alerts",
		SourceMatch: "endpoint", AutoAssign: 42, Enabled: true,
	})

	e := &NetworkEvent{OrgId: 1, Source: "endpoint", Title: "Test"}
	PostNetworkEvent(e)

	if e.AssignedTo != 42 {
		t.Fatalf("expected assigned_to=42, got %d", e.AssignedTo)
	}
}

func TestApplyNetworkEventRules_NoMatch(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEventRule(&NetworkEventRule{
		OrgId: 1, Name: "SIEM only",
		SourceMatch: "siem", AutoSeverity: "critical", Enabled: true,
	})

	e := &NetworkEvent{OrgId: 1, Source: "firewall", Severity: "low", Title: "Test"}
	PostNetworkEvent(e)

	if e.Severity != "low" {
		t.Fatalf("expected severity unchanged at 'low', got '%s'", e.Severity)
	}
}

func TestApplyNetworkEventRules_DisabledRule(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEventRule(&NetworkEventRule{
		OrgId: 1, Name: "Disabled rule",
		SourceMatch: "siem", AutoSeverity: "critical", Enabled: false,
	})

	e := &NetworkEvent{OrgId: 1, Source: "siem", Severity: "info", Title: "Test"}
	PostNetworkEvent(e)

	if e.Severity != "info" {
		t.Fatalf("disabled rule should not change severity, got '%s'", e.Severity)
	}
}

// ---- matchesField ----

func TestMatchesField_Exact(t *testing.T) {
	if !matchesField("siem", "siem") {
		t.Error("exact match should return true")
	}
}

func TestMatchesField_CaseInsensitive(t *testing.T) {
	if !matchesField("SIEM", "siem") {
		t.Error("case-insensitive match should return true")
	}
}

func TestMatchesField_Substring(t *testing.T) {
	if !matchesField("email_gateway", "email") {
		t.Error("substring match should return true")
	}
}

func TestMatchesField_NoMatch(t *testing.T) {
	if matchesField("firewall", "siem") {
		t.Error("non-matching should return false")
	}
}

// ---- Dashboard ----

func TestGetNetworkEventDashboard_Empty(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	dash, err := GetNetworkEventDashboard(1)
	if err != nil {
		t.Fatalf("GetNetworkEventDashboard: %v", err)
	}
	if dash.TotalEvents != 0 {
		t.Fatalf("expected 0 total events, got %d", dash.TotalEvents)
	}
	if dash.OpenEvents != 0 {
		t.Fatalf("expected 0 open events, got %d", dash.OpenEvents)
	}
}

func TestGetNetworkEventDashboard_WithData(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", Severity: "critical", Title: "E1"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", Severity: "high", Title: "E2"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "endpoint", Severity: "low", Title: "E3"})

	dash, err := GetNetworkEventDashboard(1)
	if err != nil {
		t.Fatalf("GetNetworkEventDashboard: %v", err)
	}
	if dash.TotalEvents != 3 {
		t.Fatalf("expected 3 total events, got %d", dash.TotalEvents)
	}
	if dash.OpenEvents != 3 {
		t.Fatalf("expected 3 open events (all new), got %d", dash.OpenEvents)
	}
	if dash.CriticalOpen != 1 {
		t.Fatalf("expected 1 critical open, got %d", dash.CriticalOpen)
	}
	if dash.HighOpen != 1 {
		t.Fatalf("expected 1 high open, got %d", dash.HighOpen)
	}
}

// ---- Trend ----

func TestGetNetworkEventTrend(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEvent(&NetworkEvent{
		OrgId: 1, Source: "custom", Title: "Today",
		EventDate: time.Now().UTC(),
	})

	trend, err := GetNetworkEventTrend(1, 7)
	if err != nil {
		t.Fatalf("GetNetworkEventTrend: %v", err)
	}
	if len(trend) == 0 {
		t.Fatal("expected at least 1 day in trend")
	}
	total := 0
	for _, d := range trend {
		total += d.Count
	}
	if total != 1 {
		t.Fatalf("expected total count of 1, got %d", total)
	}
}

func TestGetNetworkEventTrend_Empty(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	trend, err := GetNetworkEventTrend(1, 30)
	if err != nil {
		t.Fatalf("GetNetworkEventTrend: %v", err)
	}
	if len(trend) != 0 {
		t.Fatalf("expected 0 days in trend for empty DB, got %d", len(trend))
	}
}

// ---- Default values ----

func TestNetworkEventDashboardDefaults(t *testing.T) {
	d := NetworkEventDashboard{}
	if d.TotalEvents != 0 || d.OpenEvents != 0 || d.CriticalOpen != 0 {
		t.Error("default dashboard should have zero values")
	}
}

func TestNetworkEventFilterDefaults(t *testing.T) {
	f := NetworkEventFilter{}
	if f.Limit != 0 || f.Source != "" || f.Status != "" {
		t.Error("default filter should have zero values")
	}
}

// ---- MITRE ATT&CK ----

func TestMitreTacticTechniqueMapping(t *testing.T) {
	if len(MitreTacticTechnique) != 12 {
		t.Fatalf("expected 12 tactics, got %d", len(MitreTacticTechnique))
	}
	if len(MitreTacticOrder) != 12 {
		t.Fatalf("expected 12 ordered tactics, got %d", len(MitreTacticOrder))
	}
	for _, tactic := range MitreTacticOrder {
		if _, ok := MitreTacticTechnique[tactic]; !ok {
			t.Errorf("tactic %q not found in MitreTacticTechnique", tactic)
		}
	}
}

func TestMitreTechniqueNamesCompleteness(t *testing.T) {
	for _, techniques := range MitreTacticTechnique {
		for _, tid := range techniques {
			if _, ok := MitreTechniqueNames[tid]; !ok {
				t.Errorf("technique %s missing from MitreTechniqueNames", tid)
			}
		}
	}
}

func TestGetMitreHeatmap_Empty(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	data, err := GetMitreHeatmap(1)
	if err != nil {
		t.Fatalf("GetMitreHeatmap: %v", err)
	}
	if data.TotalMapped != 0 {
		t.Fatalf("expected 0 mapped, got %d", data.TotalMapped)
	}
	if len(data.Rows) != 12 {
		t.Fatalf("expected 12 tactic rows, got %d", len(data.Rows))
	}
}

func TestGetMitreHeatmap_WithData(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", MitreTechniqueId: "T1566", Title: "Phishing 1"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", MitreTechniqueId: "T1566", Title: "Phishing 2"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "endpoint", MitreTechniqueId: "T1110", Title: "Brute Force"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "firewall", Title: "No MITRE"})

	data, err := GetMitreHeatmap(1)
	if err != nil {
		t.Fatalf("GetMitreHeatmap: %v", err)
	}
	if data.TotalMapped != 3 {
		t.Fatalf("expected 3 mapped, got %d", data.TotalMapped)
	}
	if data.TotalUnmapped != 1 {
		t.Fatalf("expected 1 unmapped, got %d", data.TotalUnmapped)
	}
	if len(data.TopTechniques) < 1 {
		t.Fatal("expected at least 1 top technique")
	}
	if data.TopTechniques[0].TechniqueId != "T1566" {
		t.Fatalf("expected top technique T1566, got %s", data.TopTechniques[0].TechniqueId)
	}
}

// ---- Network Event with new fields ----

func TestNetworkEventMitreField(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	e := &NetworkEvent{OrgId: 1, Source: "siem", MitreTechniqueId: "T1566", Title: "Phish"}
	PostNetworkEvent(e)

	fetched, err := GetNetworkEvent(e.Id, 1)
	if err != nil {
		t.Fatalf("GetNetworkEvent: %v", err)
	}
	if fetched.MitreTechniqueId != "T1566" {
		t.Fatalf("expected mitre_technique_id='T1566', got '%s'", fetched.MitreTechniqueId)
	}
}

func TestNetworkEventsFilterByMitre(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", MitreTechniqueId: "T1566", Title: "Phish"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", MitreTechniqueId: "T1110", Title: "Brute"})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", Title: "No MITRE"})

	events, err := GetNetworkEvents(1, NetworkEventFilter{MitreTechniqueId: "T1566"})
	if err != nil {
		t.Fatalf("GetNetworkEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event with T1566, got %d", len(events))
	}
}

// ---- Incident Correlation ----

func TestCorrelateNetworkEvents_NoEvents(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	count, err := CorrelateNetworkEvents(1)
	if err != nil {
		t.Fatalf("CorrelateNetworkEvents: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 incidents, got %d", count)
	}
}

func TestCorrelateNetworkEvents_CreatesIncident(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()
	db.Exec("DELETE FROM network_incidents")

	now := time.Now().UTC()
	// Two events from same IP + same user within 1 hour
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", SourceIP: "10.0.0.1", UserEmail: "alice@test.com", Title: "E1", EventDate: now})
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "siem", SourceIP: "10.0.0.1", UserEmail: "alice@test.com", Title: "E2", EventDate: now.Add(30 * time.Minute)})
	// One event from a different IP (should not correlate)
	PostNetworkEvent(&NetworkEvent{OrgId: 1, Source: "endpoint", SourceIP: "10.0.0.99", UserEmail: "bob@test.com", Title: "E3", EventDate: now})

	count, err := CorrelateNetworkEvents(1)
	if err != nil {
		t.Fatalf("CorrelateNetworkEvents: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 incident, got %d", count)
	}

	incidents, err := GetNetworkIncidents(1, "", 50, 0)
	if err != nil {
		t.Fatalf("GetNetworkIncidents: %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident in DB, got %d", len(incidents))
	}
	if incidents[0].EventCount != 2 {
		t.Fatalf("expected 2 events in incident, got %d", incidents[0].EventCount)
	}

	// Verify events are linked
	inc, err := GetNetworkIncident(incidents[0].Id, 1)
	if err != nil {
		t.Fatalf("GetNetworkIncident: %v", err)
	}
	if len(inc.Events) != 2 {
		t.Fatalf("expected 2 events hydrated on incident, got %d", len(inc.Events))
	}
}

func TestNetworkIncidentStatusUpdate(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()
	db.Exec("DELETE FROM network_incidents")

	inc := NetworkIncident{OrgId: 1, Title: "Test", Status: "open", CreatedDate: time.Now().UTC(), ModifiedDate: time.Now().UTC()}
	db.Create(&inc)

	err := UpdateNetworkIncidentStatus(inc.Id, 1, "resolved")
	if err != nil {
		t.Fatalf("UpdateNetworkIncidentStatus: %v", err)
	}

	fetched, _ := GetNetworkIncident(inc.Id, 1)
	if fetched.Status != "resolved" {
		t.Fatalf("expected status 'resolved', got '%s'", fetched.Status)
	}
}

// ---- Playbook / Extended Rules ----

func TestNetworkEventRuleWithSeverityMatch(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()

	PostNetworkEventRule(&NetworkEventRule{
		OrgId: 1, Name: "Critical email gateway",
		SourceMatch: "email_gateway", SeverityMatch: "critical",
		AutoAssign: 99, Enabled: true,
	})

	// Should match
	e := &NetworkEvent{OrgId: 1, Source: "email_gateway", Severity: "critical", Title: "Match"}
	PostNetworkEvent(e)
	if e.AssignedTo != 99 {
		t.Fatalf("expected assigned_to=99, got %d", e.AssignedTo)
	}

	// Should NOT match (different severity)
	e2 := &NetworkEvent{OrgId: 1, Source: "email_gateway", Severity: "low", Title: "No Match"}
	PostNetworkEvent(e2)
	if e2.AssignedTo != 0 {
		t.Fatalf("expected assigned_to=0, got %d", e2.AssignedTo)
	}
}

func TestPlaybookExecution(t *testing.T) {
	teardown := setupNetworkEventTest(t)
	defer teardown()
	db.Exec("DELETE FROM playbook_execution_logs")

	actions := `[{"type":"set_severity","value":"critical"},{"type":"add_note","value":"Auto-escalated by playbook"}]`
	PostNetworkEventRule(&NetworkEventRule{
		OrgId: 1, Name: "Escalate email gateway critical",
		SourceMatch: "email_gateway", SeverityMatch: "high",
		IsPlaybook: true, PlaybookActions: actions, Enabled: true,
	})

	e := &NetworkEvent{OrgId: 1, Source: "email_gateway", Severity: "high", Title: "Test Playbook"}
	PostNetworkEvent(e)

	if e.Severity != "critical" {
		t.Fatalf("expected severity='critical' from playbook, got '%s'", e.Severity)
	}

	// Check execution log was created
	logs, err := GetPlaybookExecutionLogs(1, 10)
	if err != nil {
		t.Fatalf("GetPlaybookExecutionLogs: %v", err)
	}
	if len(logs) < 1 {
		t.Fatal("expected at least 1 playbook execution log")
	}
	if logs[0].RuleName != "Escalate email gateway critical" {
		t.Fatalf("expected rule name in log, got '%s'", logs[0].RuleName)
	}
}

func TestNetworkIncidentTableName(t *testing.T) {
	inc := NetworkIncident{}
	if inc.TableName() != "network_incidents" {
		t.Errorf("expected 'network_incidents', got '%s'", inc.TableName())
	}
}

func TestPlaybookExecutionLogTableName(t *testing.T) {
	l := PlaybookExecutionLog{}
	if l.TableName() != "playbook_execution_logs" {
		t.Errorf("expected 'playbook_execution_logs', got '%s'", l.TableName())
	}
}
