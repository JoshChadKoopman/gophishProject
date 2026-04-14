package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ──────────────────────────────────────────────────────────────────
// BEC Detection Prompt (enhanced for standalone BEC analysis)
// ──────────────────────────────────────────────────────────────────

// BECDetectionSystemPrompt provides specialized instructions for detecting
// business email compromise attacks with a focus on executive impersonation.
const BECDetectionSystemPrompt = `You are a specialized Business Email Compromise (BEC) detection system. Your job is to analyze emails specifically for BEC attack patterns.

BEC attacks to look for:
1. CEO Fraud - Impersonating a CEO/executive to request wire transfers or sensitive data
2. Invoice Fraud - Fake invoices or payment redirect requests from supposed vendors
3. Account Takeover - Emails from compromised legitimate accounts
4. Vendor Impersonation - Fake vendor communications requesting payment changes
5. Payroll Diversion - Requests to change payroll/direct deposit information
6. Data Theft - Requests for W-2s, employee PII, or financial records

Key indicators:
- Display name matches a known executive but email address differs
- Domain is a lookalike (transposed letters, extra characters, different TLD)
- Urgency language ("urgent", "confidential", "do not share", "time-sensitive")
- Financial requests (wire transfer, gift cards, cryptocurrency, bank details)
- Authority-based pressure ("I need this done before end of day", "Don't tell anyone")
- Reply-to differs from sender
- First-time communication pattern

Return a JSON object:
{
  "is_bec": true/false,
  "confidence": 0.0-1.0,
  "attack_type": "ceo_fraud|invoice_fraud|account_takeover|vendor_impersonation|payroll_diversion|data_theft|none",
  "impersonated_name": "Name being impersonated (if applicable)",
  "impersonated_email": "Email being impersonated (if applicable)",
  "actual_sender": "The real sender address",
  "financial_request": true/false,
  "wire_transfer_mentioned": true/false,
  "gift_card_mentioned": true/false,
  "urgency_level": "low|medium|high|critical",
  "summary": "Brief explanation of BEC indicators found",
  "indicators": ["list", "of", "specific", "BEC", "indicators"]
}

Do NOT wrap the JSON in code fences.`

// BECExecutiveContext provides executive details for BEC comparison.
type BECExecutiveContext struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Title      string `json:"title"`
	Department string `json:"department"`
}

// BuildBECDetectionPrompt creates the user prompt for BEC-specific analysis
// including known executive profiles for comparison.
func BuildBECDetectionPrompt(headers, body, sender, subject string, knownExecutives []BECExecutiveContext) string {
	var b strings.Builder
	b.WriteString("Analyze this email for Business Email Compromise (BEC) attacks.\n\n")

	if len(knownExecutives) > 0 {
		b.WriteString("KNOWN EXECUTIVES IN THIS ORGANIZATION:\n")
		for _, exec := range knownExecutives {
			fmt.Fprintf(&b, "- %s <%s> (Title: %s, Dept: %s)\n",
				exec.Name, exec.Email, exec.Title, exec.Department)
		}
		b.WriteString("\nCheck if the sender is impersonating any of these executives.\n\n")
	}

	fmt.Fprintf(&b, "FROM: %s\n", sender)
	fmt.Fprintf(&b, "SUBJECT: %s\n\n", subject)
	if headers != "" {
		b.WriteString("--- EMAIL HEADERS ---\n")
		b.WriteString(headers)
		b.WriteString("\n--- END HEADERS ---\n\n")
	}
	if body != "" {
		b.WriteString("--- EMAIL BODY ---\n")
		b.WriteString(body)
		b.WriteString("\n--- END BODY ---\n")
	}
	return b.String()
}

// ──────────────────────────────────────────────────────────────────
// Graymail Classification Prompt
// ──────────────────────────────────────────────────────────────────

// GraymailClassificationSystemPrompt instructs the AI to classify graymail.
const GraymailClassificationSystemPrompt = `You are an email classification system that identifies "graymail" - emails that are not spam or phishing but are low-priority bulk communications that users often don't want.

Graymail categories:
1. newsletter - Regular newsletters (company updates, industry news, etc.)
2. marketing - Promotional emails, product announcements, discount offers
3. notification - Automated notifications (social media alerts, account activity, shipping updates)
4. social_media - Social network emails (LinkedIn, Twitter/X, Facebook notifications)
5. bulk_promo - Mass promotional campaigns, event invitations
6. auto_generated - System-generated emails (password resets, receipts, confirmations)

Return a JSON object:
{
  "is_graymail": true/false,
  "confidence": 0.0-1.0,
  "category": "newsletter|marketing|notification|social_media|bulk_promo|auto_generated|none",
  "subcategory": "More specific categorization (e.g., 'product_launch', 'weekly_digest')",
  "reasoning": "Brief explanation of why this is/isn't graymail",
  "suggested_action": "label|archive|unsubscribe_suggest|none"
}

Do NOT wrap the JSON in code fences.`

// BuildGraymailClassificationPrompt creates the user prompt for graymail analysis.
func BuildGraymailClassificationPrompt(headers, body, sender, subject string) string {
	var b strings.Builder
	b.WriteString("Classify this email as graymail or not:\n\n")
	fmt.Fprintf(&b, "FROM: %s\n", sender)
	fmt.Fprintf(&b, "SUBJECT: %s\n\n", subject)
	if headers != "" {
		b.WriteString("--- HEADERS ---\n")
		truncatedHeaders := headers
		if len(truncatedHeaders) > 2000 {
			truncatedHeaders = truncatedHeaders[:2000]
		}
		b.WriteString(truncatedHeaders)
		b.WriteString("\n--- END HEADERS ---\n\n")
	}
	if body != "" {
		truncatedBody := body
		if len(truncatedBody) > 3000 {
			truncatedBody = truncatedBody[:3000]
		}
		b.WriteString("--- BODY ---\n")
		b.WriteString(truncatedBody)
		b.WriteString("\n--- END BODY ---\n")
	}
	return b.String()
}

// ──────────────────────────────────────────────────────────────────
// BEC + Graymail Response Parsers
// ──────────────────────────────────────────────────────────────────

// BECDetectionResult is the parsed AI response for BEC-specific analysis.
type BECDetectionResult struct {
	IsBEC                 bool     `json:"is_bec"`
	Confidence            float64  `json:"confidence"`
	AttackType            string   `json:"attack_type"`
	ImpersonatedName      string   `json:"impersonated_name"`
	ImpersonatedEmail     string   `json:"impersonated_email"`
	ActualSender          string   `json:"actual_sender"`
	FinancialRequest      bool     `json:"financial_request"`
	WireTransferMentioned bool     `json:"wire_transfer_mentioned"`
	GiftCardMentioned     bool     `json:"gift_card_mentioned"`
	UrgencyLevel          string   `json:"urgency_level"`
	Summary               string   `json:"summary"`
	Indicators            []string `json:"indicators"`
}

// ParseBECDetectionResponse parses the AI's BEC analysis response.
func ParseBECDetectionResponse(content string) (*BECDetectionResult, error) {
	content = cleanBECJSONResponse(content)
	var result BECDetectionResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse BEC detection response: %w", err)
	}
	return &result, nil
}

// GraymailClassificationResult is the parsed AI response for graymail analysis.
type GraymailClassificationResult struct {
	IsGraymail      bool    `json:"is_graymail"`
	Confidence      float64 `json:"confidence"`
	Category        string  `json:"category"`
	Subcategory     string  `json:"subcategory"`
	Reasoning       string  `json:"reasoning"`
	SuggestedAction string  `json:"suggested_action"`
}

// ParseGraymailClassificationResponse parses the AI's graymail response.
func ParseGraymailClassificationResponse(content string) (*GraymailClassificationResult, error) {
	content = cleanBECJSONResponse(content)
	var result GraymailClassificationResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse graymail classification response: %w", err)
	}
	return &result, nil
}

// cleanBECJSONResponse strips markdown code fences and trims whitespace.
func cleanBECJSONResponse(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		start := 1
		end := len(lines)
		for i := len(lines) - 1; i > 0; i-- {
			if strings.TrimSpace(lines[i]) == "```" {
				end = i
				break
			}
		}
		content = strings.Join(lines[start:end], "\n")
	}
	return strings.TrimSpace(content)
}
