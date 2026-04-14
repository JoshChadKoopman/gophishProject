package models

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Vishing Campaign Scheduling ─────────────────────────────────
// Handles the logic for scheduling calls within active hours, managing
// retries, and ingesting results from the telephony provider.

// VishingCallTask represents a single scheduled call to be placed.
type VishingCallTask struct {
	CampaignId  int64     `json:"campaign_id"`
	ResultId    int64     `json:"result_id"`
	PhoneNumber string    `json:"phone_number"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	ScenarioId  int64     `json:"scenario_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Attempt     int       `json:"attempt"`
}

// LaunchVishingCampaign populates the results table with call tasks for
// each target in the campaign's groups and marks the campaign as in-progress.
func LaunchVishingCampaign(campaignId, orgId int64) error {
	campaign, err := GetVishingCampaign(campaignId, orgId)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	if campaign.Status != CampaignCreated {
		return fmt.Errorf("campaign must be in 'Created' status to launch (current: %s)", campaign.Status)
	}

	// Gather targets from the groups
	groupIds := campaign.GetTargetGroupIds()
	if len(groupIds) == 0 {
		return fmt.Errorf("no target groups specified")
	}

	targets, err := getVishingTargets(orgId, groupIds)
	if err != nil {
		return fmt.Errorf("error loading targets: %w", err)
	}
	if len(targets) == 0 {
		return fmt.Errorf("no targets with phone numbers found in the specified groups")
	}

	// Create a VishingResult for each target
	now := time.Now().UTC()
	for _, t := range targets {
		result := &VishingResult{
			CampaignId:   campaignId,
			OrgId:        orgId,
			Email:        t.Email,
			FirstName:    t.FirstName,
			LastName:     t.LastName,
			PhoneNumber:  t.Phone,
			Status:       VishingStatusPending,
			AttemptCount: 0,
			CreatedDate:  now,
			ModifiedDate: now,
		}
		if err := RecordVishingResult(result); err != nil {
			log.Errorf("vishing: failed to create result for %s: %v", t.Email, err)
		}
	}

	// Mark campaign as in-progress
	campaign.Status = CampaignInProgress
	campaign.LaunchDate = now
	campaign.ModifiedDate = now
	db.Save(&campaign)

	log.Infof("vishing: launched campaign %d with %d targets", campaignId, len(targets))
	return nil
}

// getVishingTargets loads targets from the specified groups, filtering for
// those that have phone numbers.
func getVishingTargets(orgId int64, groupIds []int64) ([]Target, error) {
	var targets []Target
	err := db.Table("targets").
		Joins("JOIN group_targets gt ON gt.target_id = targets.id").
		Joins("JOIN groups g ON g.id = gt.group_id").
		Where("g.id IN (?) AND g.org_id = ? AND targets.phone != ''", groupIds, orgId).
		Find(&targets).Error
	return targets, err
}

// GetPendingVishingCalls returns all calls that are pending and within
// the campaign's active calling hours.
func GetPendingVishingCalls(campaignId int64) ([]VishingCallTask, error) {
	campaign := VishingCampaign{}
	if err := db.Where("id = ?", campaignId).First(&campaign).Error; err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	currentHour := now.Hour()

	// Check if we're within active calling hours
	if currentHour < campaign.ActiveHoursStart || currentHour >= campaign.ActiveHoursEnd {
		return nil, nil // Outside calling hours
	}

	var results []VishingResult
	q := db.Where("campaign_id = ? AND status = ?", campaignId, VishingStatusPending)
	if campaign.RetryAttempts > 0 {
		// Also include no-answer results that haven't exceeded retry limit
		q = q.Or("campaign_id = ? AND status IN (?) AND attempt_count < ?",
			campaignId,
			[]string{VishingStatusNoAnswer, VishingStatusBusy, VishingStatusVoicemail},
			campaign.RetryAttempts+1,
		)
	}
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}

	tasks := make([]VishingCallTask, 0, len(results))
	for _, r := range results {
		tasks = append(tasks, VishingCallTask{
			CampaignId:  campaignId,
			ResultId:    r.Id,
			PhoneNumber: r.PhoneNumber,
			Email:       r.Email,
			FirstName:   r.FirstName,
			LastName:    r.LastName,
			ScenarioId:  campaign.ScenarioId,
			ScheduledAt: now,
			Attempt:     r.AttemptCount + 1,
		})
	}

	return tasks, nil
}

// ProcessVishingCallResult ingests a call result from the telephony provider
// webhook and updates the campaign status.
func ProcessVishingCallResult(callSid string, status string, durationSec int, callData map[string]interface{}) error {
	var result VishingResult
	if err := db.Where("call_sid = ?", callSid).First(&result).Error; err != nil {
		return fmt.Errorf("result not found for call SID %s: %w", callSid, err)
	}

	result.Status = status
	result.CallDuration = durationSec
	result.CompletedDate = time.Now().UTC()
	result.ModifiedDate = time.Now().UTC()

	if callData != nil {
		if info, ok := callData["info_disclosed"]; ok {
			b, _ := json.Marshal(info)
			result.InfoDisclosed = string(b)
		}
		if recording, ok := callData["recording_url"].(string); ok {
			result.RecordingURL = recording
		}
		if path, ok := callData["ivr_path"]; ok {
			b, _ := json.Marshal(path)
			result.IVRPath = string(b)
		}
	}

	if err := db.Save(&result).Error; err != nil {
		return err
	}

	// Apply BRS penalty/reward
	go func() {
		var user Target
		if err := db.Where("email = ?", result.Email).First(&user).Error; err == nil {
			ApplyVishingBRSPenalty(user.Id, result.Status)
		}
	}()

	// Check if campaign should be marked as complete
	go checkVishingCampaignCompletion(result.CampaignId)

	return nil
}

// checkVishingCampaignCompletion checks if all results in a campaign have
// a terminal status and marks the campaign as complete if so.
func checkVishingCampaignCompletion(campaignId int64) {
	var pending int64
	db.Model(&VishingResult{}).
		Where("campaign_id = ? AND status IN (?)", campaignId,
			[]string{VishingStatusPending, VishingStatusDialing}).
		Count(&pending)

	if pending == 0 {
		now := time.Now().UTC()
		db.Model(&VishingCampaign{}).
			Where("id = ?", campaignId).
			Updates(map[string]interface{}{
				"status":         CampaignComplete,
				"completed_date": now,
				"modified_date":  now,
			})
		log.Infof("vishing: campaign %d marked as completed", campaignId)
	}
}

// CompleteVishingCampaign manually marks a campaign as completed.
func CompleteVishingCampaign(campaignId, orgId int64) error {
	return db.Model(&VishingCampaign{}).
		Where("id = ? AND org_id = ?", campaignId, orgId).
		Updates(map[string]interface{}{
			"status":         CampaignComplete,
			"completed_date": time.Now().UTC(),
			"modified_date":  time.Now().UTC(),
		}).Error
}

// ── Vishing Org-Level Stats ─────────────────────────────────────

// VishingOrgStats provides organisation-wide vishing metrics.
type VishingOrgStats struct {
	TotalCampaigns  int64   `json:"total_campaigns"`
	TotalCalls      int64   `json:"total_calls"`
	AvgAnswerRate   float64 `json:"avg_answer_rate"`
	AvgEngageRate   float64 `json:"avg_engagement_rate"`
	AvgFailRate     float64 `json:"avg_fail_rate"`
	AvgReportRate   float64 `json:"avg_report_rate"`
	AvgCallDuration float64 `json:"avg_call_duration_sec"`
}

// GetVishingOrgStats returns aggregated vishing stats for an organisation.
func GetVishingOrgStats(orgId int64) VishingOrgStats {
	stats := VishingOrgStats{}

	db.Model(&VishingCampaign{}).Where(queryWhereOrgID, orgId).Count(&stats.TotalCampaigns)

	var results []VishingResult
	db.Where(queryWhereOrgID, orgId).Find(&results)

	stats.TotalCalls = int64(len(results))
	if stats.TotalCalls == 0 {
		return stats
	}

	var answered, engaged, credGiven, reported int64
	var totalDuration int64
	for _, r := range results {
		switch r.Status {
		case VishingStatusAnswered, VishingStatusEngaged, VishingStatusCredGiven, VishingStatusHungUp:
			answered++
			totalDuration += int64(r.CallDuration)
		}
		if r.Status == VishingStatusEngaged || r.Status == VishingStatusCredGiven {
			engaged++
		}
		if r.Status == VishingStatusCredGiven {
			credGiven++
		}
		if r.Reported {
			reported++
		}
	}

	total := float64(stats.TotalCalls)
	stats.AvgAnswerRate = float64(answered) / total * 100
	stats.AvgEngageRate = float64(engaged) / total * 100
	stats.AvgFailRate = float64(credGiven) / total * 100
	stats.AvgReportRate = float64(reported) / total * 100
	if answered > 0 {
		stats.AvgCallDuration = float64(totalDuration) / float64(answered)
	}

	return stats
}
