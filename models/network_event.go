package models

import (
	"errors"
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

// NetworkEvent represents a security event ingested from an external source.
type NetworkEvent struct {
	Id            int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId         int64     `json:"org_id" gorm:"column:org_id"`
	Source        string    `json:"source"`
	EventType     string    `json:"event_type" gorm:"column:event_type"`
	Severity      string    `json:"severity" gorm:"default:'info'"`
	Title         string    `json:"title"`
	Description   string    `json:"description" gorm:"type:text"`
	SourceIP      string    `json:"source_ip" gorm:"column:source_ip"`
	DestinationIP string    `json:"destination_ip" gorm:"column:destination_ip"`
	UserId        int64     `json:"user_id" gorm:"column:user_id"`
	UserEmail     string    `json:"user_email" gorm:"column:user_email"`
	DeviceId      string    `json:"device_id" gorm:"column:device_id"`
	RawPayload    string    `json:"raw_payload" gorm:"column:raw_payload;type:text"`
	Status        string    `json:"status" gorm:"default:'new'"`
	AssignedTo    int64     `json:"assigned_to" gorm:"column:assigned_to"`
	ResolvedBy    int64     `json:"resolved_by" gorm:"column:resolved_by"`
	ResolvedDate  time.Time `json:"resolved_date" gorm:"column:resolved_date"`
	EventDate     time.Time `json:"event_date" gorm:"column:event_date"`
	CreatedDate   time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate  time.Time `json:"modified_date" gorm:"column:modified_date"`
	Notes         []NetworkEventNote `json:"notes,omitempty" gorm:"-"`
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

// NetworkEventRule defines an automation rule that can auto-assign severity
// or ownership when an incoming event matches certain criteria.
type NetworkEventRule struct {
	Id             int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId          int64     `json:"org_id" gorm:"column:org_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	SourceMatch    string    `json:"source_match" gorm:"column:source_match"`
	EventTypeMatch string    `json:"event_type_match" gorm:"column:event_type_match"`
	AutoSeverity   string    `json:"auto_severity" gorm:"column:auto_severity"`
	AutoAssign     int64     `json:"auto_assign" gorm:"column:auto_assign"`
	Enabled        bool      `json:"enabled" gorm:"default:true"`
	CreatedDate    time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate   time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// TableName overrides the default table name.
func (NetworkEventRule) TableName() string {
	return "network_event_rules"
}

// NetworkEventFilter holds optional filter parameters for listing events.
type NetworkEventFilter struct {
	Source    string    `json:"source"`
	EventType string   `json:"event_type"`
	Severity  string   `json:"severity"`
	Status    string   `json:"status"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	UserEmail string   `json:"user_email"`
	Limit     int      `json:"limit"`
	Offset    int      `json:"offset"`
}

// NetworkEventDashboard is the aggregate response for the dashboard view.
type NetworkEventDashboard struct {
	TotalEvents        int              `json:"total_events"`
	OpenEvents         int              `json:"open_events"`
	CriticalOpen       int              `json:"critical_open"`
	HighOpen           int              `json:"high_open"`
	AvgResolutionHours float64          `json:"avg_resolution_hours"`
	EventsBySource     []SourceCount    `json:"events_by_source"`
	EventsByType       []TypeCount      `json:"events_by_type"`
	EventsBySeverity   []SeverityCount  `json:"events_by_severity"`
	RecentEvents       []NetworkEvent   `json:"recent_events"`
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
// CRUD operations
// ---------------------------------------------------------------------------

// PostNetworkEvent creates a new network event, applying any matching
// automation rules before persisting.
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

// UpdateNetworkEventStatus transitions an event to a new status. If the
// status is "resolved" or "false_positive", the resolution metadata is
// recorded.
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

	// Total events
	db.Model(&NetworkEvent{}).Where("org_id = ?", orgId).Count(&dash.TotalEvents)

	// Open events (not resolved or false_positive)
	db.Model(&NetworkEvent{}).
		Where("org_id = ? AND status NOT IN (?)", orgId, []string{NetworkEventStatusResolved, NetworkEventStatusFalsePositive}).
		Count(&dash.OpenEvents)

	// Critical open
	db.Model(&NetworkEvent{}).
		Where("org_id = ? AND severity = ? AND status NOT IN (?)", orgId, NetworkEventSeverityCritical, []string{NetworkEventStatusResolved, NetworkEventStatusFalsePositive}).
		Count(&dash.CriticalOpen)

	// High open
	db.Model(&NetworkEvent{}).
		Where("org_id = ? AND severity = ? AND status NOT IN (?)", orgId, NetworkEventSeverityHigh, []string{NetworkEventStatusResolved, NetworkEventStatusFalsePositive}).
		Count(&dash.HighOpen)

	// Average resolution hours
	type avgRow struct {
		AvgHours float64
	}
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

	// Events by source
	db.Raw(`
		SELECT source, COUNT(*) as count
		FROM network_events WHERE org_id = ?
		GROUP BY source ORDER BY count DESC
	`, orgId).Scan(&dash.EventsBySource)

	// Events by type
	db.Raw(`
		SELECT event_type, COUNT(*) as count
		FROM network_events WHERE org_id = ?
		GROUP BY event_type ORDER BY count DESC
	`, orgId).Scan(&dash.EventsByType)

	// Events by severity
	db.Raw(`
		SELECT severity, COUNT(*) as count
		FROM network_events WHERE org_id = ?
		GROUP BY severity ORDER BY count DESC
	`, orgId).Scan(&dash.EventsBySeverity)

	// 10 most recent events
	var recent []NetworkEvent
	db.Where("org_id = ?", orgId).Order("event_date DESC").Limit(10).Find(&recent)
	dash.RecentEvents = recent

	// 30-day trend
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
	err := db.Create(rule).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// GORM v1 omits bool zero-values on Create, so the DB default (true)
	// may override the caller's intent. Explicitly sync the column.
	if !wantEnabled {
		db.Exec("UPDATE network_event_rules SET enabled = 0 WHERE id = ?", rule.Id)
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
// Internal helpers
// ---------------------------------------------------------------------------

// applyNetworkEventRules loads all enabled rules for the event's org and
// applies auto-severity and auto-assign overrides when the event matches.
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
		if rule.AutoSeverity != "" {
			e.Severity = rule.AutoSeverity
		}
		if rule.AutoAssign > 0 {
			e.AssignedTo = rule.AutoAssign
		}
	}
}

// matchesRule checks whether an event matches a rule's source and event type
// criteria. An empty match field is treated as a wildcard (matches everything).
func matchesRule(e *NetworkEvent, rule NetworkEventRule) bool {
	sourceOK := rule.SourceMatch == "" || matchesField(e.Source, rule.SourceMatch)
	typeOK := rule.EventTypeMatch == "" || matchesField(e.EventType, rule.EventTypeMatch)
	return sourceOK && typeOK
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
