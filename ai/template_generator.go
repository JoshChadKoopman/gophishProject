package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GenerateRequest contains the parameters for generating a phishing template.
type GenerateRequest struct {
	Prompt           string `json:"prompt"`            // Free-text scenario description
	DifficultyLevel  int    `json:"difficulty_level"`  // 1-4
	Language         string `json:"language"`          // ISO code, default "en"
	TargetRole       string `json:"target_role"`       // e.g. "Software Engineer"
	TargetDepartment string `json:"target_department"` // e.g. "Engineering"
	TargetIndustry   string `json:"target_industry"`   // e.g. "Financial Services"
	SenderName       string `json:"sender_name"`       // Impersonated sender
	CompanyName      string `json:"company_name"`      // Target company
}

// GenerateResult contains the generated template content and metadata.
type GenerateResult struct {
	Subject      string `json:"subject"`
	HTML         string `json:"html"`
	Text         string `json:"text"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
}

// templateJSON is the expected JSON structure from the LLM response.
type templateJSON struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

// GenerateTemplate calls the AI provider to generate a phishing email template.
func GenerateTemplate(client Client, req GenerateRequest) (*GenerateResult, error) {
	if req.DifficultyLevel < DifficultyEasy || req.DifficultyLevel > DifficultySophisticated {
		req.DifficultyLevel = DifficultyMedium
	}
	if req.Language == "" {
		req.Language = "en"
	}

	systemPrompt := GetSystemPrompt(req.DifficultyLevel)
	userPrompt := BuildUserPrompt(req)

	resp, err := client.Generate(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("ai: generation failed: %w", err)
	}

	parsed, err := parseTemplateResponse(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("ai: failed to parse LLM response: %w", err)
	}

	return &GenerateResult{
		Subject:      parsed.Subject,
		HTML:         parsed.HTML,
		Text:         parsed.Text,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		Model:        clientModel(client),
		Provider:     client.Provider(),
	}, nil
}

// parseTemplateResponse extracts the JSON template from the LLM response.
// It handles cases where the LLM wraps the JSON in markdown code fences.
func parseTemplateResponse(content string) (*templateJSON, error) {
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

	var t templateJSON
	if err := json.Unmarshal([]byte(content), &t); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w\nRaw content: %.500s", err, content)
	}

	if t.Subject == "" {
		return nil, fmt.Errorf("LLM response missing 'subject' field")
	}
	if t.HTML == "" && t.Text == "" {
		return nil, fmt.Errorf("LLM response missing both 'html' and 'text' fields")
	}

	return &t, nil
}

// clientModel extracts the model name from the client for logging.
func clientModel(c Client) string {
	switch v := c.(type) {
	case *ClaudeClient:
		return v.Model
	case *OpenAIClient:
		return v.Model
	default:
		return "unknown"
	}
}
