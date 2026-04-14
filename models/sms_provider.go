package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// SMSProvider contains the attributes needed to send SMS messages via an
// external provider such as Twilio.
type SMSProvider struct {
	Id           int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId       int64     `json:"-" gorm:"column:user_id"`
	OrgId        int64     `json:"-" gorm:"column:org_id"`
	Name         string    `json:"name"`
	ProviderType string    `json:"provider_type" gorm:"column:provider_type"`
	AccountSid   string    `json:"account_sid" gorm:"column:account_sid"`
	AuthToken    string    `json:"auth_token,omitempty" gorm:"column:auth_token"`
	FromNumber   string    `json:"from_number" gorm:"column:from_number"`
	ModifiedDate time.Time `json:"modified_date"`
}

// TableName specifies the database table for Gorm.
func (sp SMSProvider) TableName() string {
	return "sms_providers"
}

// ErrSMSProviderNameNotSpecified is thrown when no name is given
var ErrSMSProviderNameNotSpecified = errors.New("SMS provider name not specified")

// ErrSMSAccountSidNotSpecified is thrown when the account SID is blank
var ErrSMSAccountSidNotSpecified = errors.New("SMS account SID not specified")

// ErrSMSAuthTokenNotSpecified is thrown when the auth token is blank
var ErrSMSAuthTokenNotSpecified = errors.New("SMS auth token not specified")

// ErrSMSFromNumberNotSpecified is thrown when no from number is given
var ErrSMSFromNumberNotSpecified = errors.New("SMS from number not specified")

// ErrSMSProviderNotFound indicates the SMS provider doesn't exist
var ErrSMSProviderNotFound = errors.New("SMS provider not found")

// phoneRegex is a simple check for phone number format
var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)

// Validate checks the SMSProvider fields for required values.
func (sp *SMSProvider) Validate() error {
	switch {
	case sp.Name == "":
		return ErrSMSProviderNameNotSpecified
	case sp.AccountSid == "":
		return ErrSMSAccountSidNotSpecified
	case sp.AuthToken == "":
		return ErrSMSAuthTokenNotSpecified
	case sp.FromNumber == "":
		return ErrSMSFromNumberNotSpecified
	}
	return nil
}

// GetSMSProviders returns all SMS providers for the given scope.
func GetSMSProviders(scope OrgScope) ([]SMSProvider, error) {
	ps := []SMSProvider{}
	query := db.Where("user_id=?", scope.UserId)
	if scope.OrgId > 0 {
		query = query.Where("org_id=?", scope.OrgId)
	}
	err := query.Find(&ps).Error
	return ps, err
}

// GetSMSProvider returns the SMS provider with the given id.
func GetSMSProvider(id int64, scope OrgScope) (SMSProvider, error) {
	sp := SMSProvider{}
	query := db.Where("id=? AND user_id=?", id, scope.UserId)
	if scope.OrgId > 0 {
		query = query.Where("org_id=?", scope.OrgId)
	}
	err := query.First(&sp).Error
	return sp, err
}

// GetSMSProviderByName returns the SMS provider with the given name.
func GetSMSProviderByName(name string, scope OrgScope) (SMSProvider, error) {
	sp := SMSProvider{}
	query := db.Where("name=? AND user_id=?", name, scope.UserId)
	if scope.OrgId > 0 {
		query = query.Where("org_id=?", scope.OrgId)
	}
	err := query.First(&sp).Error
	return sp, err
}

// PostSMSProvider creates a new SMS provider in the database.
func PostSMSProvider(sp *SMSProvider, scope OrgScope) error {
	err := sp.Validate()
	if err != nil {
		return err
	}
	sp.UserId = scope.UserId
	sp.OrgId = scope.OrgId
	sp.ModifiedDate = time.Now().UTC()
	return db.Save(sp).Error
}

// PutSMSProvider updates an existing SMS provider in the database.
func PutSMSProvider(sp *SMSProvider, scope OrgScope) error {
	err := sp.Validate()
	if err != nil {
		return err
	}
	// If the auth token is blank, keep the existing one
	if sp.AuthToken == "" {
		existing, err := GetSMSProvider(sp.Id, scope)
		if err != nil {
			return err
		}
		sp.AuthToken = existing.AuthToken
	}
	sp.UserId = scope.UserId
	sp.OrgId = scope.OrgId
	sp.ModifiedDate = time.Now().UTC()
	return db.Save(sp).Error
}

// DeleteSMSProvider deletes an SMS provider by id.
func DeleteSMSProvider(id int64, scope OrgScope) error {
	sp, err := GetSMSProvider(id, scope)
	if err != nil {
		return err
	}
	return db.Delete(&sp).Error
}

// SendSMS sends a single SMS message using the provider's API.
// Currently supports Twilio.
func (sp *SMSProvider) SendSMS(to string, body string) error {
	switch strings.ToLower(sp.ProviderType) {
	case "twilio", "":
		return sp.sendTwilio(to, body)
	default:
		return fmt.Errorf("unsupported SMS provider type: %s", sp.ProviderType)
	}
}

// twilioResponse captures relevant fields from Twilio's API response.
type twilioResponse struct {
	Sid          string `json:"sid"`
	Status       string `json:"status"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// sendTwilio sends an SMS via the Twilio REST API.
func (sp *SMSProvider) sendTwilio(to string, body string) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", sp.AccountSid)

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", sp.FromNumber)
	data.Set("Body", body)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(sp.AccountSid, sp.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("twilio request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading twilio response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var tr twilioResponse
		if json.Unmarshal(respBody, &tr) == nil && tr.ErrorMessage != "" {
			return fmt.Errorf("twilio error %d: %s", tr.ErrorCode, tr.ErrorMessage)
		}
		return fmt.Errorf("twilio HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	log.Infof("SMS sent to %s via Twilio (status: %d)", to, resp.StatusCode)
	return nil
}

// GenerateSMSBody creates the SMS message body by executing the template
// with the phishing context. The body should include the phishing URL.
func GenerateSMSBody(templateText string, ptx PhishingTemplateContext) (string, error) {
	var buf bytes.Buffer
	result, err := ExecuteTemplate(templateText, ptx)
	if err != nil {
		return "", err
	}
	buf.WriteString(result)
	return buf.String(), nil
}

// ValidatePhone checks if a phone number looks valid for SMS.
func ValidatePhone(phone string) bool {
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	return phoneRegex.MatchString(cleaned)
}

// CleanPhone removes common formatting from phone numbers.
func CleanPhone(phone string) string {
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	if !strings.HasPrefix(cleaned, "+") {
		cleaned = "+" + cleaned
	}
	return cleaned
}

// GetSMSProviderInternal returns an SMS provider without scope checks (for worker use).
func GetSMSProviderInternal(id int64) (SMSProvider, error) {
	sp := SMSProvider{}
	err := db.Where("id=?", id).First(&sp).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return sp, ErrSMSProviderNotFound
		}
		return sp, err
	}
	return sp, nil
}
