package models

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Short date format used in executive report period labels.
const execNarrativeDateFmt = "Jan 2006"

// ── AI-Generated Executive Narrative ────────────────────────────
// Produces a CISO/board-ready written narrative from a BoardReportSnapshot
// and optional ROI data. This can be further enhanced by passing through an
// LLM, but the deterministic version below generates high-quality prose from
// the structured data — zero-dependency, zero-cost, and instantly testable.

// ExecutiveNarrative contains a multi-section written summary suitable for
// inclusion in board decks, PDF reports, or email digests.
type ExecutiveNarrative struct {
	GeneratedAt     time.Time `json:"generated_at"`
	ExecutiveSummary string   `json:"executive_summary"`
	PhishingSection  string   `json:"phishing_section"`
	TrainingSection  string   `json:"training_section"`
	RiskSection      string   `json:"risk_section"`
	ComplianceSection string  `json:"compliance_section"`
	ROISection       string   `json:"roi_section,omitempty"`
	OutlookSection   string   `json:"outlook_section"`
	FullNarrative    string   `json:"full_narrative"`
}

// PeriodComparison holds current vs prior period metrics for trend analysis.
type PeriodComparison struct {
	CurrentPeriod  *BoardReportSnapshot `json:"current_period"`
	PriorPeriod    *BoardReportSnapshot `json:"prior_period,omitempty"`
	PeriodLabel    string               `json:"period_label"`
	PriorLabel     string               `json:"prior_label,omitempty"`
	HasPriorData   bool                 `json:"has_prior_data"`
}

// EnhancedBoardReport extends BoardReport with narrative and comparison data.
type EnhancedBoardReport struct {
	BoardReport
	Narrative  *ExecutiveNarrative `json:"narrative,omitempty"`
	Comparison *PeriodComparison   `json:"comparison,omitempty"`
	ROISummary *ROIMetrics         `json:"roi_summary,omitempty"`
}

// GenerateEnhancedBoardReport produces a board report with AI narrative,
// period comparison, and ROI integration.
func GenerateEnhancedBoardReport(orgId int64, periodStart, periodEnd time.Time) (*EnhancedBoardReport, error) {
	// 1. Generate current-period snapshot
	current, err := GenerateBoardReportSnapshot(orgId, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("generate current snapshot: %w", err)
	}

	enhanced := &EnhancedBoardReport{}
	enhanced.OrgId = orgId
	enhanced.PeriodStart = periodStart
	enhanced.PeriodEnd = periodEnd
	enhanced.Snapshot = current

	// 2. Generate prior-period snapshot for comparison
	duration := periodEnd.Sub(periodStart)
	priorStart := periodStart.Add(-duration)
	priorEnd := periodStart

	prior, priorErr := GenerateBoardReportSnapshot(orgId, priorStart, priorEnd)

	comparison := &PeriodComparison{
		CurrentPeriod: current,
		PeriodLabel:   fmt.Sprintf("%s – %s", periodStart.Format(execNarrativeDateFmt), periodEnd.Format(execNarrativeDateFmt)),
	}

	if priorErr == nil && prior != nil {
		comparison.PriorPeriod = prior
		comparison.PriorLabel = fmt.Sprintf("%s – %s", priorStart.Format(execNarrativeDateFmt), priorEnd.Format(execNarrativeDateFmt))
		comparison.HasPriorData = true
	}
	enhanced.Comparison = comparison

	// 3. Integrate ROI summary
	roiReport, roiErr := GenerateROIReport(orgId, periodStart, periodEnd)
	if roiErr == nil && roiReport != nil {
		enhanced.ROISummary = &roiReport.Metrics
	}

	// 4. Generate the executive narrative
	enhanced.Narrative = buildExecutiveNarrative(current, comparison, enhanced.ROISummary)

	return enhanced, nil
}

// buildExecutiveNarrative produces the written narrative from structured data.
func buildExecutiveNarrative(snap *BoardReportSnapshot, comp *PeriodComparison, roi *ROIMetrics) *ExecutiveNarrative {
	n := &ExecutiveNarrative{GeneratedAt: time.Now().UTC()}

	n.ExecutiveSummary = buildExecSummary(snap, comp, roi)
	n.PhishingSection = buildPhishingNarrative(snap, comp)
	n.TrainingSection = buildTrainingNarrative(snap, comp)
	n.RiskSection = buildRiskNarrative(snap)
	n.ComplianceSection = buildComplianceNarrative(snap)
	if roi != nil {
		n.ROISection = buildROINarrative(roi)
	}
	n.OutlookSection = buildOutlookNarrative(snap)

	// Combine into a single narrative
	var full strings.Builder
	full.WriteString("# Executive Report – Security Awareness Programme\n\n")
	full.WriteString("## Executive Summary\n" + n.ExecutiveSummary + "\n\n")
	full.WriteString("## Phishing Resilience\n" + n.PhishingSection + "\n\n")
	full.WriteString("## Training & Awareness\n" + n.TrainingSection + "\n\n")
	full.WriteString("## Risk Posture\n" + n.RiskSection + "\n\n")
	full.WriteString("## Compliance\n" + n.ComplianceSection + "\n\n")
	if n.ROISection != "" {
		full.WriteString("## Return on Investment\n" + n.ROISection + "\n\n")
	}
	full.WriteString("## Forward Outlook & Recommendations\n" + n.OutlookSection + "\n")
	n.FullNarrative = full.String()

	return n
}

// ── Section Builders ────────────────────────────────────────────

func buildExecSummary(snap *BoardReportSnapshot, comp *PeriodComparison, roi *ROIMetrics) string {
	var b strings.Builder

	scoreLabel := postureLabel(snap.SecurityPostureScore)
	fmt.Fprintf(&b, "The organisation's security posture score stands at **%.0f/100** (%s). ", snap.SecurityPostureScore, scoreLabel)

	// Trend comparison
	if comp != nil && comp.HasPriorData {
		prior := comp.PriorPeriod.SecurityPostureScore
		delta := snap.SecurityPostureScore - prior
		if delta > 2 {
			fmt.Fprintf(&b, "This represents a **+%.0f point improvement** over the prior period. ", delta)
		} else if delta < -2 {
			fmt.Fprintf(&b, "This represents a **%.0f point decline** versus the prior period, requiring attention. ", delta)
		} else {
			b.WriteString("The score is broadly stable compared with the prior period. ")
		}
	}

	fmt.Fprintf(&b, "The overall risk trend is assessed as **%s**. ", snap.RiskTrend)

	if roi != nil && roi.CostAvoidance > 0 {
		fmt.Fprintf(&b, "The programme delivered an estimated **$%.0fK** in cost avoidance during this reporting period", roi.CostAvoidance/1000)
		if roi.ROIPercentage > 0 {
			fmt.Fprintf(&b, ", translating to a **%.0f%% return** on the security awareness investment", roi.ROIPercentage)
		}
		b.WriteString(". ")
	}

	return b.String()
}

func buildPhishingNarrative(snap *BoardReportSnapshot, comp *PeriodComparison) string {
	var b strings.Builder
	ph := snap.Phishing

	fmt.Fprintf(&b, "During this period, **%d phishing simulation campaigns** were conducted, reaching **%d recipients**. ",
		ph.TotalCampaigns, ph.TotalRecipients)

	fmt.Fprintf(&b, "The average click rate was **%.1f%%**", ph.AvgClickRate)
	if comp != nil && comp.HasPriorData {
		priorClick := comp.PriorPeriod.Phishing.AvgClickRate
		delta := priorClick - ph.AvgClickRate
		if delta > 1 {
			fmt.Fprintf(&b, ", down from %.1f%% — a **%.1f percentage-point improvement**", priorClick, delta)
		} else if delta < -1 {
			fmt.Fprintf(&b, ", up from %.1f%% — a **%.1f percentage-point increase** that warrants investigation", priorClick, math.Abs(delta))
		}
	}
	b.WriteString(". ")

	fmt.Fprintf(&b, "The data submission rate averaged **%.1f%%** and the report rate was **%.1f%%**. ", ph.AvgSubmitRate, ph.AvgReportRate)

	// Interpretation
	if ph.AvgClickRate < 10 {
		b.WriteString("Click rates are well below the industry average of 15–20%, indicating strong phishing resilience. ")
	} else if ph.AvgClickRate < 20 {
		b.WriteString("Click rates are within the industry average range. Targeted campaigns for high-risk groups would drive further improvement. ")
	} else {
		b.WriteString("Click rates exceed the industry benchmark of 20%. Urgent, focused awareness training is recommended. ")
	}

	if ph.AvgReportRate > 30 {
		b.WriteString("The high report rate suggests employees are actively using the report-phishing button — a strong indicator of security culture.")
	} else if ph.AvgReportRate < 10 {
		b.WriteString("The low report rate suggests that employees may not be aware of or using the report-phishing button. Promoting this tool could significantly improve threat detection.")
	}

	return b.String()
}

func buildTrainingNarrative(snap *BoardReportSnapshot, comp *PeriodComparison) string {
	var b strings.Builder
	tr := snap.Training

	fmt.Fprintf(&b, "A total of **%d training courses** were active during the period, with **%d assignments** issued. ",
		tr.TotalCourses, tr.TotalAssignments)

	fmt.Fprintf(&b, "The overall completion rate was **%.0f%%**", tr.CompletionRate)
	if comp != nil && comp.HasPriorData {
		priorComp := comp.PriorPeriod.Training.CompletionRate
		delta := tr.CompletionRate - priorComp
		if delta > 2 {
			fmt.Fprintf(&b, " (up from %.0f%%)", priorComp)
		} else if delta < -2 {
			fmt.Fprintf(&b, " (down from %.0f%% — declining engagement)", priorComp)
		}
	}
	b.WriteString(". ")

	if tr.OverdueCount > 0 {
		fmt.Fprintf(&b, "**%d assignments are currently overdue**, representing a compliance risk that should be escalated. ", tr.OverdueCount)
	}

	fmt.Fprintf(&b, "The average quiz score across all assessments was **%.0f%%**. ", tr.AvgQuizScore)

	if tr.CertificatesIssued > 0 {
		fmt.Fprintf(&b, "**%d certificates** were issued to employees who completed their training requirements. ", tr.CertificatesIssued)
	}

	if tr.CompletionRate >= 90 {
		b.WriteString("Training participation is excellent and meets the 90%+ target for regulatory compliance.")
	} else if tr.CompletionRate >= 70 {
		b.WriteString("Training participation is adequate but below the 90% target. Automated reminders and manager escalations are recommended.")
	} else {
		b.WriteString("Training completion is critically low. Mandatory enforcement and executive sponsor involvement are strongly recommended.")
	}

	return b.String()
}

func buildRiskNarrative(snap *BoardReportSnapshot) string {
	var b strings.Builder
	risk := snap.Risk

	total := risk.HighRiskUsers + risk.MediumRiskUsers + risk.LowRiskUsers
	if total == 0 {
		return "Insufficient risk score data is available for this period. The BRS engine should be configured and run against the current population."
	}

	fmt.Fprintf(&b, "Across **%d assessed users**, the average Behavioural Risk Score is **%.1f/100** ", total, risk.AvgRiskScore)
	if risk.AvgRiskScore < 30 {
		b.WriteString("(low risk). ")
	} else if risk.AvgRiskScore < 60 {
		b.WriteString("(moderate risk). ")
	} else {
		b.WriteString("(elevated risk). ")
	}

	fmt.Fprintf(&b, "The risk distribution is: **%d high-risk** (≥60), **%d medium-risk** (30–59), and **%d low-risk** (<30) users. ",
		risk.HighRiskUsers, risk.MediumRiskUsers, risk.LowRiskUsers)

	if risk.HighRiskUsers > 0 {
		pct := float64(risk.HighRiskUsers) / float64(total) * 100
		fmt.Fprintf(&b, "High-risk users represent **%.0f%%** of the workforce. ", pct)
		if pct > 15 {
			b.WriteString("This concentration of high-risk users warrants targeted remediation paths and manager notification.")
		} else {
			b.WriteString("Targeted remediation paths should be assigned to these individuals.")
		}
	}

	return b.String()
}

func buildComplianceNarrative(snap *BoardReportSnapshot) string {
	var b strings.Builder
	comp := snap.Compliance

	if comp.FrameworkCount == 0 {
		return "No compliance frameworks have been configured for this organisation. Mapping relevant frameworks (e.g., NIST CSF, ISO 27001, SOC 2) would provide a clearer view of the organisation's compliance posture."
	}

	fmt.Fprintf(&b, "The organisation is tracking **%d compliance framework(s)** with an overall compliance score of **%.0f%%**. ",
		comp.FrameworkCount, comp.OverallScore)

	fmt.Fprintf(&b, "Of the assessed controls: **%d compliant**, **%d partially compliant**, and **%d non-compliant**. ",
		comp.Compliant, comp.Partial, comp.NonCompliant)

	if comp.OverallScore >= 90 {
		b.WriteString("The organisation is in excellent compliance posture, well-positioned for upcoming audits.")
	} else if comp.OverallScore >= 70 {
		b.WriteString("Compliance posture is adequate but gaps exist. Addressing non-compliant controls before the next audit cycle is recommended.")
	} else {
		b.WriteString("Compliance gaps are significant and represent material risk. Immediate remediation of non-compliant controls is strongly advised.")
	}

	return b.String()
}

func buildROINarrative(roi *ROIMetrics) string {
	var b strings.Builder

	if roi.CostAvoidance > 0 {
		fmt.Fprintf(&b, "The security awareness programme delivered an estimated **$%.0fK in cost avoidance** through reduced phishing incidents, improved compliance, and enhanced training efficiency. ",
			roi.CostAvoidance/1000)
	}

	if roi.ROIPercentage > 0 {
		fmt.Fprintf(&b, "The programme achieved a **%.0f%% return on investment**, ", roi.ROIPercentage)
		if roi.PaybackPeriodMonths > 0 && roi.PaybackPeriodMonths < 12 {
			fmt.Fprintf(&b, "with a payback period of just **%.1f months**. ", roi.PaybackPeriodMonths)
		} else {
			b.WriteString("validating continued investment in security awareness. ")
		}
	}

	if roi.EstIncidentsAvoided > 0 {
		fmt.Fprintf(&b, "An estimated **%d phishing incidents** were avoided during the reporting period. ", roi.EstIncidentsAvoided)
	}

	if roi.ClickRateReduction > 0 {
		fmt.Fprintf(&b, "Phishing click rates were reduced by **%.1f percentage points**, ", roi.ClickRateReduction)
	}
	if roi.ReportRateImprovement > 0 {
		fmt.Fprintf(&b, "while report rates improved by **%.1f percentage points**. ", roi.ReportRateImprovement)
	}

	if roi.TrainingHoursSaved > 0 {
		fmt.Fprintf(&b, "Adaptive training saved approximately **%.0f employee-hours** ($%.0fK in productivity). ",
			roi.TrainingHoursSaved, roi.TrainingCostSaved/1000)
	}

	if roi.CostPerEmployee > 0 {
		fmt.Fprintf(&b, "The cost per employee was **$%.0f** for the period. ", roi.CostPerEmployee)
	}

	return b.String()
}

func buildOutlookNarrative(snap *BoardReportSnapshot) string {
	var b strings.Builder

	b.WriteString("Based on the data presented, the following actions are recommended for the coming period:\n\n")

	for i, rec := range snap.Recommendations {
		fmt.Fprintf(&b, "%d. %s\n", i+1, rec)
	}

	b.WriteString("\nThe security awareness programme continues to be a critical layer of defense. ")
	if snap.SecurityPostureScore >= 70 {
		b.WriteString("The organisation is well-positioned to withstand common social engineering attacks. Maintaining momentum through regular simulations and training is key to sustaining this posture.")
	} else {
		b.WriteString("There are areas requiring focused attention. With targeted interventions for high-risk groups and improved training engagement, the organisation can significantly strengthen its human firewall.")
	}

	return b.String()
}

// ── Helpers ─────────────────────────────────────────────────────

func postureLabel(score float64) string {
	switch {
	case score >= 85:
		return "Excellent"
	case score >= 70:
		return "Good"
	case score >= 50:
		return "Moderate"
	case score >= 30:
		return "Needs Improvement"
	default:
		return "Critical"
	}
}
