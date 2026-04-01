package worker

import (
	"fmt"
	"math/rand"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// AutopilotCheckInterval is how often the autopilot worker checks for due orgs.
const AutopilotCheckInterval = 1 * time.Hour

// StartAutopilotWorker launches a goroutine that periodically checks for
// autopilot-enabled orgs that are due for a new simulation cycle.
func StartAutopilotWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Autopilot Worker: recovered from panic: %v", r)
		}
	}()
	log.Info("Autopilot Worker Started - checking every hour")

	for range time.Tick(AutopilotCheckInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Autopilot Worker: recovered from panic in cycle: %v", r)
				}
			}()
			processAutopilotCycle()
		}()
	}
}

func processAutopilotCycle() {
	now := time.Now().UTC()
	configs, err := models.GetEnabledAutopilots(now)
	if err != nil {
		log.Errorf("Autopilot Worker: error fetching configs: %v", err)
		return
	}
	if len(configs) == 0 {
		return
	}

	log.Infof("Autopilot Worker: found %d org(s) due for processing", len(configs))

	for _, ac := range configs {
		if err := processOrgAutopilot(ac); err != nil {
			log.Errorf("Autopilot Worker: org %d failed: %v", ac.OrgId, err)
		}
		// Update last/next run regardless of success to avoid tight retry loops
		models.UpdateAutopilotRun(&ac)
	}
}

func processOrgAutopilot(ac models.AutopilotConfig) error {
	// Check blackout
	loc, err := time.LoadLocation(ac.Timezone)
	if err != nil {
		loc = time.UTC
	}
	localNow := time.Now().In(loc)

	if models.IsBlackoutDate(ac.OrgId, localNow) {
		log.Infof("Autopilot Worker: org %d skipped - blackout date", ac.OrgId)
		return nil
	}

	// Check active hours
	hour := localNow.Hour()
	if hour < ac.ActiveHoursStart || hour >= ac.ActiveHoursEnd {
		log.Infof("Autopilot Worker: org %d skipped - outside active hours (%d not in %d-%d)",
			ac.OrgId, hour, ac.ActiveHoursStart, ac.ActiveHoursEnd)
		return nil
	}

	// Check feature gate
	if !models.OrgHasFeature(ac.OrgId, models.FeatureAutopilot) {
		log.Infof("Autopilot Worker: org %d skipped - autopilot feature not available", ac.OrgId)
		return nil
	}

	// Get target groups and eligible users
	groupIds := ac.GetGroupIds()
	if len(groupIds) == 0 {
		log.Infof("Autopilot Worker: org %d skipped - no target groups configured", ac.OrgId)
		return nil
	}

	// Get last-sent dates for users
	lastSent, err := models.GetUsersLastSentDate(ac.OrgId)
	if err != nil {
		return fmt.Errorf("error getting last sent dates: %w", err)
	}

	// Collect eligible users from target groups
	cadenceWindow := time.Duration(ac.CadenceDays) * 24 * time.Hour
	now := time.Now().UTC()
	var eligibleTargets []models.Target
	var eligibleGroupName string

	for _, gid := range groupIds {
		scope := models.OrgScope{OrgId: ac.OrgId, IsSuperAdmin: true}
		group, err := models.GetGroup(gid, scope)
		if err != nil {
			log.Errorf("Autopilot Worker: org %d, group %d not found: %v", ac.OrgId, gid, err)
			continue
		}
		if eligibleGroupName == "" {
			eligibleGroupName = group.Name
		}
		for _, t := range group.Targets {
			if last, ok := lastSent[t.Email]; ok {
				if now.Sub(last) < cadenceWindow {
					continue // Not yet due
				}
			}
			eligibleTargets = append(eligibleTargets, t)
		}
	}

	if len(eligibleTargets) == 0 {
		log.Infof("Autopilot Worker: org %d - no eligible users this cycle", ac.OrgId)
		return nil
	}

	log.Infof("Autopilot Worker: org %d - %d eligible users for simulation", ac.OrgId, len(eligibleTargets))

	// Pick a random existing template from the org
	scope := models.OrgScope{OrgId: ac.OrgId, IsSuperAdmin: true}
	templates, err := models.GetTemplates(scope)
	if err != nil || len(templates) == 0 {
		log.Errorf("Autopilot Worker: org %d - no templates available", ac.OrgId)
		return fmt.Errorf("no templates available for org %d", ac.OrgId)
	}
	template := templates[rand.Intn(len(templates))]

	// Get the sending profile
	smtp := models.SMTP{}
	if err := getByID("smtp", ac.SendingProfileId, &smtp); err != nil {
		return fmt.Errorf("sending profile %d not found: %w", ac.SendingProfileId, err)
	}

	// Get the landing page
	page := models.Page{}
	if err := getByID("pages", ac.LandingPageId, &page); err != nil {
		return fmt.Errorf("landing page %d not found: %w", ac.LandingPageId, err)
	}

	// Create the campaign
	campaignName := fmt.Sprintf("Autopilot - %s", time.Now().Format("2006-01-02 15:04"))

	// Build a temporary group with eligible targets for campaign creation
	// We use the first configured group name for the campaign
	campaign := models.Campaign{
		Name:     campaignName,
		Template: models.Template{Name: template.Name},
		Page:     models.Page{Name: page.Name},
		SMTP:     models.SMTP{Name: smtp.Name},
		URL:      ac.PhishURL,
		Groups: []models.Group{
			{Name: eligibleGroupName},
		},
		LaunchDate: time.Now().UTC(),
		SendByDate: time.Now().UTC().Add(time.Duration(ac.ActiveHoursEnd-ac.ActiveHoursStart) * time.Hour),
	}

	// Find a user in this org to attribute the campaign to
	orgUsers, err := models.GetUsersByOrg(models.OrgScope{OrgId: ac.OrgId, IsSuperAdmin: true})
	if err != nil || len(orgUsers) == 0 {
		return fmt.Errorf("no users found in org %d", ac.OrgId)
	}
	// Use the first admin user
	var campaignOwner models.User
	for _, u := range orgUsers {
		hasModify, _ := u.HasPermission(models.PermissionModifyObjects)
		if hasModify {
			campaignOwner = u
			break
		}
	}
	if campaignOwner.Id == 0 {
		campaignOwner = orgUsers[0]
	}

	campaignScope := models.OrgScope{
		OrgId:        ac.OrgId,
		UserId:       campaignOwner.Id,
		IsSuperAdmin: campaignOwner.Role.Slug == models.RoleSuperAdmin,
	}

	err = models.PostCampaign(&campaign, campaignScope)
	if err != nil {
		return fmt.Errorf("error creating autopilot campaign: %w", err)
	}

	log.Infof("Autopilot Worker: org %d - created campaign %d (%s) with %d recipients",
		ac.OrgId, campaign.Id, campaignName, len(eligibleTargets))

	// Record schedule entries
	for _, t := range eligibleTargets {
		models.CreateAutopilotSchedule(&models.AutopilotSchedule{
			OrgId:           ac.OrgId,
			UserEmail:       t.Email,
			CampaignId:      campaign.Id,
			DifficultyLevel: 2, // Default medium
			ScheduledDate:   now,
			Sent:            true,
		})
	}

	return nil
}

// getByID is a helper to load a model by ID from a specific table.
func getByID(table string, id int64, dest interface{}) error {
	return models.GetDB().Table(table).Where("id = ?", id).First(dest).Error
}
