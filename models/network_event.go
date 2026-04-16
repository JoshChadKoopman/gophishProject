package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Network event source constants.
const (
	NetworkEventSourceSIEM         = "siem"
	NetworkEventSourceEndpoint     = "endpoint"
	NetworkEventSourceFirewall     = "firewall"
	NetworkEventSourceEmailGateway = "email_gateway"
	NetworkEventSourceCloud        = "cloud"
	NetworkEventSourceIdentity     = "identity"
	NetworkEventSourceCustom       = "custom"
)

// Network event type constants.
const (
	NetworkEventTypeLoginAnomaly       = "login_anomaly"
	NetworkEventTypeBruteForce         = "brute_force"
	NetworkEventTypeImpossibleTravel   = "impossible_travel"
	NetworkEventTypeMalwareDetected    = "malware_detected"
	NetworkEventTypeDataExfiltration   = "data_exfiltration"
	NetworkEventTypeUnauthorizedAccess = "unauthorized_access"
	NetworkEventTypePolicyViolation    = "policy_violation"
	NetworkEventTypeSuspiciousEmail    = "suspicious_email"
	NetworkEventTypeEndpointAlert      = "endpoint_alert"
	NetworkEventTypeNetworkIntrusion   = "network_intrusion"
	NetworkEventTypeDNSAnomaly         = "dns_anomaly"
	NetworkEventTypeCertificateIssue   = "certificate_issue"
)

// Network event severity constants.
const (
	NetworkEventSeverityInfo     = "info"
	NetworkEventSeverityLow      = "low"
	NetworkEventSeverityMedium   = "medium"
	NetworkEventSeverityHigh     = "high"
	NetworkEventSeverityCritical = "critical"
)

// Network event status constants.
const (
	NetworkEventStatusNew           = "new"
	NetworkEventStatusAcknowledged  = "acknowledged"
	NetworkEventStatusInvestigating = "investigating"
	NetworkEventStatusResolved      = "resolved"
	NetworkEventStatusFalsePositive = "false_positive"
)

// ErrNetworkEventNotFound is returned when a network event cannot be found.
var ErrNetworkEventNotFound = errors.New("network event not found")

// ErrNetworkIncidentNotFound is returned when an incident cannot be found.
var ErrNetworkIncidentNotFound = errors.New("network incident not found")

// NetworkEvent represents a security event ingested from an external source.
type NetworkEvent struct {
	Id               int64              `json:"id" gorm:"primary_key;auto_increment"`
	OrgId            int64              `json:"org_id" gorm:"column:org_id"`
	Source           string             `json:"source"`
	EventType        string             `json:"event_type" gorm:"column:event_type"`
	Severity         string             `json:"severity" gorm:"default:'info'"`
	Title            string             `json:"title"`
	Description      string             `json:"description" gorm:"type:text"`
	SourceIP         string             `json:"source_ip" gorm:"column:source_ip"`
	DestinationIP    string             `json:"destination_ip" gorm:"column:destination_ip"`
	UserId           int64              `json:"user_id" gorm:"column:user_id"`
	UserEmail        string             `json:"user_email" gorm:"column:user_email"`
	DeviceId         string             `json:"device_id" gorm:"column:device_id"`
	MitreTechniqueId string             `json:"mitre_technique_id" gorm:"column:mitre_technique_id"`
	IncidentId       int64              `json:"incident_id" gorm:"column:incident_id"`
	RawPayload       string             `json:"raw_payload" gorm:"column:raw_payload;type:text"`
	Status           string             `json:"status" gorm:"default:'new'"`
	AssignedTo       int64              `json:"assigned_to" gorm:"column:assigned_to"`
	ResolvedBy       int64              `json:"resolved_by" gorm:"column:resolved_by"`
	ResolvedDate     time.Time          `json:"resolved_date" gorm:"column:resolved_date"`
	EventDate        time.Time          `json:"event_date" gorm:"column:event_date"`
	CreatedDate      time.Time          `json:"created_date" gorm:"column:created_date"`
	ModifiedDate     time.Time          `json:"modified_date" gorm:"column:modified_date"`
	Notes            []NetworkEventNote `json:"notes,omitempty" gorm:"-"`
}

// TableName overrides the default table name.
func (NetworkEvent) TableName() string {
	return "network_events"
}

// NetworkEventNote stores analyst notes attached to a network event.
type NetworkEventNote struct {
	Id          int64     `json:"id" gorm:"primary_key;auto_increment"`
	EventId     int64     `json:"event_id" gorm:"column:event_id"`
	UserId      int64     `json:"user_id" gorm:"column:user_id"`
	Content     string    `json:"content" gorm:"type:text"`
	CreatedDate time.Time `json:"created_date" gorm:"column:created_date"`
}

// TableName overrides the default table name.
func (NetworkEventNote) TableName() string {
	return "network_event_notes"
}

// NetworkEventRule defines an automation rule / playbook.
// Legacy rules use SourceMatch + EventTypeMatch + AutoSeverity + AutoAssign.
// Playbook rules (IsPlaybook=true) also honour SeverityMatch and execute
// a chain of PlaybookActions stored as JSON in the PlaybookActions column.
type NetworkEventRule struct {
	Id              int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId           int64     `json:"org_id" gorm:"column:org_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	SourceMatch     string    `json:"source_match" gorm:"column:source_match"`
	EventTypeMatch  string    `json:"event_type_match" gorm:"column:event_type_match"`
	SeverityMatch   string    `json:"severity_match" gorm:"column:severity_match"`
	AutoSeverity    string    `json:"auto_severity" gorm:"column:auto_severity"`
	AutoAssign      int64     `json:"auto_assign" gorm:"column:auto_assign"`
	IsPlaybook      bool      `json:"is_playbook" gorm:"column:is_playbook;default:false"`
	PlaybookActions string    `json:"playbook_actions" gorm:"column:playbook_actions;type:text"`
	Enabled         bool      `json:"enabled" gorm:"default:true"`
	CreatedDate     time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate    time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// TableName overrides the default table name.
func (NetworkEventRule) TableName() string {
	return "network_event_rules"
}

// NetworkEventFilter holds optional filter parameters for listing events.
type NetworkEventFilter struct {
	Source           string    `json:"source"`
	EventType        string    `json:"event_type"`
	Severity         string    `json:"severity"`
	Status           string    `json:"status"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	UserEmail        string    `json:"user_email"`
	MitreTechniqueId string    `json:"mitre_technique_id"`
	IncidentId       int64     `json:"incident_id"`
	Limit            int       `json:"limit"`
	Offset           int       `json:"offset"`
}

// NetworkEventDashboard is the aggregate response for the dashboard view.
type NetworkEventDashboard struct {
	TotalEvents        int               `json:"total_events"`
	OpenEvents         int               `json:"open_events"`
	CriticalOpen       int               `json:"critical_open"`
	HighOpen           int               `json:"high_open"`
	AvgResolutionHours float64           `json:"avg_resolution_hours"`
	EventsBySource     []SourceCount     `json:"events_by_source"`
	EventsByType       []TypeCount       `json:"events_by_type"`
	EventsBySeverity   []SeverityCount   `json:"events_by_severity"`
	RecentEvents       []NetworkEvent    `json:"recent_events"`
	TrendData          []DailyEventCount `json:"trend_data"`
}

// SourceCount is a source-level aggregate for the dashboard.
type SourceCount struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

// TypeCount is an event-type-level aggregate for the dashboard.
type TypeCount struct {
	EventType string `json:"event_type"`
	Count     int    `json:"count"`
}

// SeverityCount is a severity-level aggregate for the dashboard.
type SeverityCount struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

// DailyEventCount holds the event count for a single calendar day.
type DailyEventCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// ---------------------------------------------------------------------------
// MITRE ATT&CK Heatmap types
// ---------------------------------------------------------------------------

// MitreTacticTechnique maps a MITRE ATT&CK tactic to its techniques.
var MitreTacticTechnique = map[string][]string{
	"Initial Access":       {"T1566", "T1190", "T1133", "T1078", "T1195"},
	"Execution":            {"T1059", "T1204", "T1053", "T1203", "T1047"},
	"Persistence":          {"T1098", "T1136", "T1053", "T1547", "T1078"},
	"Privilege Escalation": {"T1548", "T1134", "T1068", "T1078", "T1547"},
	"Defense Evasion":      {"T1070", "T1562", "T1036", "T1027", "T1218"},
	"Credential Access":    {"T1110", "T1003", "T1558", "T1556", "T1555"},
	"Discovery":            {"T1087", "T1069", "T1046", "T1057", "T1018"},
	"Lateral Movement":     {"T1021", "T1570", "T1210", "T1080", "T1563"},
	"Collection":           {"T1560", "T1005", "T1114", "T1039", "T1213"},
	"Exfiltration":         {"T1041", "T1048", "T1567", "T1029", "T1537"},
	"Command & Control":    {"T1071", "T1132", "T1573", "T1090", "T1105"},
	"Impact":               {"T1486", "T1489", "T1490", "T1561", "T1496"},
}

// MitreTechniqueNames provides human-readable names for technique IDs.
var MitreTechniqueNames = map[string]string{
	"T1566": "Phishing",
	"T1190": "Exploit Public-Facing App",
	"T1133": "External Remote Services",
	"T1078": "Valid Accounts",
	"T1195": "Supply Chain Compromise",
	"T1059": "Command & Scripting Interpreter",
	"T1204": "User Execution",
	"T1053": "Scheduled Task/Job",
	"T1203": "Exploitation for Client Execution",
	"T1047": "WMI",
	"T1098": "Account Manipulation",
	"T1136": "Create Account",
	"T1547": "Boot/Logon Autostart",
	"T1548": "Abuse Elevation Control",
	"T1134": "Access Token Manipulation",
	"T1068": "Exploitation for Privilege Escalation",
	"T1070": "Indicator Removal",
	"T1562": "Impair Defenses",
	"T1036": "Masquerading",
	"T1027": "Obfuscated Files",
	"T1218": "System Binary Proxy Execution",
	"T1110": "Brute Force",
	"T1003": "OS Credential Dumping",
	"T1558": "Steal/Forge Kerberos Tickets",
	"T1556": "Modify Authentication Process",
	"T1555": "Credentials from Password Stores",
	"T1087": "Account Discovery",
	"T1069": "Permission Groups Discovery",
	"T1046": "Network Service Discovery",
	"T1057": "Process Discovery",
	"T1018": "Remote System Discovery",
	"T1021": "Remote Services",
	"T1570": "Lateral Tool Transfer",
	"T1210": "Exploitation of Remote Services",
	"T1080": "Taint Shared Content",
	"T1563": "Remote Service Session Hijacking",
	"T1560": "Archive Collected Data",
	"T1005": "Data from Local System",
	"T1114": "Email Collection",
	"T1039": "Data from Network Shared Drive",
	"T1213": "Data from Information Repositories",
	"T1041": "Exfiltration Over C2 Channel",
	"T1048": "Exfiltration Over Alternative Protocol",
	"T1567": "Exfiltration Over Web Service",
	"T1029": "Scheduled Transfer",
	"T1537": "Transfer Data to Cloud Account",
	"T1071": "Application Layer Protocol",
	"T1132": "Data Encoding",
	"T1573": "Encrypted Channel",
	"T1090": "Proxy",
	"T1105": "Ingress Tool Transfer",
	"T1486": "Data Encrypted for Impact",
	"T1489": "Service Stop",
	"T1490": "Inhibit System Recovery",
	"T1561": "Disk Wipe",
	"T1496": "Resource Hijacking",
}

// MitreTacticOrder defines the canonical display order for tactics.
var MitreTacticOrder = []string{
	"Initial Access", "Execution", "Persistence", "Privilege Escalation",
	"Defense Evasion", "Credential Access", "Discovery", "Lateral Movement",
	"Collection", "Exfiltration", "Command & Control", "Impact",
}

// MitreHeatmapCell represents one cell in the MITRE ATT&CK heatmap.
type MitreHeatmapCell struct {
	TechniqueId   string `json:"technique_id"`
	TechniqueName string `json:"technique_name"`
	Count         int    `json:"count"`
}

// MitreHeatmapRow represents a tactic row with its techniques and counts.
type MitreHeatmapRow struct {
	Tactic     string             `json:"tactic"`
	Techniques []MitreHeatmapCell `json:"techniques"`
}

// MitreHeatmapData is the full heatmap response.
type MitreHeatmapData struct {
	Rows          []MitreHeatmapRow  `json:"rows"`
	TopTechniques []MitreHeatmapCell `json:"top_techniques"`
	TotalMapped   int                `json:"total_mapped"`
	TotalUnmapped int                `json:"total_unmapped"`
}

// ---------------------------------------------------------------------------
// Incident Correlation types
// ---------------------------------------------------------------------------

// NetworkIncident represents a correlated group of network events.
type NetworkIncident struct {
	Id           int64          `json:"id" gorm:"primary_key;auto_increment"`
	OrgId        int64          `json:"org_id" gorm:"column:org_id"`
	Title        string         `json:"title"`
	Description  string         `json:"description" gorm:"type:text"`
	Severity     string         `json:"severity" gorm:"default:'medium'"`
	Status       string         `json:"status" gorm:"default:'open'"`
	SourceIP     string         `json:"source_ip" gorm:"column:source_ip"`
	UserEmail    string         `json:"user_email" gorm:"column:user_email"`
	EventCount   int            `json:"event_count" gorm:"column:event_count"`
	FirstSeen    time.Time      `json:"first_seen" gorm:"column:first_seen"`
	LastSeen     time.Time      `json:"last_seen" gorm:"column:last_seen"`
	AssignedTo   int64          `json:"assigned_to" gorm:"column:assigned_to"`
	CreatedDate  time.Time      `json:"created_date" gorm:"column:created_date"`
	ModifiedDate time.Time      `json:"modified_date" gorm:"column:modified_date"`
	Events       []NetworkEvent `json:"events,omitempty" gorm:"-"`
}

func (NetworkIncident) TableName() string {
	return "network_incidents"
}

// ---------------------------------------------------------------------------
// Playbook types
// ---------------------------------------------------------------------------

// PlaybookAction defines a single action in a playbook's action chain.
type PlaybookAction struct {
	Type  string `json:"type"`  // "set_severity", "assign_to", "create_incident", "create_remediation", "notify_webhook", "add_note"
	Value string `json:"value"` // e.g. severity value, user ID, webhook URL, note text
}

// PlaybookExecutionLog tracks when a playbook fires.
type PlaybookExecutionLog struct {
	Id          int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId       int64     `json:"org_id" gorm:"column:org_id"`
	RuleId      int64     `json:"rule_id" gorm:"column:rule_id"`
	EventId     int64     `json:"event_id" gorm:"column:event_id"`
	RuleName    string    `json:"rule_name" gorm:"column:rule_name"`
	ActionsRun  string    `json:"actions_run" gorm:"column:actions_run;type:text"`
	Status      string    `json:"status" gorm:"default:'success'"`
	CreatedDate time.Time `json:"created_date" gorm:"column:created_date"`
}

func (PlaybookExecutionLog) TableName() string {
	return "playbook_execution_logs"
}

// ---------------------------------------------------------------------------
// CRUD operations
// ---------------------------------------------------------------------------

// PostNetworkEvent creates a new network event, applying any matching
// automation rules and playbooks before persisting.
func PostNetworkEvent(e *NetworkEvent) error {
	now := time.Now().UTC()
	e.CreatedDate = now
	e.ModifiedDate = now
	if e.Status == "" {
		e.Status = NetworkEventStatusNew
	}
	if e.EventDate.IsZero() {
		e.EventDate = now
	}

	applyNetworkEventRules(e)

	err := db.Create(e).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// GetNetworkEvent returns a single event by ID scoped to an organization,
// with its notes hydrated.
func GetNetworkEvent(id, orgId int64) (NetworkEvent, error) {
	var event NetworkEvent
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&event).Error
	if err != nil {
		return event, ErrNetworkEventNotFound
	}
	hydrateNetworkEvent(&event)
	return event, nil
}

// GetNetworkEvents returns a filtered list of events for an organization.
func GetNetworkEvents(orgId int64, filter NetworkEventFilter) ([]NetworkEvent, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	query := db.Where("org_id = ?", orgId)
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}
	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if !filter.StartDate.IsZero() {
		query = query.Where("event_date >= ?", filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("event_date <= ?", filter.EndDate)
	}
	if filter.UserEmail != "" {
		query = query.Where("user_email = ?", filter.UserEmail)
	}
	if filter.MitreTechniqueId != "" {
		query = query.Where("mitre_technique_id = ?", filter.MitreTechniqueId)
	}
	if filter.IncidentId > 0 {
		query = query.Where("incident_id = ?", filter.IncidentId)
	}

	var events []NetworkEvent
	err := query.Order("event_date DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

// UpdateNetworkEventStatus transitions an event to a new status.
func UpdateNetworkEventStatus(id, orgId int64, status string, userId int64) error {
	event, err := GetNetworkEvent(id, orgId)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status":        status,
		"modified_date": time.Now().UTC(),
	}

	if status == NetworkEventStatusResolved || status == NetworkEventStatusFalsePositive {
		updates["resolved_by"] = userId
		updates["resolved_date"] = time.Now().UTC()
	}

	return db.Model(&event).Updates(updates).Error
}

// AddNetworkEventNote appends a note to an event.
func AddNetworkEventNote(note *NetworkEventNote) error {
	note.CreatedDate = time.Now().UTC()
	err := db.Create(note).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// BulkIngestNetworkEvents ingests a batch of events for an organization,
// applying rules to each. It returns the number of events successfully created.
func BulkIngestNetworkEvents(orgId int64, events []NetworkEvent) (int, error) {
	created := 0
	now := time.Now().UTC()
	for i := range events {
		events[i].OrgId = orgId
		events[i].CreatedDate = now
		events[i].ModifiedDate = now
		if events[i].Status == "" {
			events[i].Status = NetworkEventStatusNew
		}
		if events[i].EventDate.IsZero() {
			events[i].EventDate = now
		}

		applyNetworkEventRules(&events[i])

		if err := db.Create(&events[i]).Error; err != nil {
			log.Errorf("network_event: bulk ingest failed for event %d: %v", i, err)
			continue
		}
		created++
	}
	return created, nil
}

// ---------------------------------------------------------------------------
// Dashboard / trend
// ---------------------------------------------------------------------------

// GetNetworkEventDashboard builds aggregate statistics for the dashboard.
func GetNetworkEventDashboard(orgId int64) (NetworkEventDashboard, error) {
	dash := NetworkEventDashboard{}

	db.Model(&NetworkEvent{}).Where("org_id = ?", orgId).Count(&dash.TotalEvents)

	db.Model(&NetworkEvent{}).
		Where("org_id = ? AND status NOT IN (?)", orgId, []string{NetworkEventStatusResolved, NetworkEventStatusFalsePositive}).
		Count(&dash.OpenEvents)

	db.Model(&NetworkEvent{}).
		Where("org_id = ? AND severity = ? AND status NOT IN (?)", orgId, NetworkEventSeverityCritical, []string{NetworkEventStatusResolved, NetworkEventStatusFalsePositive}).
		Count(&dash.CriticalOpen)

	db.Model(&NetworkEvent{}).
		Where("org_id = ? AND severity = ? AND status NOT IN (?)", orgId, NetworkEventSeverityHigh, []string{NetworkEventStatusResolved, NetworkEventStatusFalsePositive}).
		Count(&dash.HighOpen)

	type avgRow struct{ AvgHours float64 }
	var ar avgRow
	db.Raw(`
SELECT COALESCE(AVG(
(julianday(resolved_date) - julianday(created_date)) * 24
), 0) as avg_hours
FROM network_events
WHERE org_id = ? AND status IN (?, ?)
AND resolved_date > created_date
`, orgId, NetworkEventStatusResolved, NetworkEventStatusFalsePositive).Scan(&ar)
	dash.AvgResolutionHours = ar.AvgHours

	db.Raw(`SELECT source, COUNT(*) as count FROM network_events WHERE org_id = ? GROUP BY source ORDER BY count DESC`, orgId).Scan(&dash.EventsBySource)
	db.Raw(`SELECT event_type, COUNT(*) as count FROM network_events WHERE org_id = ? GROUP BY event_type ORDER BY count DESC`, orgId).Scan(&dash.EventsByType)
	db.Raw(`SELECT severity, COUNT(*) as count FROM network_events WHERE org_id = ? GROUP BY severity ORDER BY count DESC`, orgId).Scan(&dash.EventsBySeverity)

	var recent []NetworkEvent
	db.Where("org_id = ?", orgId).Order("event_date DESC").Limit(10).Find(&recent)
	dash.RecentEvents = recent

	trend, err := GetNetworkEventTrend(orgId, 30)
	if err != nil {
		log.Errorf("network_event: failed to load trend data for org %d: %v", orgId, err)
	}
	dash.TrendData = trend

	return dash, nil
}

// GetNetworkEventTrend returns daily event counts for the last N days.
func GetNetworkEventTrend(orgId int64, days int) ([]DailyEventCount, error) {
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	var counts []DailyEventCount
	err := db.Raw(`
SELECT DATE(event_date) as date, COUNT(*) as count
FROM network_events
WHERE org_id = ? AND event_date >= ?
GROUP BY DATE(event_date)
ORDER BY date ASC
`, orgId, cutoff).Scan(&counts).Error
	if err != nil {
		return nil, err
	}
	return counts, nil
}

// ---------------------------------------------------------------------------
// Rules CRUD
// ---------------------------------------------------------------------------

// GetNetworkEventRules returns all automation rules for an organization.
func GetNetworkEventRules(orgId int64) ([]NetworkEventRule, error) {
	var rules []NetworkEventRule
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// PostNetworkEventRule creates a new automation rule.
func PostNetworkEventRule(rule *NetworkEventRule) error {
	now := time.Now().UTC()
	rule.CreatedDate = now
	rule.ModifiedDate = now
	wantEnabled := rule.Enabled
	wantIsPlaybook := rule.IsPlaybook
	err := db.Create(rule).Error
	if err != nil {
		log.Error(err)
		return err
	}
	if !wantEnabled {
		db.Exec("UPDATE network_event_rules SET enabled = 0 WHERE id = ?", rule.Id)
	}
	if wantIsPlaybook {
		db.Exec("UPDATE network_event_rules SET is_playbook = 1 WHERE id = ?", rule.Id)
	}
	return nil
}

// PutNetworkEventRule updates an existing automation rule.
func PutNetworkEventRule(rule *NetworkEventRule) error {
	rule.ModifiedDate = time.Now().UTC()
	err := db.Save(rule).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// DeleteNetworkEventRule removes an automation rule by ID within an org scope.
func DeleteNetworkEventRule(id, orgId int64) error {
	var rule NetworkEventRule
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&rule).Error
	if err != nil {
		return err
	}
	return db.Delete(&rule).Error
}

// ---------------------------------------------------------------------------
// MITRE ATT&CK Heatmap
// ---------------------------------------------------------------------------

// GetMitreHeatmap builds a heatmap of observed MITRE techniques for an org.
func GetMitreHeatmap(orgId int64) (MitreHeatmapData, error) {
	type techCount struct {
		MitreTechniqueId string `gorm:"column:mitre_technique_id"`
		Count            int
	}

	var counts []techCount
	err := db.Raw(`
SELECT mitre_technique_id, COUNT(*) as count
FROM network_events
WHERE org_id = ? AND mitre_technique_id != '' AND mitre_technique_id IS NOT NULL
GROUP BY mitre_technique_id
ORDER BY count DESC
`, orgId).Scan(&counts).Error
	if err != nil {
		return MitreHeatmapData{}, err
	}

	// Build a lookup map: technique_id -> count
	techMap := map[string]int{}
	totalMapped := 0
	for _, tc := range counts {
		techMap[tc.MitreTechniqueId] = tc.Count
		totalMapped += tc.Count
	}

	// Count unmapped
	var totalAll int
	db.Model(&NetworkEvent{}).Where("org_id = ?", orgId).Count(&totalAll)
	totalUnmapped := totalAll - totalMapped

	// Build heatmap rows in canonical tactic order
	var rows []MitreHeatmapRow
	for _, tactic := range MitreTacticOrder {
		techniques, ok := MitreTacticTechnique[tactic]
		if !ok {
			continue
		}
		row := MitreHeatmapRow{Tactic: tactic}
		for _, tid := range techniques {
			cell := MitreHeatmapCell{
				TechniqueId:   tid,
				TechniqueName: MitreTechniqueNames[tid],
				Count:         techMap[tid],
			}
			row.Techniques = append(row.Techniques, cell)
		}
		rows = append(rows, row)
	}

	// Top 10 techniques
	topN := make([]MitreHeatmapCell, 0, 10)
	for _, tc := range counts {
		if len(topN) >= 10 {
			break
		}
		topN = append(topN, MitreHeatmapCell{
			TechniqueId:   tc.MitreTechniqueId,
			TechniqueName: MitreTechniqueNames[tc.MitreTechniqueId],
			Count:         tc.Count,
		})
	}

	return MitreHeatmapData{
		Rows:          rows,
		TopTechniques: topN,
		TotalMapped:   totalMapped,
		TotalUnmapped: totalUnmapped,
	}, nil
}

// ---------------------------------------------------------------------------
// Incident Correlation
// ---------------------------------------------------------------------------

// CorrelateNetworkEvents scans recent events and groups them into incidents
// based on the correlation rule: same source_ip + same user_email within 1 hour.
// Events already assigned to an incident are skipped.
func CorrelateNetworkEvents(orgId int64) (int, error) {
	// Fetch unassigned events from the last 24 hours
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	var events []NetworkEvent
	err := db.Where("org_id = ? AND incident_id = 0 AND event_date >= ?", orgId, cutoff).
		Order("event_date ASC").Find(&events).Error
	if err != nil {
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	// Group by (source_ip, user_email) key
	type groupKey struct {
		SourceIP  string
		UserEmail string
	}
	groups := map[groupKey][]NetworkEvent{}
	for _, e := range events {
		key := groupKey{SourceIP: e.SourceIP, UserEmail: e.UserEmail}
		// If both are empty, skip grouping
		if key.SourceIP == "" && key.UserEmail == "" {
			continue
		}
		groups[key] = append(groups[key], e)
	}

	incidentsCreated := 0

	for key, evts := range groups {
		if len(evts) < 2 {
			continue // need at least 2 events to form an incident
		}

		// Sort by event_date
		sort.Slice(evts, func(i, j int) bool {
			return evts[i].EventDate.Before(evts[j].EventDate)
		})

		// Sliding window: group events within 1 hour of each other
		var cluster []NetworkEvent
		cluster = append(cluster, evts[0])

		for i := 1; i < len(evts); i++ {
			lastInCluster := cluster[len(cluster)-1]
			if evts[i].EventDate.Sub(lastInCluster.EventDate) <= time.Hour {
				cluster = append(cluster, evts[i])
			} else {
				// Flush current cluster if big enough
				if len(cluster) >= 2 {
					if err := createIncidentFromCluster(orgId, key.SourceIP, key.UserEmail, cluster); err != nil {
						log.Errorf("network_event: incident creation failed: %v", err)
					} else {
						incidentsCreated++
					}
				}
				cluster = []NetworkEvent{evts[i]}
			}
		}
		// Flush remaining cluster
		if len(cluster) >= 2 {
			if err := createIncidentFromCluster(orgId, key.SourceIP, key.UserEmail, cluster); err != nil {
				log.Errorf("network_event: incident creation failed: %v", err)
			} else {
				incidentsCreated++
			}
		}
	}

	return incidentsCreated, nil
}

// createIncidentFromCluster creates an incident from a cluster of correlated events.
func createIncidentFromCluster(orgId int64, srcIP, userEmail string, events []NetworkEvent) error {
	now := time.Now().UTC()

	// Determine severity: highest among events
	sevOrder := map[string]int{"info": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}
	maxSev := "info"
	for _, e := range events {
		if sevOrder[e.Severity] > sevOrder[maxSev] {
			maxSev = e.Severity
		}
	}

	title := fmt.Sprintf("Correlated incident: %d events", len(events))
	if srcIP != "" && userEmail != "" {
		title = fmt.Sprintf("Incident from %s / %s (%d events)", srcIP, userEmail, len(events))
	} else if srcIP != "" {
		title = fmt.Sprintf("Incident from IP %s (%d events)", srcIP, len(events))
	} else if userEmail != "" {
		title = fmt.Sprintf("Incident for %s (%d events)", userEmail, len(events))
	}

	incident := NetworkIncident{
		OrgId:        orgId,
		Title:        title,
		Description:  fmt.Sprintf("Auto-correlated %d events within 1-hour window", len(events)),
		Severity:     maxSev,
		Status:       "open",
		SourceIP:     srcIP,
		UserEmail:    userEmail,
		EventCount:   len(events),
		FirstSeen:    events[0].EventDate,
		LastSeen:     events[len(events)-1].EventDate,
		CreatedDate:  now,
		ModifiedDate: now,
	}

	if err := db.Create(&incident).Error; err != nil {
		return err
	}

	// Link events to incident
	ids := make([]int64, len(events))
	for i, e := range events {
		ids[i] = e.Id
	}
	return db.Model(&NetworkEvent{}).Where("id IN (?)", ids).
		Updates(map[string]interface{}{"incident_id": incident.Id, "modified_date": now}).Error
}

// GetNetworkIncidents returns incidents for an org, with optional status filter.
func GetNetworkIncidents(orgId int64, status string, limit, offset int) ([]NetworkIncident, error) {
	if limit <= 0 {
		limit = 50
	}
	query := db.Where("org_id = ?", orgId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var incidents []NetworkIncident
	err := query.Order("last_seen DESC").Limit(limit).Offset(offset).Find(&incidents).Error
	if err != nil {
		return nil, err
	}
	return incidents, nil
}

// GetNetworkIncident returns a single incident with its events hydrated.
func GetNetworkIncident(id, orgId int64) (NetworkIncident, error) {
	var incident NetworkIncident
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&incident).Error
	if err != nil {
		return incident, ErrNetworkIncidentNotFound
	}
	// Hydrate events
	var events []NetworkEvent
	db.Where("incident_id = ? AND org_id = ?", id, orgId).Order("event_date ASC").Find(&events)
	incident.Events = events
	return incident, nil
}

// UpdateNetworkIncidentStatus updates an incident's status.
func UpdateNetworkIncidentStatus(id, orgId int64, status string) error {
	return db.Model(&NetworkIncident{}).Where("id = ? AND org_id = ?", id, orgId).
		Updates(map[string]interface{}{
			"status":        status,
			"modified_date": time.Now().UTC(),
		}).Error
}

// ---------------------------------------------------------------------------
// Playbook execution
// ---------------------------------------------------------------------------

// GetPlaybookExecutionLogs returns recent playbook execution logs for an org.
func GetPlaybookExecutionLogs(orgId int64, limit int) ([]PlaybookExecutionLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var logs []PlaybookExecutionLog
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// executePlaybookActions runs all actions defined in a playbook rule against an event.
func executePlaybookActions(rule NetworkEventRule, e *NetworkEvent) {
	var actions []PlaybookAction
	if err := json.Unmarshal([]byte(rule.PlaybookActions), &actions); err != nil {
		log.Errorf("network_event: failed to parse playbook actions for rule %d: %v", rule.Id, err)
		return
	}

	var executedActions []string

	for _, action := range actions {
		switch action.Type {
		case "set_severity":
			e.Severity = action.Value
			executedActions = append(executedActions, "set_severity="+action.Value)

		case "assign_to":
			// Expects a user ID as string
			var uid int64
			if _, err := fmt.Sscanf(action.Value, "%d", &uid); err == nil && uid > 0 {
				e.AssignedTo = uid
				executedActions = append(executedActions, fmt.Sprintf("assign_to=%d", uid))
			}

		case "add_note":
			// We'll add the note after the event is created (need event ID)
			// Store in raw payload for now, actual note added post-create
			executedActions = append(executedActions, "add_note=queued")

		case "create_remediation":
			executedActions = append(executedActions, "create_remediation=queued")

		case "notify_webhook":
			// Fire-and-forget webhook notification
			go firePlaybookWebhook(action.Value, e)
			executedActions = append(executedActions, "notify_webhook="+action.Value)

		case "create_incident":
			executedActions = append(executedActions, "create_incident=queued")

		default:
			executedActions = append(executedActions, "unknown_action="+action.Type)
		}
	}

	// Log execution
	actionsJSON, _ := json.Marshal(executedActions)
	logEntry := PlaybookExecutionLog{
		OrgId:       e.OrgId,
		RuleId:      rule.Id,
		EventId:     e.Id,
		RuleName:    rule.Name,
		ActionsRun:  string(actionsJSON),
		Status:      "success",
		CreatedDate: time.Now().UTC(),
	}
	if err := db.Create(&logEntry).Error; err != nil {
		log.Errorf("network_event: failed to log playbook execution: %v", err)
	}

	// Post-create actions that need the event ID
	for _, action := range actions {
		switch action.Type {
		case "add_note":
			if e.Id > 0 {
				note := NetworkEventNote{
					EventId:     e.Id,
					UserId:      0,
					Content:     fmt.Sprintf("[Playbook: %s] %s", rule.Name, action.Value),
					CreatedDate: time.Now().UTC(),
				}
				db.Create(&note)
			}
		}
	}
}

// firePlaybookWebhook sends a JSON payload to a webhook URL.
func firePlaybookWebhook(url string, e *NetworkEvent) {
	payload, err := json.Marshal(map[string]interface{}{
		"type":    "playbook_triggered",
		"event":   e,
		"message": fmt.Sprintf("Playbook triggered for event: %s (severity: %s)", e.Title, e.Severity),
	})
	if err != nil {
		log.Errorf("network_event: webhook marshal error: %v", err)
		return
	}
	resp, err := http.Post(url, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		log.Errorf("network_event: webhook POST to %s failed: %v", url, err)
		return
	}
	resp.Body.Close()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// applyNetworkEventRules loads all enabled rules for the event's org and
// applies auto-severity/auto-assign overrides and executes playbook actions.
func applyNetworkEventRules(e *NetworkEvent) {
	var rules []NetworkEventRule
	err := db.Where("org_id = ? AND enabled = ?", e.OrgId, true).Find(&rules).Error
	if err != nil {
		log.Errorf("network_event: failed to load rules for org %d: %v", e.OrgId, err)
		return
	}

	for _, rule := range rules {
		if !matchesRule(e, rule) {
			continue
		}

		if rule.IsPlaybook && rule.PlaybookActions != "" {
			executePlaybookActions(rule, e)
		} else {
			// Legacy rule behaviour
			if rule.AutoSeverity != "" {
				e.Severity = rule.AutoSeverity
			}
			if rule.AutoAssign > 0 {
				e.AssignedTo = rule.AutoAssign
			}
		}
	}
}

// matchesRule checks whether an event matches a rule's conditions.
func matchesRule(e *NetworkEvent, rule NetworkEventRule) bool {
	sourceOK := rule.SourceMatch == "" || matchesField(e.Source, rule.SourceMatch)
	typeOK := rule.EventTypeMatch == "" || matchesField(e.EventType, rule.EventTypeMatch)
	severityOK := rule.SeverityMatch == "" || matchesField(e.Severity, rule.SeverityMatch)
	return sourceOK && typeOK && severityOK
}

// matchesField performs a case-insensitive exact or substring match.
func matchesField(value, pattern string) bool {
	v := strings.ToLower(value)
	p := strings.ToLower(pattern)
	if v == p {
		return true
	}
	return strings.Contains(v, p)
}

// hydrateNetworkEvent loads associated notes onto the event.
func hydrateNetworkEvent(e *NetworkEvent) {
	var notes []NetworkEventNote
	err := db.Where("event_id = ?", e.Id).Order("created_date ASC").Find(&notes).Error
	if err != nil {
		log.Errorf("network_event: failed to load notes for event %d: %v", e.Id, err)
		return
	}
	e.Notes = notes
}
