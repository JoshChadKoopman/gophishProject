package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EmailAnalysisSystemPrompt instructs the AI to act as an expert email
// security analyst that classifies reported emails and extracts threat
// indicators.
const EmailAnalysisSystemPrompt = `You are an expert email security analyst working for an enterprise security operations team. Your job is to analyze emails that employees have reported as suspicious and determine whether they are legitimate, spam, or phishing attempts.

When analyzing an email you MUST evaluate all of the following dimensions:

1. **Header Analysis**
   - Check for sender spoofing (mismatched envelope-from and header-from)
   - Look for authentication failures (SPF, DKIM, DMARC results if present)
   - Identify suspicious relay hops, unusual originating IPs, or anonymizing infrastructure
   - Flag missing or unusual headers (e.g., missing Message-ID, forged X-Mailer)

2. **Sender Reputation Signals**
   - Evaluate the sender domain age and legitimacy
   - Check for look-alike domains (typosquatting, homoglyph attacks, cousin domains)
   - Identify free webmail usage where corporate mail would be expected
   - Note any display name vs. address mismatches

3. **Social Engineering Patterns**
   - Urgency cues ("act now", "immediately", "within 24 hours")
   - Authority exploitation (impersonating executives, IT, HR, legal)
   - Fear and consequence threats ("account suspended", "legal action")
   - Reward and curiosity lures ("you've won", "see attached invoice")
   - Reciprocity or trust manipulation

4. **URL and Domain Analysis**
   - Identify all URLs in the email body
   - Flag shortened URLs, data URIs, or obfuscated links
   - Detect mismatches between displayed link text and actual href
   - Check for suspicious TLDs or newly registered domains

5. **Business Email Compromise (BEC) Patterns**
   - Wire transfer or payment redirection requests
   - Changes to banking details or vendor payment info
   - Executive impersonation with unusual requests
   - Requests to bypass normal approval processes

6. **Impersonation Detection**
   - Brand impersonation (logos, formatting mimicking known companies)
   - Internal colleague impersonation
   - Vendor or partner impersonation
   - Government or authority impersonation

7. **Language and Content Analysis**
   - Grammar and spelling anomalies inconsistent with the purported sender
   - Unusual tone or phrasing for the claimed sender
   - Mixed languages or machine-translation artifacts
   - Generic greetings where personalization would be expected

You MUST respond with ONLY a valid JSON object (no markdown code fences, no explanation outside the JSON) using this exact schema:
{
  "threat_level": "safe|suspicious|likely_phishing|confirmed_phishing",
  "confidence": 0.0 to 1.0,
  "classification": "phishing|spear_phishing|bec|spam|legitimate|unknown",
  "summary": "A concise 2-4 sentence summary explaining your assessment and the key factors that led to your conclusion.",
  "indicators": [
    {
      "type": "url|domain|ip|email_address|attachment|header_anomaly|language_pattern|urgency_cue|impersonation",
      "value": "The specific indicator found in the email",
      "severity": "info|low|medium|high|critical",
      "description": "Brief explanation of why this indicator is significant"
    }
  ]
}

Guidelines for your assessment:
- "safe" = no indicators of malicious intent, appears to be a legitimate email
- "suspicious" = some minor red flags but insufficient evidence for a phishing determination
- "likely_phishing" = multiple strong indicators of phishing or social engineering
- "confirmed_phishing" = overwhelming evidence of a phishing attack (known malicious patterns, clear spoofing, etc.)
- Confidence should reflect how certain you are in your threat_level assessment
- Always include at least one indicator, even for safe emails (e.g., an "info" level indicator noting the email appears legitimate)
- Order indicators from highest to lowest severity`

// BuildEmailAnalysisPrompt constructs the user prompt containing the email
// data to be analyzed. The prompt is structured so the AI can systematically
// examine each component of the email.
func BuildEmailAnalysisPrompt(headers, body, sender, subject string) string {
	var b strings.Builder
	b.WriteString("Analyze the following reported email for phishing indicators and threats.\n\n")

	fmt.Fprintf(&b, "=== SENDER ===\n%s\n\n", sender)
	fmt.Fprintf(&b, "=== SUBJECT ===\n%s\n\n", subject)

	if headers != "" {
		fmt.Fprintf(&b, "=== EMAIL HEADERS ===\n%s\n\n", headers)
	} else {
		b.WriteString("=== EMAIL HEADERS ===\n[Not available]\n\n")
	}

	if body != "" {
		fmt.Fprintf(&b, "=== EMAIL BODY ===\n%s\n\n", body)
	} else {
		b.WriteString("=== EMAIL BODY ===\n[Not available]\n\n")
	}

	b.WriteString("Provide your analysis as a JSON object following the schema described in your instructions.")
	return b.String()
}

// EmailAnalysisResult represents the parsed JSON response from the AI
// provider after analyzing a reported email.
type EmailAnalysisResult struct {
	ThreatLevel    string                       `json:"threat_level"`
	Confidence     float64                      `json:"confidence"`
	Classification string                       `json:"classification"`
	Summary        string                       `json:"summary"`
	Indicators     []EmailAnalysisResultIndicator `json:"indicators"`
}

// EmailAnalysisResultIndicator represents a single threat indicator
// extracted by the AI from the analyzed email.
type EmailAnalysisResultIndicator struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// ParseEmailAnalysisResponse extracts the structured analysis result from
// the raw AI response content. It handles cases where the LLM wraps the
// JSON in markdown code fences.
func ParseEmailAnalysisResponse(content string) (*EmailAnalysisResult, error) {
	content = strings.TrimSpace(content)

	// Strip markdown code fences if present
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		start := 0
		end := len(lines)
		for i, line := range lines {
			if i == 0 && strings.HasPrefix(line, "```") {
				start = i + 1
				continue
			}
			if strings.TrimSpace(line) == "```" {
				end = i
				break
			}
		}
		content = strings.Join(lines[start:end], "\n")
	}

	content = strings.TrimSpace(content)

	var result EmailAnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w\nRaw content: %.500s", err, content)
	}

	if result.ThreatLevel == "" {
		return nil, fmt.Errorf("LLM response missing 'threat_level' field")
	}
	if result.Classification == "" {
		return nil, fmt.Errorf("LLM response missing 'classification' field")
	}
	if result.Summary == "" {
		return nil, fmt.Errorf("LLM response missing 'summary' field")
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		return nil, fmt.Errorf("LLM response 'confidence' out of range [0,1]: %f", result.Confidence)
	}

	return &result, nil
}
