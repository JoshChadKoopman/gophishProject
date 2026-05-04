package models

import (
	"net/mail"
	"sort"
	"strings"
	"time"
)

// EmailSecurityOps bundles operational-quality metrics for the Email
// Security Dashboard: mean-time-to-remediate, false-positive rate, top
// sender intel, and SLA compliance. Returned by GetEmailSecurityOps.
type EmailSecurityOps struct {
	MTTR          MTTRStats          `json:"mttr"`
	FalsePositive FalsePositiveStats `json:"false_positive"`
	TopSenders    []SenderIntelEntry `json:"top_senders"`
	SLACompliance SLAComplianceStats `json:"sla_compliance"`
}

// MTTRStats captures mean-time-to-remediate in minutes plus a 30-day daily
// trend series. Values are based on completed remediation actions.
type MTTRStats struct {
	AvgMinutes    float64          `json:"avg_minutes"`
	MedianMinutes float64          `json:"median_minutes"`
	SampleSize    int              `json:"sample_size"`
	WindowDays    int              `json:"window_days"`
	DailyTrend    []MTTRTrendPoint `json:"daily_trend"`
}

// MTTRTrendPoint is a single day bucket in the 30-day MTTR trend line.
type MTTRTrendPoint struct {
	Date       string  `json:"date"` // YYYY-MM-DD UTC
	AvgMinutes float64 `json:"avg_minutes"`
	Count      int     `json:"count"`
}

// FalsePositiveStats tracks ticket classification accuracy: how many
// tickets the system (or analysts) flagged as phishing that were later
// reclassified as false positives vs. confirmed threats.
type FalsePositiveStats struct {
	ConfirmedThreats  int     `json:"confirmed_threats"`
	FalsePositives    int     `json:"false_positives"`
	TotalClassified   int     `json:"total_classified"`
	FalsePositiveRate float64 `json:"false_positive_rate"` // 0..1
	WindowDays        int     `json:"window_days"`
}

// SenderIntelEntry is a single row in the "Top Senders" threat-intel
// panel — domains/senders most frequently acted on (blocked, quarantined,
// etc.) in the recent window.
type SenderIntelEntry struct {
	Domain          string `json:"domain"`
	ActionCount     int    `json:"action_count"`
	QuarantineCount int    `json:"quarantine_count"`
	BlockCount      int    `json:"block_count"`
	DeleteCount     int    `json:"delete_count"`
	LastSeen        string `json:"last_seen"` // RFC3339
}

// SLAComplianceStats summarises ticket SLA performance for the gauge
// widget on the Tickets tab.
type SLAComplianceStats struct {
	TotalTickets      int     `json:"total_tickets"`
	WithinSLA         int     `json:"within_sla"`
	Breached          int     `json:"breached"`
	AtRisk            int     `json:"at_risk"` // still open, deadline within 1h
	CompliancePercent float64 `json:"compliance_percent"`
	WindowDays        int     `json:"window_days"`
}

// opsWindowDays is the fixed 30-day look-back window used by the
// operational panels. Matches the UI copy ("last 30 days").
const opsWindowDays = 30

// GetEmailSecurityOps computes the full operational metrics bundle for the
// dashboard in a single call. Heavy queries run against small per-org
// tables so the combined latency stays well under the dashboard budget.
func GetEmailSecurityOps(orgId int64) (EmailSecurityOps, error) {
	since := time.Now().UTC().AddDate(0, 0, -opsWindowDays)

	ops := EmailSecurityOps{}
	ops.MTTR = computeMTTR(orgId, since)
	ops.FalsePositive = computeFalsePositive(orgId, since)
	ops.TopSenders = computeTopSenders(orgId, since)
	ops.SLACompliance = computeSLACompliance(orgId, since)
	return ops, nil
}

// computeMTTR pulls completed remediation actions for the org in the
// window and calculates avg / median resolve time plus a per-day series.
func computeMTTR(orgId int64, since time.Time) MTTRStats {
	var actions []RemediationAction
	db.Where("org_id = ? AND status = ? AND created_date >= ?", orgId, InboxRemStatusCompleted, since).
		Find(&actions)

	stats := MTTRStats{WindowDays: opsWindowDays}
	if len(actions) == 0 {
		stats.DailyTrend = emptyDailyTrend(since, opsWindowDays)
		return stats
	}

	type dayBucket struct {
		totalMinutes float64
		count        int
	}
	days := map[string]*dayBucket{}
	durations := make([]float64, 0, len(actions))

	for _, a := range actions {
		if a.CompletedDate.IsZero() || a.CompletedDate.Before(a.CreatedDate) {
			continue
		}
		d := a.CompletedDate.Sub(a.CreatedDate).Minutes()
		durations = append(durations, d)
		key := a.CreatedDate.UTC().Format("2006-01-02")
		b, ok := days[key]
		if !ok {
			b = &dayBucket{}
			days[key] = b
		}
		b.totalMinutes += d
		b.count++
	}

	stats.SampleSize = len(durations)
	if stats.SampleSize == 0 {
		stats.DailyTrend = emptyDailyTrend(since, opsWindowDays)
		return stats
	}

	sum := 0.0
	for _, d := range durations {
		sum += d
	}
	stats.AvgMinutes = round2(sum / float64(stats.SampleSize))

	sort.Float64s(durations)
	mid := len(durations) / 2
	if len(durations)%2 == 0 {
		stats.MedianMinutes = round2((durations[mid-1] + durations[mid]) / 2)
	} else {
		stats.MedianMinutes = round2(durations[mid])
	}

	// Fill the full window with zero-count days so the chart isn't jagged.
	stats.DailyTrend = emptyDailyTrend(since, opsWindowDays)
	for i, p := range stats.DailyTrend {
		if b, ok := days[p.Date]; ok && b.count > 0 {
			stats.DailyTrend[i] = MTTRTrendPoint{
				Date:       p.Date,
				AvgMinutes: round2(b.totalMinutes / float64(b.count)),
				Count:      b.count,
			}
		}
	}

	return stats
}

// emptyDailyTrend produces one point per day across the window, in
// chronological order, with zero values.
func emptyDailyTrend(since time.Time, days int) []MTTRTrendPoint {
	out := make([]MTTRTrendPoint, 0, days)
	start := time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, time.UTC)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		out = append(out, MTTRTrendPoint{Date: d.Format("2006-01-02")})
	}
	return out
}

// computeFalsePositive counts tickets reclassified as false positive vs.
// confirmed phishing in the window. Tickets still in "pending"
// classification are excluded from the denominator so the rate reflects
// analyst-reviewed outcomes only.
func computeFalsePositive(orgId int64, since time.Time) FalsePositiveStats {
	stats := FalsePositiveStats{WindowDays: opsWindowDays}

	var confirmed, fp, total int
	db.Model(&PhishingTicket{}).
		Where("org_id = ? AND created_date >= ? AND classification = ?",
			orgId, since, "confirmed_phishing").
		Count(&confirmed)
	db.Model(&PhishingTicket{}).
		Where("org_id = ? AND created_date >= ? AND classification = ?",
			orgId, since, "false_positive").
		Count(&fp)
	db.Model(&PhishingTicket{}).
		Where("org_id = ? AND created_date >= ? AND classification NOT IN (?)",
			orgId, since, []string{"pending", ""}).
		Count(&total)

	stats.ConfirmedThreats = confirmed
	stats.FalsePositives = fp
	stats.TotalClassified = total
	if total > 0 {
		stats.FalsePositiveRate = round4(float64(fp) / float64(total))
	}
	return stats
}

// computeTopSenders aggregates remediation actions by sender domain. Only
// actions that actually removed mail from users (quarantine/block/delete)
// are counted — the intent is a "who's hitting us most" feed, not raw
// volume of attempts.
func computeTopSenders(orgId int64, since time.Time) []SenderIntelEntry {
	var actions []RemediationAction
	db.Where("org_id = ? AND created_date >= ? AND sender_email != ''", orgId, since).
		Find(&actions)

	type agg struct {
		domain     string
		total      int
		quarantine int
		block      int
		del        int
		lastSeen   time.Time
	}
	byDomain := map[string]*agg{}
	for _, a := range actions {
		domain := extractDomain(a.SenderEmail)
		if domain == "" {
			continue
		}
		action := strings.ToLower(a.ActionType)
		if action != "quarantine" && action != "block" && action != "delete" && action != "block_sender" {
			continue
		}
		row, ok := byDomain[domain]
		if !ok {
			row = &agg{domain: domain}
			byDomain[domain] = row
		}
		row.total++
		switch action {
		case "quarantine":
			row.quarantine++
		case "block", "block_sender":
			row.block++
		case "delete":
			row.del++
		}
		if a.CreatedDate.After(row.lastSeen) {
			row.lastSeen = a.CreatedDate
		}
	}

	entries := make([]SenderIntelEntry, 0, len(byDomain))
	for _, row := range byDomain {
		entries = append(entries, SenderIntelEntry{
			Domain:          row.domain,
			ActionCount:     row.total,
			QuarantineCount: row.quarantine,
			BlockCount:      row.block,
			DeleteCount:     row.del,
			LastSeen:        row.lastSeen.UTC().Format(time.RFC3339),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ActionCount != entries[j].ActionCount {
			return entries[i].ActionCount > entries[j].ActionCount
		}
		return entries[i].Domain < entries[j].Domain
	})
	if len(entries) > 10 {
		entries = entries[:10]
	}
	return entries
}

// extractDomain returns the domain portion of an email address in lower
// case. Uses net/mail.ParseAddress for RFC-compliant parsing (handles
// display names and angle brackets). Falls back to a simple split for
// plain bare addresses that the RFC parser rejects. Returns "" on failure.
func extractDomain(email string) string {
	addr, err := mail.ParseAddress(email)
	if err == nil {
		email = addr.Address
	}
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(email[at+1:]))
}

// computeSLACompliance counts tickets created in the window as within
// SLA, breached, or at-risk. Open tickets past their deadline are
// breaches; open tickets within one hour of the deadline are at-risk.
func computeSLACompliance(orgId int64, since time.Time) SLAComplianceStats {
	stats := SLAComplianceStats{WindowDays: opsWindowDays}
	var tickets []PhishingTicket
	db.Where("org_id = ? AND created_date >= ?", orgId, since).Find(&tickets)

	now := time.Now().UTC()
	atRiskCutoff := now.Add(1 * time.Hour)
	resolvedStatuses := map[string]bool{
		TicketStatusResolved:     true,
		TicketStatusAutoResolved: true,
		TicketStatusClosed:       true,
	}

	for _, t := range tickets {
		stats.TotalTickets++
		if resolvedStatuses[t.Status] {
			if !t.ResolvedDate.IsZero() && !t.SLADeadline.IsZero() && t.ResolvedDate.After(t.SLADeadline) {
				stats.Breached++
			} else {
				stats.WithinSLA++
			}
			continue
		}
		// Still open / in_progress / escalated
		if !t.SLADeadline.IsZero() && t.SLADeadline.Before(now) {
			stats.Breached++
		} else if !t.SLADeadline.IsZero() && t.SLADeadline.Before(atRiskCutoff) {
			stats.AtRisk++
			stats.WithinSLA++
		} else {
			stats.WithinSLA++
		}
	}

	if stats.TotalTickets > 0 {
		stats.CompliancePercent = round2(float64(stats.WithinSLA) / float64(stats.TotalTickets) * 100)
	}
	return stats
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func round4(v float64) float64 {
	return float64(int64(v*10000+0.5)) / 10000
}
