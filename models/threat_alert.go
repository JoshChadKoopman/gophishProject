package models

import (
	"time"
)

// queryWhereAlertID is the shared WHERE clause fragment for alert_id lookups.
const queryWhereAlertID = "alert_id = ?"

// ThreatAlert represents a security alert/article created by admins and
// targeted at specific roles or departments within an organization.
type ThreatAlert struct {
	Id                int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId             int64     `json:"org_id"`
	Title             string    `json:"title"`
	Body              string    `json:"body"`
	Severity          string    `json:"severity" gorm:"default:'info'"`
	TargetRoles       string    `json:"target_roles" gorm:"type:text"`
	TargetDepartments string    `json:"target_departments" gorm:"type:text"`
	Published         bool      `json:"published" gorm:"default:false"`
	PublishedDate     time.Time `json:"published_date"`
	CreatedBy         int64     `json:"created_by"`
	CreatedDate       time.Time `json:"created_date"`
	ModifiedDate      time.Time `json:"modified_date"`
	ReadCount         int64     `json:"read_count" gorm:"-"`
	IsRead            bool      `json:"is_read" gorm:"-"`
}

// TableName overrides the default table name.
func (ThreatAlert) TableName() string {
	return "threat_alerts"
}

// ThreatAlertRead tracks which users have read which alerts.
type ThreatAlertRead struct {
	Id       int64     `json:"id" gorm:"primary_key;auto_increment"`
	AlertId  int64     `json:"alert_id"`
	UserId   int64     `json:"user_id"`
	ReadDate time.Time `json:"read_date"`
}

// TableName overrides the default table name.
func (ThreatAlertRead) TableName() string {
	return "threat_alert_reads"
}

// CreateThreatAlert saves a new threat alert.
func CreateThreatAlert(alert *ThreatAlert) error {
	alert.CreatedDate = time.Now().UTC()
	alert.ModifiedDate = time.Now().UTC()
	if alert.Published {
		alert.PublishedDate = time.Now().UTC()
	}
	return db.Create(alert).Error
}

// UpdateThreatAlert updates an existing threat alert.
func UpdateThreatAlert(alert *ThreatAlert) error {
	alert.ModifiedDate = time.Now().UTC()
	if alert.Published && alert.PublishedDate.IsZero() {
		alert.PublishedDate = time.Now().UTC()
	}
	return db.Save(alert).Error
}

// GetThreatAlerts returns all alerts for an org (admin view), newest first.
func GetThreatAlerts(orgId int64) ([]ThreatAlert, error) {
	var alerts []ThreatAlert
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Find(&alerts).Error
	if err != nil {
		return nil, err
	}
	// Populate read counts
	for i := range alerts {
		db.Model(&ThreatAlertRead{}).Where(queryWhereAlertID, alerts[i].Id).Count(&alerts[i].ReadCount)
	}
	return alerts, nil
}

// GetPublishedThreatAlerts returns published alerts for an org (user view), newest first.
func GetPublishedThreatAlerts(orgId int64, userId int64) ([]ThreatAlert, error) {
	var alerts []ThreatAlert
	err := db.Where("org_id = ? AND published = ?", orgId, true).
		Order("published_date DESC").Find(&alerts).Error
	if err != nil {
		return nil, err
	}
	// Mark which alerts the user has read
	for i := range alerts {
		var count int64
		db.Model(&ThreatAlertRead{}).Where("alert_id = ? AND user_id = ?", alerts[i].Id, userId).Count(&count)
		alerts[i].IsRead = count > 0
	}
	return alerts, nil
}

// GetThreatAlert returns a single threat alert by ID within an org scope.
func GetThreatAlert(id int64, orgId int64) (ThreatAlert, error) {
	var alert ThreatAlert
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&alert).Error
	if err != nil {
		return alert, err
	}
	db.Model(&ThreatAlertRead{}).Where(queryWhereAlertID, alert.Id).Count(&alert.ReadCount)
	return alert, nil
}

// DeleteThreatAlert removes a threat alert and its read records.
func DeleteThreatAlert(id int64, orgId int64) error {
	alert, err := GetThreatAlert(id, orgId)
	if err != nil {
		return err
	}
	db.Where(queryWhereAlertID, alert.Id).Delete(&ThreatAlertRead{})
	return db.Delete(&alert).Error
}

// MarkThreatAlertRead marks a threat alert as read by a user.
// Uses INSERT OR IGNORE / INSERT IGNORE to handle duplicate reads gracefully.
func MarkThreatAlertRead(alertId int64, userId int64) error {
	var count int64
	db.Model(&ThreatAlertRead{}).Where("alert_id = ? AND user_id = ?", alertId, userId).Count(&count)
	if count > 0 {
		return nil
	}
	read := ThreatAlertRead{
		AlertId:  alertId,
		UserId:   userId,
		ReadDate: time.Now().UTC(),
	}
	return db.Create(&read).Error
}

// GetUnreadThreatAlertCount returns the number of published alerts the user hasn't read.
func GetUnreadThreatAlertCount(orgId int64, userId int64) int64 {
	var total int64
	db.Model(&ThreatAlert{}).Where("org_id = ? AND published = ?", orgId, true).Count(&total)
	var read int64
	db.Table("threat_alert_reads").
		Joins("JOIN threat_alerts ON threat_alerts.id = threat_alert_reads.alert_id").
		Where("threat_alerts.org_id = ? AND threat_alerts.published = ? AND threat_alert_reads.user_id = ?", orgId, true, userId).
		Count(&read)
	unread := total - read
	if unread < 0 {
		return 0
	}
	return unread
}
