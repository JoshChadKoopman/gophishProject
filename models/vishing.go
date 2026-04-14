package models

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Vishing (Voice Phishing) Simulations ────────────────────────
// Complete data model for voice phishing simulations including scenarios,
// campaigns, results, and integration with the BRS/adaptive engine.

// CampaignTypeVishing is the campaign type for voice phishing.
const CampaignTypeVishing = "vishing"

// Vishing call outcome statuses
const (
	VishingStatusPending      = "pending"       // Call not yet placed
	VishingStatusDialing      = "dialing"       // Call in progress
	VishingStatusNoAnswer     = "no_answer"     // Target didn't answer
	VishingStatusBusy         = "busy"          // Line was busy
	VishingStatusAnswered     = "answered"       // Target answered, didn't fall for it
	VishingStatusEngaged      = "engaged"        // Target engaged with the IVR/script
	VishingStatusCredGiven    = "cred_given"     // Target provided credentials/info
	VishingStatusReported     = "reported"       // Target reported the call as suspicious
	VishingStatusHungUp       = "hung_up"        // Target hung up during the call
	VishingStatusVoicemail    = "voicemail"      // Call went to voicemail
	VishingStatusFailed       = "failed"         // Technical failure
)

// VishingScenario is a reusable voice phishing script/scenario.
type VishingScenario struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	OrgId           int64     `json:"org_id"`
	UserId          int64     `json:"user_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`         // e.g. "IT Support", "Bank", "Executive Assistant"
	DifficultyLevel int       `json:"difficulty_level"`  // 1-4
	Language        string    `json:"language"`          // ISO code
	CallerIdName    string    `json:"caller_id_name"`    // Spoofed caller ID name
	CallerIdNumber  string    `json:"caller_id_number"`  // Spoofed caller ID number
	Script          string    `json:"script"`            // JSON-encoded IVR script (TwiML or generic)
	ScriptType      string    `json:"script_type"`       // "twiml", "text", "ai_conversational"
	Greeting        string    `json:"greeting"`          // Initial greeting text/audio URL
	Pretext         string    `json:"pretext"`           // The social engineering premise
	SuccessCriteria string    `json:"success_criteria"`  // What constitutes a "fail" (info disclosure)
	RecordingEnabled bool     `json:"recording_enabled"` // Whether to record calls (requires consent)
	ConsentMessage  string    `json:"consent_message"`   // Pre-call consent notice
	MaxDurationSec  int       `json:"max_duration_sec"`  // Max call duration before auto-hangup
	CreatedDate     time.Time `json:"created_date"`
	ModifiedDate    time.Time `json:"modified_date"`
}

func (VishingScenario) TableName() string { return "vishing_scenarios" }

// VishingCampaign is a voice phishing campaign targeting a set of users.
type VishingCampaign struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	OrgId           int64     `json:"org_id"`
	UserId          int64     `json:"user_id"` // creator
	Name            string    `json:"name"`
	Status          string    `json:"status"` // Created, In progress, Completed
	ScenarioId      int64     `json:"scenario_id"`
	GroupIds        string    `json:"group_ids"`         // JSON array of target group IDs
	TelephonyProvider string  `json:"telephony_provider"` // "twilio", "vonage"
	SMSProviderId   int64     `json:"sms_provider_id"`    // Reuse SMS provider for credentials
	ScheduleStart   time.Time `json:"schedule_start"`
	ScheduleEnd     time.Time `json:"schedule_end"`
	ActiveHoursStart int      `json:"active_hours_start"` // Don't call before this hour
	ActiveHoursEnd   int      `json:"active_hours_end"`   // Don't call after this hour
	Timezone        string    `json:"timezone"`
	RetryAttempts   int       `json:"retry_attempts"`    // Retry on no-answer (max 3)
	LaunchDate      time.Time `json:"launch_date"`
	CompletedDate   time.Time `json:"completed_date"`
	CreatedDate     time.Time `json:"created_date"`
	ModifiedDate    time.Time `json:"modified_date"`
	// Hydrated
	Scenario *VishingScenario `json:"scenario,omitempty" gorm:"-"`
	Results  []VishingResult  `json:"results,omitempty" gorm:"-"`
}

func (VishingCampaign) TableName() string { return "vishing_campaigns" }

// VishingResult tracks the outcome of a single vishing call to a target.
type VishingResult struct {
	Id             int64     `json:"id" gorm:"primary_key"`
	CampaignId     int64     `json:"campaign_id"`
	OrgId          int64     `json:"org_id"`
	Email          string    `json:"email"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	PhoneNumber    string    `json:"phone_number"`
	Status         string    `json:"status"`           // VishingStatus* constants
	CallSid        string    `json:"call_sid"`          // Provider's call ID
	CallDuration   int       `json:"call_duration_sec"` // Duration in seconds
	IVRPath        string    `json:"ivr_path"`          // JSON array of DTMF inputs / script steps taken
	InfoDisclosed  string    `json:"info_disclosed"`    // JSON of what info was given (password, PIN, etc.)
	RecordingURL   string    `json:"recording_url"`     // URL to call recording (if enabled + consented)
	AttemptCount   int       `json:"attempt_count"`     // Which attempt this is (1-based)
	Reported       bool      `json:"reported"`          // Did the target report this call
	ReportedDate   time.Time `json:"reported_date"`
	SendDate       time.Time `json:"send_date"`         // When the call was placed
	CompletedDate  time.Time `json:"completed_date"`
	CreatedDate    time.Time `json:"created_date"`
	ModifiedDate   time.Time `json:"modified_date"`
}

func (VishingResult) TableName() string { return "vishing_results" }

// VishingCampaignStats provides aggregate stats for a vishing campaign.
type VishingCampaignStats struct {
	Total          int64   `json:"total"`
	Called         int64   `json:"called"`
	Answered       int64   `json:"answered"`
	Engaged        int64   `json:"engaged"`
	CredGiven      int64   `json:"cred_given"`
	Reported       int64   `json:"reported"`
	NoAnswer       int64   `json:"no_answer"`
	HungUp         int64   `json:"hung_up"`
	AnswerRate     float64 `json:"answer_rate"`
	EngagementRate float64 `json:"engagement_rate"`
	FailRate       float64 `json:"fail_rate"`      // % that gave info
	ReportRate     float64 `json:"report_rate"`
	AvgCallDuration float64 `json:"avg_call_duration_sec"`
}

// Shared query for vishing id+org lookups.
const queryVishIdOrg = "id = ? AND org_id = ?"

// ── CRUD Operations ─────────────────────────────────────────────

// --- Scenarios ---

// GetVishingScenarios returns all vishing scenarios for an org.
func GetVishingScenarios(orgId int64) ([]VishingScenario, error) {
	var scenarios []VishingScenario
	err := db.Where(queryWhereOrgID, orgId).Order("created_date DESC").Find(&scenarios).Error
	return scenarios, err
}

// GetVishingScenario returns a single scenario by ID.
func GetVishingScenario(id, orgId int64) (VishingScenario, error) {
	var s VishingScenario
	err := db.Where(queryVishIdOrg, id, orgId).First(&s).Error
	return s, err
}

// PostVishingScenario creates a new vishing scenario.
func PostVishingScenario(s *VishingScenario) error {
	if s.Name == "" {
		return fmt.Errorf("scenario name is required")
	}
	if s.MaxDurationSec <= 0 {
		s.MaxDurationSec = 120 // 2 minutes default
	}
	if s.Language == "" {
		s.Language = "en"
	}
	s.CreatedDate = time.Now().UTC()
	s.ModifiedDate = time.Now().UTC()
	return db.Save(s).Error
}

// PutVishingScenario updates a scenario.
func PutVishingScenario(s *VishingScenario) error {
	s.ModifiedDate = time.Now().UTC()
	return db.Save(s).Error
}

// DeleteVishingScenario removes a scenario.
func DeleteVishingScenario(id, orgId int64) error {
	return db.Where(queryVishIdOrg, id, orgId).Delete(&VishingScenario{}).Error
}

// --- Campaigns ---

// GetVishingCampaigns returns all vishing campaigns for an org.
func GetVishingCampaigns(orgId int64) ([]VishingCampaign, error) {
	var campaigns []VishingCampaign
	err := db.Where(queryWhereOrgID, orgId).Order("created_date DESC").Find(&campaigns).Error
	return campaigns, err
}

// GetVishingCampaign returns a single vishing campaign with results.
func GetVishingCampaign(id, orgId int64) (VishingCampaign, error) {
	var c VishingCampaign
	err := db.Where(queryVishIdOrg, id, orgId).First(&c).Error
	if err != nil {
		return c, err
	}

	// Hydrate scenario
	scenario, _ := GetVishingScenario(c.ScenarioId, orgId)
	c.Scenario = &scenario

	// Hydrate results
	db.Where("campaign_id = ?", c.Id).Find(&c.Results)

	return c, nil
}

// PostVishingCampaign creates a new vishing campaign.
func PostVishingCampaign(c *VishingCampaign) error {
	if c.Name == "" {
		return fmt.Errorf("campaign name is required")
	}
	if c.ScenarioId == 0 {
		return fmt.Errorf("scenario is required")
	}
	if c.ActiveHoursStart == 0 && c.ActiveHoursEnd == 0 {
		c.ActiveHoursStart = 9
		c.ActiveHoursEnd = 17
	}
	if c.Timezone == "" {
		c.Timezone = "UTC"
	}
	if c.RetryAttempts > 3 {
		c.RetryAttempts = 3
	}
	c.Status = CampaignCreated
	c.CreatedDate = time.Now().UTC()
	c.ModifiedDate = time.Now().UTC()
	return db.Save(c).Error
}

// DeleteVishingCampaign removes a vishing campaign and its results.
func DeleteVishingCampaign(id, orgId int64) error {
	db.Where("campaign_id = ? AND org_id = ?", id, orgId).Delete(&VishingResult{})
	return db.Where(queryVishIdOrg, id, orgId).Delete(&VishingCampaign{}).Error
}

// --- Results ---

// GetVishingCampaignStats computes aggregate stats for a campaign.
func GetVishingCampaignStats(campaignId int64) VishingCampaignStats {
	var stats VishingCampaignStats
	var results []VishingResult
	db.Where("campaign_id = ?", campaignId).Find(&results)

	stats.Total = int64(len(results))
	if stats.Total == 0 {
		return stats
	}

	var totalDuration int64
	for _, r := range results {
		totalDuration += classifyVishingResult(r, &stats)
	}

	computeVishingRates(&stats, totalDuration)
	return stats
}

// classifyVishingResult increments stats counters and returns call duration.
func classifyVishingResult(r VishingResult, stats *VishingCampaignStats) int64 {
	var dur int64
	switch r.Status {
	case VishingStatusAnswered, VishingStatusEngaged, VishingStatusCredGiven, VishingStatusHungUp:
		stats.Answered++
		dur = int64(r.CallDuration)
	case VishingStatusNoAnswer, VishingStatusVoicemail:
		stats.NoAnswer++
	}
	if r.Status == VishingStatusEngaged || r.Status == VishingStatusCredGiven {
		stats.Engaged++
	}
	if r.Status == VishingStatusCredGiven {
		stats.CredGiven++
	}
	if r.Reported {
		stats.Reported++
	}
	if r.Status == VishingStatusHungUp {
		stats.HungUp++
	}
	if r.Status != VishingStatusPending && r.Status != VishingStatusFailed {
		stats.Called++
	}
	return dur
}

// computeVishingRates calculates rate percentages from raw counts.
func computeVishingRates(stats *VishingCampaignStats, totalDuration int64) {
	total := float64(stats.Total)
	if stats.Called > 0 {
		calledF := float64(stats.Called)
		stats.AnswerRate = float64(stats.Answered) / calledF * 100
		stats.EngagementRate = float64(stats.Engaged) / calledF * 100
		stats.FailRate = float64(stats.CredGiven) / calledF * 100
	}
	stats.ReportRate = float64(stats.Reported) / total * 100
	if stats.Answered > 0 {
		stats.AvgCallDuration = float64(totalDuration) / float64(stats.Answered)
	}
}

// RecordVishingResult saves or updates a vishing call result.
func RecordVishingResult(r *VishingResult) error {
	r.ModifiedDate = time.Now().UTC()
	if r.CreatedDate.IsZero() {
		r.CreatedDate = time.Now().UTC()
	}
	return db.Save(r).Error
}

// GetVishingTargetGroupIds parses the JSON group IDs from a campaign.
func (c *VishingCampaign) GetTargetGroupIds() []int64 {
	var ids []int64
	if c.GroupIds != "" {
		json.Unmarshal([]byte(c.GroupIds), &ids)
	}
	return ids
}

// ── BRS Integration ─────────────────────────────────────────────
// Vishing results feed into the BRS calculation the same way email
// phishing results do.

// ApplyVishingBRSPenalty applies a BRS penalty when a user fails a vishing test.
func ApplyVishingBRSPenalty(userId int64, status string) {
	penalty := 0.0
	switch status {
	case VishingStatusCredGiven:
		penalty = -8.0 // Equivalent to submitting credentials in email phishing
	case VishingStatusEngaged:
		penalty = -3.0 // Engaged but didn't give info
	case VishingStatusReported:
		// Reward for reporting
		var currentBRS UserRiskScoreRecord
		if err := db.Where(queryWhereUserID, userId).First(&currentBRS).Error; err == nil {
			newTrend := currentBRS.TrendScore + 4.0
			if newTrend > 100 {
				newTrend = 100
			}
			db.Model(&UserRiskScoreRecord{}).Where(queryWhereUserID, userId).
				Update("trend_score", newTrend)
		}
		return
	default:
		return // No penalty for answered-but-not-engaged, hung up, etc.
	}

	var currentBRS UserRiskScoreRecord
	if err := db.Where(queryWhereUserID, userId).First(&currentBRS).Error; err == nil {
		newSim := currentBRS.SimulationScore + penalty
		if newSim < 0 {
			newSim = 0
		}
		db.Model(&UserRiskScoreRecord{}).Where(queryWhereUserID, userId).
			Update("simulation_score", newSim)
		log.Infof("vishing: applied %.1f BRS penalty to user %d (status=%s)", penalty, userId, status)
	}
}

// ── Built-in Vishing Scenario Library ───────────────────────────

// VishingScenarioLibrary contains pre-built vishing scenarios.
var VishingScenarioLibrary = []VishingScenario{
	{
		Name:            "IT Support Password Reset",
		Category:        "IT Support",
		DifficultyLevel: 1,
		Language:        "en",
		CallerIdName:    "IT Help Desk",
		Pretext:         "IT support calling about a mandatory password reset due to a security incident.",
		Greeting:        "Hello, this is the IT help desk at {{.OrgName}}. We're calling about a mandatory password reset due to a detected security incident on your account.",
		SuccessCriteria: "Target provides their current password or creates a new one over the phone.",
		Script:          `{"steps":[{"type":"say","text":"Hello, this is the IT help desk at your company. We detected suspicious activity on your account and need to verify your identity. Can you please confirm your current password so we can reset it securely?"},{"type":"gather","input":"speech","timeout":10},{"type":"say","text":"Thank you for that information. We'll process the reset now. You'll receive an email confirmation shortly."}]}`,
		ScriptType:      "text",
		MaxDurationSec:  120,
		ConsentMessage:  "This call may be monitored for quality and training purposes.",
	},
	{
		Name:            "Bank Fraud Alert",
		Category:        "Financial",
		DifficultyLevel: 2,
		Language:        "en",
		CallerIdName:    "Fraud Detection",
		Pretext:         "Bank fraud department calling about a suspicious transaction on the user's account.",
		Greeting:        "Hello, this is the fraud detection department. We've identified a suspicious transaction of €2,847 on your account ending in 4521.",
		SuccessCriteria: "Target provides account number, PIN, or other banking credentials.",
		Script:          `{"steps":[{"type":"say","text":"Hello, this is the fraud detection department. We've identified a suspicious transaction of two thousand eight hundred forty-seven euros on your account. To verify your identity, can you please confirm your account number and the last four digits of your social security number?"},{"type":"gather","input":"dtmf speech","timeout":15},{"type":"say","text":"Thank you. We're now blocking the suspicious transaction. You should see the reversal within 24 hours."}]}`,
		ScriptType:      "text",
		MaxDurationSec:  180,
		ConsentMessage:  "This call is recorded for security purposes.",
	},
	{
		Name:            "CEO Urgent Request",
		Category:        "Executive Impersonation",
		DifficultyLevel: 3,
		Language:        "en",
		CallerIdName:    "CEO Office",
		Pretext:         "CEO's assistant calling about an urgent wire transfer that needs immediate authorization.",
		Greeting:        "Hi, this is calling from the CEO's office. I have an urgent matter regarding a time-sensitive payment that needs your authorization.",
		SuccessCriteria: "Target agrees to process the payment or provides authorization codes.",
		Script:          `{"steps":[{"type":"say","text":"Hi, this is calling from the CEO's office. The CEO is in a meeting and asked me to reach you directly. We have an urgent vendor payment of forty-seven thousand five hundred euros that needs to be processed today. The CEO has already approved it but we need your authorization code to release the funds. Can you provide that?"},{"type":"gather","input":"speech","timeout":20},{"type":"say","text":"Perfect, I'll process that right away. The CEO will follow up with you after the meeting. Thank you for your help."}]}`,
		ScriptType:      "text",
		MaxDurationSec:  240,
		ConsentMessage:  "This call may be monitored for quality purposes.",
	},
	{
		Name:            "Vendor Payment Update",
		Category:        "Vendor Impersonation",
		DifficultyLevel: 3,
		Language:        "en",
		CallerIdName:    "Accounts Dept",
		Pretext:         "Vendor's accounts department calling about updated bank details for upcoming payment.",
		Greeting:        "Hello, this is the accounts department from Apex Business Solutions. I'm calling regarding the upcoming payment for invoice number INV-2024-8847.",
		SuccessCriteria: "Target confirms they will update the payment details or provides internal payment system access.",
		Script:          `{"steps":[{"type":"say","text":"Hello, this is the accounts department from Apex Business Solutions. I'm calling regarding your upcoming payment. Our banking details have changed due to a recent corporate restructure. I need to provide you with our new account details so the payment can be routed correctly. Could you access your payment system to update our information?"},{"type":"gather","input":"speech","timeout":15},{"type":"say","text":"The new bank account number is... Actually, let me email you the details to be safe. What's the best email to send that to?"},{"type":"gather","input":"speech","timeout":10}]}`,
		ScriptType:      "text",
		MaxDurationSec:  180,
		ConsentMessage:  "This call may be monitored for quality purposes.",
	},
	{
		Name:            "Microsoft Support Scam",
		Category:        "Tech Support",
		DifficultyLevel: 1,
		Language:        "en",
		CallerIdName:    "Microsoft Support",
		Pretext:         "Microsoft technical support calling about detected malware on the user's computer.",
		Greeting:        "Hello, this is Microsoft Technical Support. We've detected unusual activity from your computer that indicates a potential malware infection.",
		SuccessCriteria: "Target agrees to install remote access software or provides login credentials.",
		Script:          `{"steps":[{"type":"say","text":"Hello, this is Microsoft Technical Support. Our security systems have detected unusual activity from your computer's IP address. It appears your system may be infected with malware. We can help you fix this right now. Can you go to your computer and press the Windows key plus R to open the Run dialog?"},{"type":"gather","input":"speech","timeout":15},{"type":"say","text":"Great. Now I need you to type in a web address so we can run a diagnostic scan. The address is support dash fix dot com."},{"type":"gather","input":"speech","timeout":10}]}`,
		ScriptType:      "text",
		MaxDurationSec:  300,
		ConsentMessage:  "This call may be recorded for quality assurance.",
	},
}

// GetVishingScenarioLibrary returns the built-in scenario library.
func GetVishingScenarioLibrary() []VishingScenario {
	return VishingScenarioLibrary
}
