package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// EscalationPolicy defines when and how repeat offenders are escalated.
type EscalationPolicy struct {
	Id               int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId            int64     `json:"org_id" gorm:"column:org_id"`
	Name             string    `json:"name" gorm:"column:name"`
	Level            int       `json:"level" gorm:"column:level"`
	FailThreshold    int       `json:"fail_threshold" gorm:"column:fail_threshold"`
	LookbackDays     int       `json:"lookback_days" gorm:"column:lookback_days"`
	Action           string    `json:"action" gorm:"column:action"`
	NotifyManager    bool      `json:"notify_manager" gorm:"column:notify_manager"`
	NotifyAdmin      bool      `json:"notify_admin" gorm:"column:notify_admin"`
	AssignTrainingId int64     `json:"assign_training_id" gorm:"column:assign_training_id"`
	IsActive         bool      `json:"is_active" gorm:"column:is_active"`
	CreatedDate      time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate     time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// Escalation action constants.
const (
	EscalationActionNotify          = "notify"
	EscalationActionTraining        = "mandatory_training"
	EscalationActionRestrictAccess  = "restrict_access"
	EscalationActionManagerEscalate = "manager_escalate"

	// queryWhereIDAndOrgID is the shared WHERE clause for id+org_id lookups.
	queryWhereIDAndOrgID = "id = ? AND org_id = ?"
)

// EscalationEvent records a triggered escalation for a user.
type EscalationEvent struct {
	Id           int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id"`
	PolicyId     int64     `json:"policy_id" gorm:"column:policy_id"`
	UserId       int64     `json:"user_id" gorm:"column:user_id"`
	UserEmail    string    `json:"user_email" gorm:"column:user_email"`
	Level        int       `json:"level" gorm:"column:level"`
	Action       string    `json:"action" gorm:"column:action"`
	FailCount    int       `json:"fail_count" gorm:"column:fail_count"`
	Details      string    `json:"details" gorm:"column:details;type:text"`
	Status       string    `json:"status" gorm:"column:status"`
	ResolvedBy   int64     `json:"resolved_by" gorm:"column:resolved_by"`
	ResolvedDate time.Time `json:"resolved_date" gorm:"column:resolved_date"`
	CreatedDate  time.Time `json:"created_date" gorm:"column:created_date"`

	// Populated at query time
	PolicyName string `json:"policy_name,omitempty" gorm:"-"`
	UserName   string `json:"user_name,omitempty" gorm:"-"`
}

// Escalation event status constants.
const (
	EscalationStatusOpen     = "open"
	EscalationStatusResolved = "resolved"
	EscalationStatusExpired  = "expired"
)

// RepeatOffender is a user who has failed multiple simulations.
type RepeatOffender struct {
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Department  string `json:"department"`
	FailCount   int    `json:"fail_count"`
	SubmitCount int    `json:"submit_count"`
	TotalSent   int    `json:"total_sent"`
	LastFail    string `json:"last_fail"`
	EscLevel    int    `json:"escalation_level"`
}

// GetEscalationPolicies returns all policies for an org, ordered by level.
func GetEscalationPolicies(orgId int64) ([]EscalationPolicy, error) {
	policies := []EscalationPolicy{}
	err := db.Where("org_id = ?", orgId).Order("level asc").Find(&policies).Error
	return policies, err
}

// GetEscalationPolicy returns a single policy by ID.
func GetEscalationPolicy(id, orgId int64) (EscalationPolicy, error) {
	p := EscalationPolicy{}
	err := db.Where(queryWhereIDAndOrgID, id, orgId).First(&p).Error
	return p, err
}

// PostEscalationPolicy creates a new escalation policy.
func PostEscalationPolicy(p *EscalationPolicy) error {
	p.CreatedDate = time.Now().UTC()
	p.ModifiedDate = p.CreatedDate
	return db.Save(p).Error
}

// PutEscalationPolicy updates an escalation policy.
func PutEscalationPolicy(p *EscalationPolicy) error {
	p.ModifiedDate = time.Now().UTC()
	return db.Table("escalation_policies").Where(queryWhereIDAndOrgID, p.Id, p.OrgId).Updates(map[string]interface{}{
		"name":               p.Name,
		"level":              p.Level,
		"fail_threshold":     p.FailThreshold,
		"lookback_days":      p.LookbackDays,
		"action":             p.Action,
		"notify_manager":     p.NotifyManager,
		"notify_admin":       p.NotifyAdmin,
		"assign_training_id": p.AssignTrainingId,
		"is_active":          p.IsActive,
		"modified_date":      p.ModifiedDate,
	}).Error
}

// DeleteEscalationPolicy deletes a policy.
func DeleteEscalationPolicy(id, orgId int64) error {
	return db.Where(queryWhereIDAndOrgID, id, orgId).Delete(EscalationPolicy{}).Error
}

// GetRepeatOffenders identifies users who have clicked/submitted in multiple simulations.
func GetRepeatOffenders(orgId int64, minFails int, lookbackDays int) ([]RepeatOffender, error) {
	cutoff := time.Now().AddDate(0, 0, -lookbackDays).Format("2006-01-02")

	var offenders []RepeatOffender
	err := db.Raw(`
		SELECT r.email, r.first_name, r.last_name,
			COALESCE(u.department, '') as department,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as fail_count,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END) as submit_count,
			COUNT(*) as total_sent,
			MAX(r.modified_date) as last_fail
		FROM results r
		JOIN campaigns c ON r.campaign_id = c.id
		LEFT JOIN users u ON r.email = u.email AND u.org_id = ?
		WHERE c.org_id = ? AND r.send_date >= ?
			AND r.status IN (?, ?)
		GROUP BY r.email, r.first_name, r.last_name, u.department
		HAVING fail_count >= ?
		ORDER BY fail_count DESC
	`, EventClicked, EventDataSubmit, EventDataSubmit,
		orgId, orgId, cutoff,
		EventClicked, EventDataSubmit,
		minFails).Scan(&offenders).Error

	if err != nil {
		return nil, err
	}

	// Determine escalation level for each offender
	policies, _ := GetEscalationPolicies(orgId)
	for i := range offenders {
		offenders[i].EscLevel = determineEscalationLevel(offenders[i].FailCount, policies)
	}

	return offenders, nil
}

// determineEscalationLevel finds the highest matching policy level for a fail count.
func determineEscalationLevel(failCount int, policies []EscalationPolicy) int {
	level := 0
	for _, p := range policies {
		if p.IsActive && failCount >= p.FailThreshold && p.Level > level {
			level = p.Level
		}
	}
	return level
}

// EvaluateAndEscalate checks all users against escalation policies and creates events.
func EvaluateAndEscalate(orgId int64) ([]EscalationEvent, error) {
	policies, err := GetEscalationPolicies(orgId)
	if err != nil {
		return nil, err
	}

	var events []EscalationEvent
	for _, policy := range policies {
		if !policy.IsActive {
			continue
		}

		policyEvents, err := evaluatePolicy(orgId, policy)
		if err != nil {
			log.Errorf("escalation policy %d: %v", policy.Id, err)
			continue
		}
		events = append(events, policyEvents...)
	}

	return events, nil
}

// evaluatePolicy evaluates a single escalation policy against all offenders and returns new events.
func evaluatePolicy(orgId int64, policy EscalationPolicy) ([]EscalationEvent, error) {
	offenders, err := GetRepeatOffenders(orgId, policy.FailThreshold, policy.LookbackDays)
	if err != nil {
		return nil, err
	}

	var events []EscalationEvent
	for _, offender := range offenders {
		event, ok := createEscalationEvent(orgId, policy, offender)
		if !ok {
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

// createEscalationEvent builds and persists a single escalation event for an offender.
// Returns the event and true on success, or a zero value and false if skipped/failed.
func createEscalationEvent(orgId int64, policy EscalationPolicy, offender RepeatOffender) (EscalationEvent, bool) {
	if hasOpenEscalation(orgId, offender.Email, policy.Level) {
		return EscalationEvent{}, false
	}

	event := EscalationEvent{
		OrgId:       orgId,
		PolicyId:    policy.Id,
		UserEmail:   offender.Email,
		Level:       policy.Level,
		Action:      policy.Action,
		FailCount:   offender.FailCount,
		Status:      EscalationStatusOpen,
		CreatedDate: time.Now().UTC(),
	}

	// Try to match to a platform user
	u, err := getUserByEmailAndOrg(offender.Email, orgId)
	if err == nil {
		event.UserId = u.Id
	}

	if err := db.Save(&event).Error; err != nil {
		log.Errorf("save escalation event: %v", err)
		return EscalationEvent{}, false
	}

	executeEscalationAction(policy, offender, event)
	return event, true
}

// hasOpenEscalation checks if there's already an unresolved escalation at this level.
func hasOpenEscalation(orgId int64, email string, level int) bool {
	var count int
	db.Table("escalation_events").
		Where("org_id = ? AND user_email = ? AND level = ? AND status = ?",
			orgId, email, level, EscalationStatusOpen).
		Count(&count)
	return count > 0
}

// executeEscalationAction performs the configured action for an escalation.
func executeEscalationAction(policy EscalationPolicy, offender RepeatOffender, event EscalationEvent) {
	switch policy.Action {
	case EscalationActionTraining:
		if policy.AssignTrainingId > 0 && event.UserId > 0 {
			if err := AutoAssignOnClick(offender.Email, policy.AssignTrainingId, 0); err != nil {
				log.Errorf("escalation auto-assign training: %v", err)
			}
		}
	case EscalationActionRestrictAccess:
		if event.UserId > 0 {
			// Flag the user for admin review (don't lock them out automatically)
			db.Table("escalation_events").Where("id = ?", event.Id).
				Update("details", "User flagged for access review due to repeated phishing failures")
		}
	}
	// Notifications (notify_manager, notify_admin) are handled by the webhook/notification
	// system — the escalation event itself triggers webhooks that admins can configure.
}

// getUserByEmailAndOrg finds a user by email within an org (internal helper).
func getUserByEmailAndOrg(email string, orgId int64) (User, error) {
	u := User{}
	err := db.Preload("Role").Where("email = ? AND org_id = ?", email, orgId).First(&u).Error
	return u, err
}

// GetEscalationEvents returns escalation events for an org with optional status filter.
func GetEscalationEvents(orgId int64, status string, limit int) ([]EscalationEvent, error) {
	events := []EscalationEvent{}
	q := db.Where("org_id = ?", orgId)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if limit <= 0 {
		limit = 100
	}
	err := q.Order("created_date desc").Limit(limit).Find(&events).Error
	if err != nil {
		return events, err
	}

	// Hydrate with policy names and user names
	for i := range events {
		p := EscalationPolicy{}
		if db.Where("id = ?", events[i].PolicyId).First(&p).Error == nil {
			events[i].PolicyName = p.Name
		}
		if events[i].UserId > 0 {
			u, err := GetUser(events[i].UserId)
			if err == nil {
				events[i].UserName = u.FirstName + " " + u.LastName
			}
		}
	}
	return events, nil
}

// ResolveEscalation marks an escalation event as resolved.
func ResolveEscalation(eventId, orgId, resolvedBy int64) error {
	return db.Table("escalation_events").Where(queryWhereIDAndOrgID, eventId, orgId).
		Updates(map[string]interface{}{
			"status":        EscalationStatusResolved,
			"resolved_by":   resolvedBy,
			"resolved_date": time.Now().UTC(),
		}).Error
}

// GetEscalationSummary returns counts by status for the dashboard.
type EscalationSummary struct {
	OpenCount      int     `json:"open_count"`
	ResolvedCount  int     `json:"resolved_count"`
	TotalOffenders int     `json:"total_offenders"`
	AvgFailCount   float64 `json:"avg_fail_count"`
}

func GetEscalationSummary(orgId int64) (EscalationSummary, error) {
	s := EscalationSummary{}
	db.Table("escalation_events").Where("org_id = ? AND status = ?", orgId, EscalationStatusOpen).Count(&s.OpenCount)
	db.Table("escalation_events").Where("org_id = ? AND status = ?", orgId, EscalationStatusResolved).Count(&s.ResolvedCount)

	// Count distinct offenders with open escalations
	db.Raw(`SELECT COUNT(DISTINCT user_email) FROM escalation_events WHERE org_id = ? AND status = ?`,
		orgId, EscalationStatusOpen).Row().Scan(&s.TotalOffenders)

	// Average fail count of open escalations
	db.Raw(`SELECT COALESCE(AVG(fail_count), 0) FROM escalation_events WHERE org_id = ? AND status = ?`,
		orgId, EscalationStatusOpen).Row().Scan(&s.AvgFailCount)

	return s, nil
}
