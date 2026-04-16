package models

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Campaign Results — Deeper Analytics ─────────────────────────
// Adds: funnel visualization, time-to-click distribution, repeat
// offender tracking, and geo/device breakdown from User-Agent data.

// ── 1. Funnel Visualization ─────────────────────────────────────

// FunnelStage represents one stage in the phishing funnel.
type FunnelStage struct {
	Stage      string  `json:"stage"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"` // % of total recipients
	DropOff    float64 `json:"drop_off"`   // % who dropped off vs prior stage
}

// CampaignFunnel contains the complete funnel for a campaign.
type CampaignFunnel struct {
	CampaignId int64         `json:"campaign_id"`
	Total      int64         `json:"total"`
	Stages     []FunnelStage `json:"stages"`
}

// GetCampaignFunnel computes the full funnel for a specific campaign.
func GetCampaignFunnel(campaignId int64) (*CampaignFunnel, error) {
	var results []Result
	err := db.Where(campaignQueryWhereCampaignID, campaignId).Find(&results).Error
	if err != nil {
		return nil, err
	}

	total := int64(len(results))
	if total == 0 {
		return &CampaignFunnel{CampaignId: campaignId, Total: 0}, nil
	}

	// Count events — backfill higher statuses
	var sent, opened, clicked, submitted, reported int64
	for _, r := range results {
		switch r.Status {
		case EventDataSubmit:
			submitted++
			clicked++
			opened++
			sent++
		case EventClicked:
			clicked++
			opened++
			sent++
		case EventOpened:
			opened++
			sent++
		case EventSent:
			sent++
		}
		if r.Reported {
			reported++
		}
	}

	stages := []FunnelStage{
		{Stage: "Email Sent", Count: sent},
		{Stage: "Email Opened", Count: opened},
		{Stage: "Clicked Link", Count: clicked},
		{Stage: "Submitted Data", Count: submitted},
		{Stage: "Reported", Count: reported},
	}

	// Calculate percentages and drop-off
	for i := range stages {
		stages[i].Percentage = math.Round(float64(stages[i].Count)/float64(total)*1000) / 10
		if i == 0 {
			stages[i].DropOff = 0
		} else {
			prev := stages[i-1].Count
			if prev > 0 {
				stages[i].DropOff = math.Round((1-float64(stages[i].Count)/float64(prev))*1000) / 10
			} else {
				stages[i].DropOff = 100
			}
		}
	}

	return &CampaignFunnel{
		CampaignId: campaignId,
		Total:      total,
		Stages:     stages,
	}, nil
}

// ── 2. Time-to-Click Distribution ───────────────────────────────

// TimeToClickBucket represents a histogram bucket.
type TimeToClickBucket struct {
	Label   string  `json:"label"`   // e.g. "0-1 min", "1-2 min"
	MinSec  int     `json:"min_sec"` // lower bound in seconds
	MaxSec  int     `json:"max_sec"` // upper bound in seconds
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// TimeToClickDistribution contains the histogram and summary stats.
type TimeToClickDistribution struct {
	CampaignId      int64               `json:"campaign_id"`
	TotalClickers   int                 `json:"total_clickers"`
	MedianSeconds   float64             `json:"median_seconds"`
	MeanSeconds     float64             `json:"mean_seconds"`
	ImpulsiveCount  int                 `json:"impulsive_count"` // clicked < 2 min
	ImpulsivePct    float64             `json:"impulsive_pct"`
	ConsideredCount int                 `json:"considered_count"` // clicked >= 2 min
	ConsideredPct   float64             `json:"considered_pct"`
	Buckets         []TimeToClickBucket `json:"buckets"`
}

// GetTimeToClickDistribution computes when users clicked relative to email delivery.
func GetTimeToClickDistribution(campaignId int64) (*TimeToClickDistribution, error) {
	// Get all results with their send dates
	var results []Result
	err := db.Where(campaignQueryWhereCampaignID, campaignId).Find(&results).Error
	if err != nil {
		return nil, err
	}

	// Get click events from timeline
	var events []Event
	err = db.Where("campaign_id = ? AND message = ?", campaignId, EventClicked).
		Order("time ASC").Find(&events).Error
	if err != nil {
		return nil, err
	}

	// Build a map of email → send_date
	sendDates := map[string]time.Time{}
	for _, r := range results {
		if !r.SendDate.IsZero() {
			sendDates[r.Email] = r.SendDate
		}
	}

	// Calculate time-to-click for each clicker
	var deltas []float64
	for _, e := range events {
		sendDate, ok := sendDates[e.Email]
		if !ok || sendDate.IsZero() {
			continue
		}
		delta := e.Time.Sub(sendDate).Seconds()
		if delta < 0 {
			delta = 0
		}
		deltas = append(deltas, delta)
	}

	dist := &TimeToClickDistribution{
		CampaignId:    campaignId,
		TotalClickers: len(deltas),
	}

	if len(deltas) == 0 {
		dist.Buckets = defaultTimeBuckets()
		return dist, nil
	}

	sort.Float64s(deltas)

	// Summary stats
	dist.MedianSeconds = median64(deltas)
	var sum float64
	for _, d := range deltas {
		sum += d
	}
	dist.MeanSeconds = sum / float64(len(deltas))

	// Impulsive vs considered (threshold: 120 seconds = 2 min)
	for _, d := range deltas {
		if d < 120 {
			dist.ImpulsiveCount++
		} else {
			dist.ConsideredCount++
		}
	}
	dist.ImpulsivePct = math.Round(float64(dist.ImpulsiveCount)/float64(len(deltas))*1000) / 10
	dist.ConsideredPct = math.Round(float64(dist.ConsideredCount)/float64(len(deltas))*1000) / 10

	// Build histogram buckets
	bucketDefs := []struct {
		label  string
		minSec int
		maxSec int
	}{
		{"< 30s", 0, 30},
		{"30s–1m", 30, 60},
		{"1–2m", 60, 120},
		{"2–5m", 120, 300},
		{"5–10m", 300, 600},
		{"10–30m", 600, 1800},
		{"30m–1h", 1800, 3600},
		{"1–4h", 3600, 14400},
		{"4–24h", 14400, 86400},
		{"> 24h", 86400, 999999999},
	}

	buckets := make([]TimeToClickBucket, len(bucketDefs))
	for i, bd := range bucketDefs {
		buckets[i] = TimeToClickBucket{
			Label:  bd.label,
			MinSec: bd.minSec,
			MaxSec: bd.maxSec,
		}
		for _, d := range deltas {
			if d >= float64(bd.minSec) && d < float64(bd.maxSec) {
				buckets[i].Count++
			}
		}
		buckets[i].Percent = math.Round(float64(buckets[i].Count)/float64(len(deltas))*1000) / 10
	}
	dist.Buckets = buckets

	return dist, nil
}

func defaultTimeBuckets() []TimeToClickBucket {
	return []TimeToClickBucket{
		{Label: "< 30s", MinSec: 0, MaxSec: 30},
		{Label: "30s–1m", MinSec: 30, MaxSec: 60},
		{Label: "1–2m", MinSec: 60, MaxSec: 120},
		{Label: "2–5m", MinSec: 120, MaxSec: 300},
		{Label: "5–10m", MinSec: 300, MaxSec: 600},
		{Label: "10–30m", MinSec: 600, MaxSec: 1800},
		{Label: "30m–1h", MinSec: 1800, MaxSec: 3600},
		{Label: "1–4h", MinSec: 3600, MaxSec: 14400},
		{Label: "4–24h", MinSec: 14400, MaxSec: 86400},
		{Label: "> 24h", MinSec: 86400, MaxSec: 999999999},
	}
}

func median64(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// ── 3. Repeat Offender Tracking (Campaign-Level) ────────────────
// Note: This is separate from the escalation.RepeatOffender which is used
// for escalation workflows. This struct provides campaign-specific context.

// CampaignRepeatOffender identifies a user who clicked in multiple campaigns.
type CampaignRepeatOffender struct {
	Email             string   `json:"email"`
	FirstName         string   `json:"first_name"`
	LastName          string   `json:"last_name"`
	Position          string   `json:"position"`
	CampaignCount     int      `json:"campaign_count"` // number of campaigns they clicked in
	CampaignNames     []string `json:"campaign_names"`
	TotalClicks       int      `json:"total_clicks"`
	TotalSubmits      int      `json:"total_submits"`
	LastClickDate     string   `json:"last_click_date"`
	RiskLevel         string   `json:"risk_level"`          // "moderate" (2), "high" (3+), "critical" (5+)
	InCurrentCampaign bool     `json:"in_current_campaign"` // clicked in THIS campaign too
}

// GetCampaignRepeatOffenders returns users who clicked in 2+ campaigns within the org.
// If campaignId > 0, marks those who are also in the specified campaign.
func GetCampaignRepeatOffenders(orgId int64, campaignId int64) ([]CampaignRepeatOffender, error) {
	type offenderRow struct {
		Email         string
		FirstName     string
		LastName      string
		Position      string
		CampaignCount int
		TotalClicks   int
		TotalSubmits  int
		LastClick     time.Time
	}

	var rows []offenderRow
	err := db.Raw(`
		SELECT r.email, r.first_name, r.last_name, r.position,
			COUNT(DISTINCT r.campaign_id) as campaign_count,
			SUM(CASE WHEN r.status IN (?, ?) THEN 1 ELSE 0 END) as total_clicks,
			SUM(CASE WHEN r.status = ? THEN 1 ELSE 0 END) as total_submits,
			MAX(r.modified_date) as last_click
		FROM results r
		JOIN campaigns c ON r.campaign_id = c.id
		WHERE c.org_id = ?
			AND r.status IN (?, ?)
		GROUP BY r.email
		HAVING COUNT(DISTINCT r.campaign_id) >= 2
		ORDER BY campaign_count DESC, total_clicks DESC
	`, EventClicked, EventDataSubmit, EventDataSubmit,
		orgId, EventClicked, EventDataSubmit).Scan(&rows).Error

	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Get campaign names for each offender
	offenders := make([]CampaignRepeatOffender, 0, len(rows))
	for _, row := range rows {
		offender := CampaignRepeatOffender{
			Email:         row.Email,
			FirstName:     row.FirstName,
			LastName:      row.LastName,
			Position:      row.Position,
			CampaignCount: row.CampaignCount,
			TotalClicks:   row.TotalClicks,
			TotalSubmits:  row.TotalSubmits,
		}
		if !row.LastClick.IsZero() {
			offender.LastClickDate = row.LastClick.Format("2006-01-02 15:04")
		}

		// Risk level
		switch {
		case row.CampaignCount >= 5:
			offender.RiskLevel = "critical"
		case row.CampaignCount >= 3:
			offender.RiskLevel = "high"
		default:
			offender.RiskLevel = "moderate"
		}

		// Get campaign names this user clicked in
		type campName struct {
			Name string
		}
		var names []campName
		db.Raw(`
			SELECT DISTINCT c.name
			FROM results r
			JOIN campaigns c ON r.campaign_id = c.id
			WHERE r.email = ? AND c.org_id = ?
				AND r.status IN (?, ?)
			ORDER BY c.launch_date DESC
			LIMIT 10
		`, row.Email, orgId, EventClicked, EventDataSubmit).Scan(&names)
		for _, n := range names {
			offender.CampaignNames = append(offender.CampaignNames, n.Name)
		}

		// Check if in the current campaign
		if campaignId > 0 {
			var count int64
			db.Model(&Result{}).Where("campaign_id = ? AND email = ? AND status IN (?, ?)",
				campaignId, row.Email, EventClicked, EventDataSubmit).Count(&count)
			offender.InCurrentCampaign = count > 0
		}

		offenders = append(offenders, offender)
	}

	return offenders, nil
}

// ── 4. Geo/Device Breakdown ─────────────────────────────────────

// DeviceBreakdownEntry represents one browser/OS/device-type combination.
type DeviceBreakdownEntry struct {
	Category string  `json:"category"` // "browser", "os", "device_type"
	Value    string  `json:"value"`
	Count    int     `json:"count"`
	Percent  float64 `json:"percent"`
}

// DeviceBreakdown contains the full device/browser/OS analysis.
type DeviceBreakdown struct {
	CampaignId  int64                  `json:"campaign_id"`
	TotalEvents int                    `json:"total_events"`
	Browsers    []DeviceBreakdownEntry `json:"browsers"`
	OSes        []DeviceBreakdownEntry `json:"oses"`
	DeviceTypes []DeviceBreakdownEntry `json:"device_types"`
}

// uaComponents holds parsed User-Agent components.
type uaComponents struct {
	Browser    string
	OS         string
	DeviceType string
}

// GetDeviceBreakdown parses User-Agent strings from click/submit events.
func GetDeviceBreakdown(campaignId int64) (*DeviceBreakdown, error) {
	var events []Event
	err := db.Where("campaign_id = ? AND message IN (?, ?)",
		campaignId, EventClicked, EventDataSubmit).Find(&events).Error
	if err != nil {
		return nil, err
	}

	bd := &DeviceBreakdown{CampaignId: campaignId}

	browsers := map[string]int{}
	oses := map[string]int{}
	deviceTypes := map[string]int{}

	for _, e := range events {
		if e.Details == "" {
			continue
		}
		ua := extractUserAgent(e.Details)
		if ua == "" {
			continue
		}
		bd.TotalEvents++

		parsed := parseUserAgent(ua)
		browsers[parsed.Browser]++
		oses[parsed.OS]++
		deviceTypes[parsed.DeviceType]++
	}

	bd.Browsers = mapToBreakdown("browser", browsers, bd.TotalEvents)
	bd.OSes = mapToBreakdown("os", oses, bd.TotalEvents)
	bd.DeviceTypes = mapToBreakdown("device_type", deviceTypes, bd.TotalEvents)

	return bd, nil
}

// extractUserAgent pulls the user-agent string from an event's JSON details.
func extractUserAgent(detailsJSON string) string {
	var details struct {
		Browser map[string]string `json:"browser"`
	}
	if err := json.Unmarshal([]byte(detailsJSON), &details); err != nil {
		return ""
	}
	return details.Browser["user-agent"]
}

// parseUserAgent does a lightweight parse of the User-Agent string.
// We keep this simple and server-side to avoid adding a dependency.
func parseUserAgent(ua string) uaComponents {
	uaLower := strings.ToLower(ua)
	result := uaComponents{
		Browser:    "Unknown",
		OS:         "Unknown",
		DeviceType: "Desktop",
	}

	// ── Browser detection ──
	switch {
	case strings.Contains(uaLower, "edg/") || strings.Contains(uaLower, "edge/"):
		result.Browser = "Edge"
	case strings.Contains(uaLower, "opr/") || strings.Contains(uaLower, "opera"):
		result.Browser = "Opera"
	case strings.Contains(uaLower, "chrome/") && !strings.Contains(uaLower, "edg/"):
		result.Browser = "Chrome"
	case strings.Contains(uaLower, "safari/") && !strings.Contains(uaLower, "chrome/"):
		result.Browser = "Safari"
	case strings.Contains(uaLower, "firefox/"):
		result.Browser = "Firefox"
	case strings.Contains(uaLower, "msie") || strings.Contains(uaLower, "trident/"):
		result.Browser = "Internet Explorer"
	}

	// ── OS detection ──
	switch {
	case strings.Contains(uaLower, "windows"):
		result.OS = "Windows"
	case strings.Contains(uaLower, "mac os x") || strings.Contains(uaLower, "macintosh"):
		result.OS = "macOS"
	case strings.Contains(uaLower, "linux") && !strings.Contains(uaLower, "android"):
		result.OS = "Linux"
	case strings.Contains(uaLower, "android"):
		result.OS = "Android"
	case strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad"):
		result.OS = "iOS"
	case strings.Contains(uaLower, "cros"):
		result.OS = "Chrome OS"
	}

	// ── Device type detection ──
	switch {
	case strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "android") && strings.Contains(uaLower, "mobile"):
		result.DeviceType = "Mobile"
	case strings.Contains(uaLower, "ipad") || (strings.Contains(uaLower, "android") && !strings.Contains(uaLower, "mobile")):
		result.DeviceType = "Tablet"
	default:
		result.DeviceType = "Desktop"
	}

	return result
}

// mapToBreakdown converts a count map to sorted DeviceBreakdownEntry slice.
func mapToBreakdown(category string, m map[string]int, total int) []DeviceBreakdownEntry {
	entries := make([]DeviceBreakdownEntry, 0, len(m))
	for val, count := range m {
		pct := 0.0
		if total > 0 {
			pct = math.Round(float64(count)/float64(total)*1000) / 10
		}
		entries = append(entries, DeviceBreakdownEntry{
			Category: category,
			Value:    val,
			Count:    count,
			Percent:  pct,
		})
	}
	// Sort by count descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})
	return entries
}
