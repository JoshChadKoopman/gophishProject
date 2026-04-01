package models

import (
	"encoding/json"
	"time"
)

// AuditLog records security-relevant actions on the platform.
// Every role change, user creation/deletion, and login event is logged here.
type AuditLog struct {
	ID             int64     `json:"id"`
	OrgId          int64     `json:"org_id" gorm:"column:org_id"`
	Timestamp      time.Time `json:"timestamp"`
	ActorID        int64     `json:"actor_id"`
	ActorUsername  string    `json:"actor_username"`
	Action         string    `json:"action"`
	TargetType     string    `json:"target_type,omitempty"`
	TargetID       int64     `json:"target_id,omitempty"`
	TargetUsername string    `json:"target_username,omitempty"`
	Details        string    `json:"details,omitempty"` // JSON-encoded map
	IPAddress      string    `json:"ip_address,omitempty"`
}

// Audit action constants for consistent log entries.
const (
	AuditActionRoleChange   = "role_change"
	AuditActionUserCreated  = "user_created"
	AuditActionUserDeleted  = "user_deleted"
	AuditActionUserLocked   = "user_locked"
	AuditActionUserUnlocked = "user_unlocked"
	AuditActionLoginSuccess = "login_success"
	AuditActionLoginFailed  = "login_failed"
	AuditActionMFAEnrolled  = "mfa_enrolled"
	AuditActionMFAVerified  = "mfa_verified"
	AuditActionMFAFailed    = "mfa_failed"
	AuditActionMFALockout   = "mfa_lockout"

	AuditActionTrainingAssigned     = "training_assigned"
	AuditActionTrainingAutoAssigned = "training_auto_assigned"
	AuditActionTrainingCompleted    = "training_completed"
	AuditActionCertificateIssued    = "certificate_issued"
)

// CreateAuditLog persists an audit log entry with the current UTC timestamp.
func CreateAuditLog(entry *AuditLog) error {
	entry.Timestamp = time.Now().UTC()
	return db.Save(entry).Error
}

// GetAuditLogs returns audit log entries in descending timestamp order.
// limit <= 0 defaults to 100; offset enables pagination.
func GetAuditLogs(scope OrgScope, limit, offset int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	logs := []AuditLog{}
	err := scopeQuery(db.Table("audit_logs"), scope).Order("timestamp desc").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// AuditLogResponse wraps audit logs with a total count for pagination.
type AuditLogResponse struct {
	Logs  []AuditLog `json:"logs"`
	Total int64      `json:"total"`
}

// GetAuditLogsFiltered returns filtered audit log entries with total count for pagination.
func GetAuditLogsFiltered(scope OrgScope, limit, offset int, action, actor, dateFrom, dateTo string) (AuditLogResponse, error) {
	if limit <= 0 {
		limit = 25
	}
	resp := AuditLogResponse{}
	query := scopeQuery(db.Table("audit_logs"), scope)

	if action != "" {
		query = query.Where("action = ?", action)
	}
	if actor != "" {
		query = query.Where("actor_username LIKE ?", "%"+actor+"%")
	}
	if dateFrom != "" {
		query = query.Where("timestamp >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("timestamp <= ?", dateTo)
	}

	// Get total count before pagination
	err := query.Count(&resp.Total).Error
	if err != nil {
		return resp, err
	}

	resp.Logs = []AuditLog{}
	err = query.Order("timestamp desc").
		Limit(limit).
		Offset(offset).
		Find(&resp.Logs).Error
	return resp, err
}

// LogRoleChange records a role change event. Called by the user API whenever
// a user's role_id is updated.
func LogRoleChange(actor User, target User, oldSlug, newSlug, ipAddr string) error {
	details, _ := json.Marshal(map[string]string{
		"old_role": oldSlug,
		"new_role": newSlug,
	})
	entry := &AuditLog{
		OrgId:          actor.OrgId,
		ActorID:        actor.Id,
		ActorUsername:  actor.Username,
		Action:         AuditActionRoleChange,
		TargetType:     "user",
		TargetID:       target.Id,
		TargetUsername: target.Username,
		Details:        string(details),
		IPAddress:      ipAddr,
	}
	return CreateAuditLog(entry)
}

// LogUserEvent records a user lifecycle event (created, deleted, locked, etc.).
func LogUserEvent(actor User, target User, action, ipAddr string) error {
	entry := &AuditLog{
		OrgId:          actor.OrgId,
		ActorID:        actor.Id,
		ActorUsername:  actor.Username,
		Action:         action,
		TargetType:     "user",
		TargetID:       target.Id,
		TargetUsername: target.Username,
		IPAddress:      ipAddr,
	}
	return CreateAuditLog(entry)
}
