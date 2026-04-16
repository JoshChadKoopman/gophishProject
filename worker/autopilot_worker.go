package worker

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gophish/gophish/ai"
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

// autopilotResources holds the shared resources needed to create campaigns.
type autopilotResources struct {
	templates     []models.Template
	smtp          models.SMTP
	page          models.Page
	campaignScope models.OrgScope
	groupName     string
	phishURL      string
	sendWindow    time.Duration
}

func processOrgAutopilot(ac models.AutopilotConfig) error {
	if reason := checkPreconditions(ac); reason != "" {
		log.Infof("Autopilot Worker: org %d skipped - %s", ac.OrgId, reason)
		return nil
	}

	eligibleTargets, groupName, err := collectEligibleTargets(ac)
	if err != nil {
		return err
	}
	if len(eligibleTargets) == 0 {
		log.Infof("Autopilot Worker: org %d - no eligible users this cycle", ac.OrgId)
		return nil
	}

	log.Infof("Autopilot Worker: org %d - %d eligible users for simulation", ac.OrgId, len(eligibleTargets))

	res, err := loadResources(ac, groupName)
	if err != nil {
		return err
	}

	batches := buildAdaptiveBatches(eligibleTargets, res.templates)
	return launchBatchedCampaigns(ac.OrgId, batches, res)
}

// checkPreconditions validates blackout dates, active hours, and feature gate.
// Returns the skip reason or "" if all checks pass.
func checkPreconditions(ac models.AutopilotConfig) string {
	loc, err := time.LoadLocation(ac.Timezone)
	if err != nil {
		loc = time.UTC
	}
	localNow := time.Now().In(loc)

	if models.IsBlackoutDate(ac.OrgId, localNow) {
		return "blackout date"
	}
	hour := localNow.Hour()
	if hour < ac.ActiveHoursStart || hour >= ac.ActiveHoursEnd {
		return fmt.Sprintf("outside active hours (%d not in %d-%d)", hour, ac.ActiveHoursStart, ac.ActiveHoursEnd)
	}
	if !models.OrgHasFeature(ac.OrgId, models.FeatureAutopilot) {
		return "autopilot feature not available"
	}
	if len(ac.GetGroupIds()) == 0 {
		return "no target groups configured"
	}
	return ""
}

// collectEligibleTargets returns users who are due for their next simulation
// based on cadence, along with the first group name for campaign labelling.
func collectEligibleTargets(ac models.AutopilotConfig) ([]models.Target, string, error) {
	lastSent, err := models.GetUsersLastSentDate(ac.OrgId)
	if err != nil {
		return nil, "", fmt.Errorf("error getting last sent dates: %w", err)
	}

	cadenceWindow := time.Duration(ac.CadenceDays) * 24 * time.Hour
	now := time.Now().UTC()
	var targets []models.Target
	var groupName string

	for _, gid := range ac.GetGroupIds() {
		scope := models.OrgScope{OrgId: ac.OrgId, IsSuperAdmin: true}
		group, err := models.GetGroup(gid, scope)
		if err != nil {
			log.Errorf("Autopilot Worker: org %d, group %d not found: %v", ac.OrgId, gid, err)
			continue
		}
		if groupName == "" {
			groupName = group.Name
		}
		for _, t := range group.Targets {
			if last, ok := lastSent[t.Email]; ok && now.Sub(last) < cadenceWindow {
				continue
			}
			targets = append(targets, t)
		}
	}
	return targets, groupName, nil
}

// loadResources fetches templates, SMTP profile, landing page, and campaign
// owner for the org.
func loadResources(ac models.AutopilotConfig, groupName string) (*autopilotResources, error) {
	scope := models.OrgScope{OrgId: ac.OrgId, IsSuperAdmin: true}
	templates, err := models.GetTemplates(scope)
	if err != nil || len(templates) == 0 {
		return nil, fmt.Errorf("no templates available for org %d", ac.OrgId)
	}

	smtp := models.SMTP{}
	if err := getByID("smtp", ac.SendingProfileId, &smtp); err != nil {
		return nil, fmt.Errorf("sending profile %d not found: %w", ac.SendingProfileId, err)
	}

	page := models.Page{}
	if err := getByID("pages", ac.LandingPageId, &page); err != nil {
		return nil, fmt.Errorf("landing page %d not found: %w", ac.LandingPageId, err)
	}

	campaignScope, err := findCampaignOwner(ac.OrgId)
	if err != nil {
		return nil, err
	}

	return &autopilotResources{
		templates:     templates,
		smtp:          smtp,
		page:          page,
		campaignScope: campaignScope,
		groupName:     groupName,
		phishURL:      ac.PhishURL,
		sendWindow:    time.Duration(ac.ActiveHoursEnd-ac.ActiveHoursStart) * time.Hour,
	}, nil
}

func findCampaignOwner(orgId int64) (models.OrgScope, error) {
	orgUsers, err := models.GetUsersByOrg(models.OrgScope{OrgId: orgId, IsSuperAdmin: true})
	if err != nil || len(orgUsers) == 0 {
		return models.OrgScope{}, fmt.Errorf("no users found in org %d", orgId)
	}
	owner := orgUsers[0]
	for _, u := range orgUsers {
		if hasModify, _ := u.HasPermission(models.PermissionModifyObjects); hasModify {
			owner = u
			break
		}
	}
	return models.OrgScope{
		OrgId:        orgId,
		UserId:       owner.Id,
		IsSuperAdmin: owner.Role.Slug == models.RoleSuperAdmin,
	}, nil
}

// campaignBatch groups targets that will receive the same template.
type campaignBatch struct {
	template models.Template
	targets  []models.Target
	profiles map[string]*models.UserTargetingProfile
}

// buildAdaptiveBatches uses each user's BRS/history to select the best template,
// then groups users by selected template for batched campaign creation.
func buildAdaptiveBatches(targets []models.Target, templates []models.Template) map[int64]*campaignBatch {
	batches := make(map[int64]*campaignBatch)

	for _, t := range targets {
		profile := getUserProfile(t)
		selected := models.SelectTemplate(profile, templates)
		if selected.Id == 0 {
			selected = templates[rand.Intn(len(templates))]
		}

		batch, ok := batches[selected.Id]
		if !ok {
			batch = &campaignBatch{
				template: selected,
				profiles: make(map[string]*models.UserTargetingProfile),
			}
			batches[selected.Id] = batch
		}
		batch.targets = append(batch.targets, t)
		batch.profiles[t.Email] = profile
	}
	return batches
}

// launchBatchedCampaigns creates one campaign per template batch and records
// per-user schedule entries with adaptive difficulty levels.
// When A/B testing is possible (even number of targets), it splits the batch
// into Variant A and Variant B to measure learning outcomes.
func launchBatchedCampaigns(orgId int64, batches map[int64]*campaignBatch, res *autopilotResources) error {
	now := time.Now().UTC()

	for _, batch := range batches {
		campaignName := fmt.Sprintf("Autopilot - %s - %s",
			batch.template.Name, now.Format("2006-01-02 15:04"))

		// Compute optimal launch time: if enough users have send-time data,
		// shift the launch to the most common preferred send window.
		launchTime := computeBatchLaunchTime(batch, now, res)

		campaign := models.Campaign{
			Name:       campaignName,
			Template:   models.Template{Name: batch.template.Name},
			Page:       models.Page{Name: res.page.Name},
			SMTP:       models.SMTP{Name: res.smtp.Name},
			URL:        res.phishURL,
			Groups:     []models.Group{{Name: res.groupName}},
			LaunchDate: launchTime,
			SendByDate: launchTime.Add(res.sendWindow),
		}

		if err := models.PostCampaign(&campaign, res.campaignScope); err != nil {
			log.Errorf("Autopilot Worker: org %d - failed to create campaign for template %q: %v",
				orgId, batch.template.Name, err)
			continue
		}

		log.Infof("Autopilot Worker: org %d - created adaptive campaign %d (%s) with %d recipients, template=%q difficulty=%d launch=%s",
			orgId, campaign.Id, campaignName, len(batch.targets),
			batch.template.Name, batch.template.DifficultyLevel,
			launchTime.Format("2006-01-02 15:04"))

		recordScheduleEntries(orgId, campaign.Id, launchTime, batch)

		// Record A/B test placeholders for post-hoc analysis
		recordABTestEntries(orgId, campaign.Id, batch)
	}
	return nil
}

func recordScheduleEntries(orgId, campaignId int64, scheduledDate time.Time, batch *campaignBatch) {
	for i, t := range batch.targets {
		difficulty := 2 // default medium
		if p := batch.profiles[t.Email]; p != nil {
			difficulty = p.RecommendedDifficulty
		}
		// Assign A/B variant: alternate A and B across the target list.
		variantId := "A"
		if i%2 == 1 {
			variantId = "B"
		}
		models.CreateAutopilotSchedule(&models.AutopilotSchedule{
			OrgId:           orgId,
			UserEmail:       t.Email,
			CampaignId:      campaignId,
			DifficultyLevel: difficulty,
			ScheduledDate:   scheduledDate,
			Sent:            true,
			VariantId:       variantId,
		})
	}
}

// recordABTestEntries pre-creates ABTestResult rows for each user in the batch
// so that the result handlers can populate outcome data later.
func recordABTestEntries(orgId, campaignId int64, batch *campaignBatch) {
	for i, t := range batch.targets {
		variantId := "A"
		if i%2 == 1 {
			variantId = "B"
		}
		user, err := models.GetUserByEmail(t.Email)
		if err != nil || user.Id == 0 {
			continue
		}
		models.RecordABTestResult(&models.ABTestResult{
			OrgId:        orgId,
			CampaignId:   campaignId,
			UserId:       user.Id,
			Email:        t.Email,
			VariantId:    variantId,
			TemplateId:   batch.template.Id,
			TemplateName: batch.template.Name,
		})
	}
}

// computeBatchLaunchTime determines the best time to launch a campaign batch
// based on the send-time profiles of the targets in the batch.
// It picks the most popular preferred send hour/day from users with sufficient data.
func computeBatchLaunchTime(batch *campaignBatch, fallback time.Time, res *autopilotResources) time.Time {
	hourVotes := make(map[int]int)
	dayVotes := make(map[int]int) // 0=Sun .. 6=Sat
	confident := 0

	for _, p := range batch.profiles {
		if p == nil || p.SendTimeConfidence < 0.3 {
			continue
		}
		confident++
		hourVotes[p.PreferredSendHour]++
		dayIdx := dayNameToIndex(p.PreferredSendDay)
		if dayIdx >= 0 {
			dayVotes[dayIdx]++
		}
	}

	// Need at least 30% of batch to have confident send-time data
	if confident < len(batch.targets)*30/100 || confident == 0 {
		return fallback
	}

	// Find peak hour and day
	bestHour, bestHourCount := 10, 0 // default 10am
	for h, c := range hourVotes {
		if c > bestHourCount {
			bestHour = h
			bestHourCount = c
		}
	}

	bestDay, bestDayCount := int(fallback.Weekday()), 0
	for d, c := range dayVotes {
		if c > bestDayCount {
			bestDay = d
			bestDayCount = c
		}
	}

	// Compute next occurrence of bestDay at bestHour
	now := time.Now().UTC()
	target := time.Date(now.Year(), now.Month(), now.Day(), bestHour, 0, 0, 0, time.UTC)
	daysUntil := (bestDay - int(now.Weekday()) + 7) % 7
	if daysUntil == 0 && target.Before(now) {
		daysUntil = 7
	}
	target = target.AddDate(0, 0, daysUntil)

	log.Infof("Autopilot Worker: send-time optimization → launch at %s (hour=%d, day=%s, confident=%d/%d)",
		target.Format("2006-01-02 15:04"), bestHour, time.Weekday(bestDay).String(), confident, len(batch.targets))

	return target
}

func dayNameToIndex(name string) int {
	switch name {
	case "Sunday":
		return 0
	case "Monday":
		return 1
	case "Tuesday":
		return 2
	case "Wednesday":
		return 3
	case "Thursday":
		return 4
	case "Friday":
		return 5
	case "Saturday":
		return 6
	default:
		return -1
	}
}

// getUserProfile builds a targeting profile for a target. If the target has a
// matching platform user with BRS data, the full adaptive profile is returned;
// otherwise nil is returned (which causes random/easy selection).
// This function now also populates send-time and department threat intelligence
// data for use by the send-time optimizer and AI prompt builder.
func getUserProfile(t models.Target) *models.UserTargetingProfile {
	user, err := models.GetUserByEmail(t.Email)
	if err != nil || user.Id == 0 {
		return nil
	}
	profile, err := models.GetUserTargetingProfile(user.Id)
	if err != nil {
		log.Errorf("Autopilot Worker: adaptive targeting failed for %s: %v", t.Email, err)
		return nil
	}
	// GetUserTargetingProfile already populates:
	// - PreferredSendDay/Hour/Confidence (from SendTimeProfile)
	// - Department/DepartmentThreats/DepartmentRiskMult (from DepartmentThreatProfile)
	return profile
}

// BuildUserContextForAI converts a targeting profile into the AI UserContext
// structure used by the prompt builder. This bridges the adaptive targeting engine
// with the AI template generator, ensuring department-specific threat intelligence
// flows into the generated phishing templates.
func BuildUserContextForAI(profile *models.UserTargetingProfile) *ai.UserContext {
	if profile == nil {
		return nil
	}
	ctx := &ai.UserContext{
		ClickRate:      profile.OverallClickRate,
		BRSScore:       profile.BRSComposite,
		TrendDirection: profile.TrendDirection,
	}

	// Weak categories
	for _, wc := range profile.WeakCategories {
		ctx.WeakCategories = append(ctx.WeakCategories, wc.Category)
	}

	// Recent categories → avoid
	ctx.AvoidCategories = profile.RecentCategories

	// Department-specific threat intelligence
	if profile.Department != "" {
		ctx.Department = profile.Department
		ctx.DepartmentThreats = profile.DepartmentThreats
		// Load contextual triggers from the full threat profile
		deptTP := models.GetDepartmentThreatProfile(profile.Department)
		ctx.ContextualTriggers = deptTP.ContextualTriggers
	}

	// Send-time optimization hints
	if profile.SendTimeConfidence >= 0.3 {
		ctx.OptimalSendDay = profile.PreferredSendDay
		ctx.OptimalSendHour = profile.PreferredSendHour
	}

	return ctx
}

// getByID is a helper to load a model by ID from a specific table.
func getByID(table string, id int64, dest interface{}) error {
	return models.GetDB().Table(table).Where("id = ?", id).First(dest).Error
}
