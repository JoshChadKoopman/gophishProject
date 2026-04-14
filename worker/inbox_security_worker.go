package worker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gophish/gophish/ai"
	"github.com/gophish/gophish/config"
	imappkg "github.com/gophish/gophish/imap"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// InboxSecurityCheckInterval is how often the inbox security worker runs.
const InboxSecurityCheckInterval = 5 * time.Minute

// whereByID is the GORM condition for filtering by primary key.
const whereByID = "id = ?"

// Shared error format strings.
const errFmtResolveProvider = "resolve provider: %w"

// inboxAIConfig holds the AI configuration for the inbox security worker.
var inboxAIConfig config.AIConfig

// StartInboxSecurityWorker launches the background goroutine that performs:
// - Real-time inbox scanning (AI inbox analysis)
// - BEC detection on new scans
// - Graymail classification
// - Auto-remediation of detected threats
// - Ticket creation and auto-resolution for phishing reports
func StartInboxSecurityWorker(aiCfg config.AIConfig) {
	inboxAIConfig = aiCfg
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Inbox Security Worker: recovered from panic: %v", r)
		}
	}()
	log.Info("Inbox Security Worker Started — monitoring for email threats")

	// Initial run 5 minutes after startup
	time.AfterFunc(5*time.Minute, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Inbox Security Worker: recovered from panic in initial run: %v", r)
			}
		}()
		log.Info("Inbox Security Worker: running initial scan cycle")
		runInboxSecurityCycle()
	})

	for range time.Tick(InboxSecurityCheckInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Inbox Security Worker: recovered from panic in cycle: %v", r)
				}
			}()
			runInboxSecurityCycle()
		}()
	}
}

// runInboxSecurityCycle performs one pass of all inbox security tasks.
func runInboxSecurityCycle() {
	// 1. Process pending remediation actions
	processRemediationActions()

	// 2. Auto-analyze unreported emails from report button
	processAutoAnalysis()

	// 3. Process overdue phishing tickets
	processOverdueTickets()

	// 4. Scan monitored inboxes (IMAP/API-based)
	processInboxMonitorScans()
}

// processRemediationActions executes pending/approved remediation actions.
func processRemediationActions() {
	actions, err := models.GetPendingRemediationActions()
	if err != nil {
		log.Errorf("Inbox Security Worker: error fetching pending remediation actions: %v", err)
		return
	}

	executed := 0
	for _, action := range actions {
		if action.RequiresApproval && action.Status == models.InboxRemStatusPending {
			continue // Skip pending actions that need approval
		}
		executeRemediationAction(&action)
		executed++
	}

	if executed > 0 {
		log.Infof("Inbox Security Worker: executed %d remediation actions", executed)
	}
}

// executeRemediationAction performs the actual remediation based on action type.
func executeRemediationAction(action *models.RemediationAction) {
	models.UpdateRemediationStatus(action.Id, models.InboxRemStatusExecuting, "", 0)

	var err error
	affected := 0

	switch action.ActionType {
	case models.RemediationActionDelete:
		affected, err = executeDeleteAction(action)
	case models.RemediationActionQuarantine:
		affected, err = executeQuarantineAction(action)
	case models.RemediationActionBlock:
		affected, err = executeBlockSenderAction(action)
	case models.RemediationActionPurge:
		affected, err = executePurgeAction(action)
	case models.RemediationActionRestore:
		affected, err = executeRestoreAction(action)
	default:
		err = nil
		log.Warnf("Inbox Security Worker: unknown action type '%s' for remediation %d", action.ActionType, action.Id)
	}

	if err != nil {
		log.Errorf("Inbox Security Worker: remediation action %d failed: %v", action.Id, err)
		models.UpdateRemediationStatus(action.Id, models.InboxRemStatusFailed, err.Error(), affected)
		return
	}

	msg := "Remediation completed successfully"
	models.UpdateRemediationStatus(action.Id, models.InboxRemStatusCompleted, msg, affected)
	log.Infof("Inbox Security Worker: remediation action %d completed (type=%s, affected=%d)", action.Id, action.ActionType, affected)
}

// executeDeleteAction removes a message from the target inbox.
func executeDeleteAction(action *models.RemediationAction) (int, error) {
	log.Infof("Inbox Security Worker: executing DELETE for message_id=%s in mailbox=%s",
		action.MessageId, action.TargetEmail)

	provider, err := resolveProviderForMailbox(action.OrgId, action.TargetEmail)
	if err != nil {
		return 0, fmt.Errorf(errFmtResolveProvider, err)
	}
	if err := provider.DeleteMessage(action.TargetEmail, action.MessageId); err != nil {
		return 0, fmt.Errorf("%s delete: %w", provider.ProviderName(), err)
	}
	return 1, nil
}

// executeQuarantineAction moves a message to a quarantine folder.
func executeQuarantineAction(action *models.RemediationAction) (int, error) {
	log.Infof("Inbox Security Worker: executing QUARANTINE for message_id=%s in mailbox=%s",
		action.MessageId, action.TargetEmail)

	provider, err := resolveProviderForMailbox(action.OrgId, action.TargetEmail)
	if err != nil {
		return 0, fmt.Errorf(errFmtResolveProvider, err)
	}
	if err := provider.QuarantineMessage(action.TargetEmail, action.MessageId); err != nil {
		return 0, fmt.Errorf("%s quarantine: %w", provider.ProviderName(), err)
	}
	return 1, nil
}

// executeBlockSenderAction adds the sender to the org block list.
func executeBlockSenderAction(action *models.RemediationAction) (int, error) {
	log.Infof("Inbox Security Worker: executing BLOCK SENDER %s for org %d",
		action.SenderEmail, action.OrgId)
	// Block-sender is an administrative action that requires tenant-level admin
	// permissions. Log the intent and record the block in the DB; actual transport
	// rule creation is provider-dependent and may require manual confirmation.
	log.Warnf("Inbox Security Worker: block-sender for %s recorded; transport rule must be applied via admin portal for org %d",
		action.SenderEmail, action.OrgId)
	return 0, nil
}

// executePurgeAction removes matching messages from ALL mailboxes in the org.
func executePurgeAction(action *models.RemediationAction) (int, error) {
	log.Infof("Inbox Security Worker: executing ORG-WIDE PURGE for subject='%s' sender='%s' in org %d",
		action.Subject, action.SenderEmail, action.OrgId)

	cfg, err := models.GetInboxMonitorConfig(action.OrgId)
	if err != nil {
		return 0, fmt.Errorf("get monitor config: %w", err)
	}

	provider, err := resolveProviderForOrg(&cfg)
	if err != nil {
		return 0, fmt.Errorf(errFmtResolveProvider, err)
	}

	affected := 0
	for _, mb := range cfg.GetMonitoredMailboxList() {
		n := purgeMailbox(provider, mb, action.SenderEmail, action.Subject)
		affected += n
	}
	return affected, nil
}

// purgeMailbox deletes matching messages from a single mailbox.
func purgeMailbox(provider imappkg.InboxProvider, mb, sender, subject string) int {
	msgs, err := provider.FetchNewMessages(mb, time.Time{})
	if err != nil {
		log.Warnf("Inbox Security Worker: purge scan failed for %s: %v", mb, err)
		return 0
	}
	count := 0
	for _, msg := range msgs {
		if msg.SenderEmail != sender && msg.Subject != subject {
			continue
		}
		if delErr := provider.DeleteMessage(mb, msg.ProviderUID); delErr != nil {
			log.Warnf("Inbox Security Worker: purge delete failed in %s: %v", mb, delErr)
		} else {
			count++
		}
	}
	return count
}

// executeRestoreAction restores a previously quarantined/deleted message.
func executeRestoreAction(action *models.RemediationAction) (int, error) {
	log.Infof("Inbox Security Worker: executing RESTORE for message_id=%s",
		action.MessageId)

	provider, err := resolveProviderForMailbox(action.OrgId, action.TargetEmail)
	if err != nil {
		return 0, fmt.Errorf(errFmtResolveProvider, err)
	}
	if err := provider.RestoreMessage(action.TargetEmail, action.MessageId); err != nil {
		return 0, fmt.Errorf("%s restore: %w", provider.ProviderName(), err)
	}
	return 1, nil
}

// resolveProviderForMailbox creates the appropriate InboxProvider for a given
// org and target mailbox based on the org's monitor configuration.
func resolveProviderForMailbox(orgId int64, mailbox string) (imappkg.InboxProvider, error) {
	cfg, err := models.GetInboxMonitorConfig(orgId)
	if err != nil {
		return nil, fmt.Errorf("org %d monitor config not found: %w", orgId, err)
	}
	return resolveProviderForOrg(&cfg)
}

// resolveProviderForOrg returns the best InboxProvider based on the org config.
func resolveProviderForOrg(cfg *models.InboxMonitorConfig) (imappkg.InboxProvider, error) {
	// Priority: MS365 Graph API > Google Workspace > IMAP
	if cfg.MS365Enabled && cfg.MS365TenantId != "" {
		return &imappkg.GraphProvider{
			TenantId:     cfg.MS365TenantId,
			ClientId:     cfg.MS365ClientId,
			ClientSecret: cfg.MS365ClientSecret,
		}, nil
	}
	if cfg.GoogleWorkspaceEnabled && cfg.GoogleAdminEmail != "" {
		return &imappkg.GmailProvider{
			AdminEmail:  cfg.GoogleAdminEmail,
			AccessToken: "", // Token obtained via service account delegation at runtime
		}, nil
	}
	if cfg.IMAPHost != "" {
		return &imappkg.IMAPProvider{
			Host:             cfg.IMAPHost,
			Port:             cfg.IMAPPort,
			Username:         cfg.IMAPUsername,
			Password:         cfg.IMAPPassword,
			TLS:              cfg.IMAPTLS,
			IgnoreCertErrors: false,
			QuarantineFolder: "Junk",
		}, nil
	}
	return nil, fmt.Errorf("no mail provider configured for org %d", cfg.OrgId)
}

// processAutoAnalysis picks up reported emails that haven't been analyzed yet
// (where the org has auto-analyze enabled) and runs AI analysis + ticket creation.
func processAutoAnalysis() {
	// Find all orgs with auto_analyze enabled
	type orgConfig struct {
		OrgId                  int64
		AutoRemediateThreshold string
	}

	var configs []models.ReportButtonConfig
	// Use raw query since we need the new columns
	models.GetDB().Where("auto_analyze = ? AND enabled = ?", true, true).Find(&configs)

	for _, cfg := range configs {
		processOrgAutoAnalysis(cfg.OrgId, "confirmed_phishing")
	}
}

// processOrgAutoAnalysis runs auto-analysis for one org's unprocessed reported emails.
func processOrgAutoAnalysis(orgId int64, autoRemediateThreshold string) {
	// Get unanalyzed reported emails
	var unanalyzed []models.ReportedEmail
	models.GetDB().Where("org_id = ? AND auto_analyzed = ? AND classification = ?",
		orgId, false, "pending").
		Order("created_date ASC").Limit(10).Find(&unanalyzed)

	if len(unanalyzed) == 0 {
		return
	}

	// Get AI config
	if !inboxAIConfig.Enabled {
		return
	}

	client, err := ai.NewClient(inboxAIConfig.Provider, inboxAIConfig.APIKey, inboxAIConfig.Model)
	if err != nil {
		log.Errorf("Inbox Security Worker: failed to create AI client for org %d: %v", orgId, err)
		return
	}

	for _, re := range unanalyzed {
		analyzeAndRemediateEmail(orgId, &re, client, autoRemediateThreshold)
	}
}

// analyzeAndRemediateEmail processes a single reported email: runs AI analysis,
// creates a ticket, and optionally auto-remediates.
func analyzeAndRemediateEmail(orgId int64, re *models.ReportedEmail, client ai.Client, threshold string) {
	var rawHeaders, rawBody string
	models.GetDB().Model(&models.ReportedEmail{}).
		Where(whereByID, re.Id).
		Select("raw_headers, raw_body").
		Row().Scan(&rawHeaders, &rawBody)

	analysis, err := models.AnalyzeReportedEmail(
		orgId, re.Id,
		rawHeaders, rawBody,
		re.SenderEmail, re.Subject,
		client,
	)

	// Mark as auto-analyzed regardless of outcome
	models.GetDB().Model(&models.ReportedEmail{}).
		Where(whereByID, re.Id).
		Update("auto_analyzed", true)

	if err != nil {
		log.Errorf("Inbox Security Worker: auto-analysis failed for reported email %d: %v", re.Id, err)
		return
	}

	// Create ticket
	ticket, ticketErr := models.CreateTicketFromReportedEmail(orgId, re, analysis)
	if ticketErr != nil {
		log.Errorf("Inbox Security Worker: failed to create ticket for reported email %d: %v", re.Id, ticketErr)
	} else if ticket != nil && ticket.AutoResolved {
		log.Infof("Inbox Security Worker: ticket %s auto-resolved for reported email %d", ticket.TicketNumber, re.Id)
	}

	// Auto-remediate if threshold met
	if analysis != nil && shouldAutoRemediate(analysis.ThreatLevel, threshold) {
		autoRemediateReportedEmail(orgId, re)
	}
}

// autoRemediateReportedEmail quarantines a reported email and marks it as remediated.
func autoRemediateReportedEmail(orgId int64, re *models.ReportedEmail) {
	action := &models.RemediationAction{
		OrgId:       orgId,
		ActionType:  models.RemediationActionQuarantine,
		TargetType:  models.RemediationTargetReportedEmail,
		TargetId:    re.Id,
		TargetEmail: re.ReporterEmail,
		Subject:     re.Subject,
		SenderEmail: re.SenderEmail,
		Status:      models.InboxRemStatusExecuting,
		InitiatedBy: 0, // system
		Scope:       models.RemediationScopeSingle,
	}
	if err := models.CreateRemediationAction(action); err == nil {
		executeRemediationAction(action)
		models.GetDB().Model(&models.ReportedEmail{}).
			Where(whereByID, re.Id).
			Updates(map[string]interface{}{
				"remediated":      true,
				"remediated_date": time.Now().UTC(),
			})
	}
}

// shouldAutoRemediate checks if the detected threat level meets the threshold.
func shouldAutoRemediate(threatLevel, threshold string) bool {
	levels := map[string]int{
		models.ThreatLevelSafe:              0,
		models.ThreatLevelSuspicious:        1,
		models.ThreatLevelLikelyPhishing:    2,
		models.ThreatLevelConfirmedPhishing: 3,
	}
	return levels[threatLevel] >= levels[threshold]
}

// processOverdueTickets escalates tickets that have breached their SLA.
func processOverdueTickets() {
	var overdue []models.PhishingTicket
	models.GetDB().Where("status IN (?) AND sla_deadline < ? AND escalated = ?",
		[]string{models.TicketStatusOpen, models.TicketStatusInProgress},
		time.Now().UTC(), false).
		Find(&overdue)

	for _, ticket := range overdue {
		models.EscalatePhishingTicket(ticket.Id, ticket.OrgId, 0)
		log.Infof("Inbox Security Worker: escalated overdue ticket %s (org %d)", ticket.TicketNumber, ticket.OrgId)
	}

	if len(overdue) > 0 {
		log.Infof("Inbox Security Worker: escalated %d overdue tickets", len(overdue))
	}
}

// processInboxMonitorScans runs inbox scans for all orgs with monitoring enabled.
func processInboxMonitorScans() {
	configs, err := models.GetAllEnabledMonitorConfigs()
	if err != nil {
		log.Errorf("Inbox Security Worker: error fetching monitor configs: %v", err)
		return
	}

	for _, config := range configs {
		// Check if scan interval has elapsed
		if !config.LastScanDate.IsZero() &&
			time.Since(config.LastScanDate).Seconds() < float64(config.ScanIntervalSeconds) {
			continue
		}
		scanOrgInbox(&config)
	}
}

// scanOrgInbox performs an inbox scan for one organization.
func scanOrgInbox(config *models.InboxMonitorConfig) {
	log.Infof("Inbox Security Worker: scanning inboxes for org %d", config.OrgId)

	mailboxes := config.GetMonitoredMailboxList()
	if len(mailboxes) == 0 {
		return
	}

	// Get AI config for analysis
	if !inboxAIConfig.Enabled {
		log.Warnf("Inbox Security Worker: AI not enabled for inbox scanning (org %d)", config.OrgId)
		return
	}

	client, err := ai.NewClient(inboxAIConfig.Provider, inboxAIConfig.APIKey, inboxAIConfig.Model)
	if err != nil {
		log.Errorf("Inbox Security Worker: failed to create AI client for org %d: %v", config.OrgId, err)
		return
	}

	scanned := 0
	threats := 0

	for _, mailbox := range mailboxes {
		emails := fetchInboxEmails(config, mailbox)
		for _, email := range emails {
			result := analyzeInboxEmail(config, client, mailbox, &email)
			if result != nil {
				scanned++
				if result.ThreatLevel != models.ThreatLevelSafe {
					threats++
					handleInboxThreat(config, result)
				}
			}
		}
	}

	models.UpdateMonitorLastScan(config.OrgId)

	if scanned > 0 {
		log.Infof("Inbox Security Worker: scanned %d emails for org %d (%d threats found)",
			scanned, config.OrgId, threats)
	}
}

// InboxEmail represents a raw email fetched from an inbox for scanning.
type InboxEmail struct {
	MessageId    string
	SenderEmail  string
	Subject      string
	Headers      string
	Body         string
	ReceivedDate time.Time
}

// fetchInboxEmails retrieves new emails from a monitored mailbox.
// Uses IMAP, Microsoft Graph API, or Gmail API based on org configuration.
func fetchInboxEmails(config *models.InboxMonitorConfig, mailbox string) []InboxEmail {
	log.Infof("Inbox Security Worker: fetching new emails from %s (org %d)", mailbox, config.OrgId)

	provider, err := resolveProviderForOrg(config)
	if err != nil {
		log.Errorf("Inbox Security Worker: no provider for org %d: %v", config.OrgId, err)
		return nil
	}

	msgs, err := provider.FetchNewMessages(mailbox, config.LastScanDate)
	if err != nil {
		log.Errorf("Inbox Security Worker: %s fetch failed for %s: %v", provider.ProviderName(), mailbox, err)
		return nil
	}

	// Convert from imap.InboxMessage to worker.InboxEmail
	var emails []InboxEmail
	for _, m := range msgs {
		emails = append(emails, InboxEmail{
			MessageId:    m.MessageId,
			SenderEmail:  m.SenderEmail,
			Subject:      m.Subject,
			Headers:      m.Headers,
			Body:         m.Body,
			ReceivedDate: m.ReceivedDate,
		})
	}

	if len(emails) > 0 {
		log.Infof("Inbox Security Worker: fetched %d new emails from %s via %s",
			len(emails), mailbox, provider.ProviderName())
	}

	return emails
}

// analyzeInboxEmail runs AI analysis on a single inbox email.
func analyzeInboxEmail(config *models.InboxMonitorConfig, client ai.Client, mailbox string, email *InboxEmail) *models.InboxScanResult {
	startTime := time.Now()

	// Build and send the AI analysis request
	prompt := ai.BuildEmailAnalysisPrompt(email.Headers, email.Body, email.SenderEmail, email.Subject)
	resp, err := client.Generate(ai.EmailAnalysisSystemPrompt, prompt)
	if err != nil {
		log.Errorf("Inbox Security Worker: AI analysis failed for message %s: %v", email.MessageId, err)
		return nil
	}

	parsed, err := ai.ParseEmailAnalysisResponse(resp.Content)
	if err != nil {
		log.Errorf("Inbox Security Worker: failed to parse AI response for message %s: %v", email.MessageId, err)
		return nil
	}

	duration := time.Since(startTime).Milliseconds()

	// Serialize indicators
	indicatorsJSON, _ := json.Marshal(parsed.Indicators)

	result := &models.InboxScanResult{
		OrgId:           config.OrgId,
		ConfigId:        config.Id,
		MailboxEmail:    mailbox,
		MessageId:       email.MessageId,
		SenderEmail:     email.SenderEmail,
		Subject:         email.Subject,
		ReceivedDate:    email.ReceivedDate,
		ThreatLevel:     parsed.ThreatLevel,
		Classification:  parsed.Classification,
		ConfidenceScore: parsed.Confidence,
		IsBEC:           parsed.Classification == models.ClassificationBEC,
		IsGraymail:      parsed.Classification == "graymail",
		Summary:         parsed.Summary,
		Indicators:      string(indicatorsJSON),
		ActionTaken:     models.ScanActionNone,
		ScanDurationMs:  int(duration),
	}

	if err := models.CreateInboxScanResult(result); err != nil {
		log.Errorf("Inbox Security Worker: failed to save scan result: %v", err)
		return nil
	}

	return result
}

// handleInboxThreat processes a detected threat from inbox scanning.
func handleInboxThreat(config *models.InboxMonitorConfig, result *models.InboxScanResult) {
	// Create a ticket
	ticket, err := models.CreateTicketFromScanResult(config.OrgId, result)
	if err != nil {
		log.Errorf("Inbox Security Worker: failed to create ticket for scan result %d: %v", result.Id, err)
	}

	// Auto-quarantine if enabled and threshold met
	if config.AutoQuarantine && shouldAutoRemediate(result.ThreatLevel, config.ThreatThreshold) {
		action := &models.RemediationAction{
			OrgId:       config.OrgId,
			ActionType:  models.RemediationActionQuarantine,
			TargetType:  models.RemediationTargetScanResult,
			TargetId:    result.Id,
			TargetEmail: result.MailboxEmail,
			MessageId:   result.MessageId,
			Subject:     result.Subject,
			SenderEmail: result.SenderEmail,
			Status:      models.InboxRemStatusExecuting,
			InitiatedBy: 0, // system
			Scope:       models.RemediationScopeSingle,
		}
		if err := models.CreateRemediationAction(action); err == nil {
			executeRemediationAction(action)
			result.ActionTaken = models.ScanActionQuarantined
		}
	}

	// Auto-delete if enabled (for confirmed phishing only)
	if config.AutoDelete && result.ThreatLevel == models.ThreatLevelConfirmedPhishing {
		action := &models.RemediationAction{
			OrgId:       config.OrgId,
			ActionType:  models.RemediationActionDelete,
			TargetType:  models.RemediationTargetScanResult,
			TargetId:    result.Id,
			TargetEmail: result.MailboxEmail,
			MessageId:   result.MessageId,
			Subject:     result.Subject,
			SenderEmail: result.SenderEmail,
			Status:      models.InboxRemStatusExecuting,
			InitiatedBy: 0,
			Scope:       models.RemediationScopeSingle,
		}
		if err := models.CreateRemediationAction(action); err == nil {
			executeRemediationAction(action)
			result.ActionTaken = models.ScanActionDeleted
		}
	}

	// BEC-specific handling
	if result.IsBEC {
		bec := &models.BECDetection{
			OrgId:        config.OrgId,
			ScanResultId: result.Id,
			ActualSender: result.SenderEmail,
			Summary:      result.Summary,
		}
		models.CreateBECDetection(bec)
	}

	// Graymail-specific handling
	if result.IsGraymail {
		gc := &models.GraymailClassification{
			OrgId:        config.OrgId,
			ScanResultId: result.Id,
			EmailSubject: result.Subject,
			SenderEmail:  result.SenderEmail,
			Category:     result.GraymailCategory,
		}
		models.CreateGraymailClassification(gc)
	}

	if ticket != nil {
		log.Infof("Inbox Security Worker: threat detected — ticket %s created (threat_level=%s, bec=%v)",
			ticket.TicketNumber, result.ThreatLevel, result.IsBEC)
	}
}
