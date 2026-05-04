package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxResponseBytes caps how many bytes we will read from an AI API response
// body. This prevents a runaway or malicious provider from exhausting memory.
const maxResponseBytes = 1 << 20 // 1 MiB

// Response represents a parsed LLM response with the generated content and
// token usage statistics.
type Response struct {
	Content      string `json:"content"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// Client is the interface that all AI provider implementations must satisfy.
type Client interface {
	// Generate sends a system prompt and user prompt to the LLM and returns
	// the generated text along with token usage.
	Generate(systemPrompt, userPrompt string) (*Response, error)
	// Provider returns the provider name ("claude" or "openai").
	Provider() string
}

// ---------- Claude (Anthropic) ----------

// ClaudeClient calls the Anthropic Messages API.
type ClaudeClient struct {
	APIKey   string
	Model    string
	Endpoint string
	client   *http.Client
}

// NewClaudeClient creates a ClaudeClient with sensible defaults.
func NewClaudeClient(apiKey, model string) *ClaudeClient {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &ClaudeClient{
		APIKey:   apiKey,
		Model:    model,
		Endpoint: "https://api.anthropic.com/v1/messages",
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *ClaudeClient) Provider() string { return "claude" }

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *ClaudeClient) Generate(systemPrompt, userPrompt string) (*Response, error) {
	body := claudeRequest{
		Model:     c.Model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages: []claudeMessage{
			{Role: "user", Content: userPrompt},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ai/claude: marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ai/claude: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai/claude: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("ai/claude: read response: %w", err)
	}

	var cr claudeResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return nil, fmt.Errorf("ai/claude: unmarshal response: %w", err)
	}
	if cr.Error != nil {
		return nil, fmt.Errorf("ai/claude: API error: %s", cr.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai/claude: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	text := ""
	if len(cr.Content) > 0 {
		text = cr.Content[0].Text
	}
	return &Response{
		Content:      text,
		InputTokens:  cr.Usage.InputTokens,
		OutputTokens: cr.Usage.OutputTokens,
	}, nil
}

// ---------- OpenAI ----------

// OpenAIClient calls the OpenAI Chat Completions API.
type OpenAIClient struct {
	APIKey   string
	Model    string
	Endpoint string
	client   *http.Client
}

// NewOpenAIClient creates an OpenAIClient with sensible defaults.
func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIClient{
		APIKey:   apiKey,
		Model:    model,
		Endpoint: "https://api.openai.com/v1/chat/completions",
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *OpenAIClient) Provider() string { return "openai" }

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *OpenAIClient) Generate(systemPrompt, userPrompt string) (*Response, error) {
	body := openaiRequest{
		Model: c.Model,
		Messages: []openaiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ai/openai: marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ai/openai: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai/openai: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("ai/openai: read response: %w", err)
	}

	var or openaiResponse
	if err := json.Unmarshal(respBody, &or); err != nil {
		return nil, fmt.Errorf("ai/openai: unmarshal response: %w", err)
	}
	if or.Error != nil {
		return nil, fmt.Errorf("ai/openai: API error: %s", or.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai/openai: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	text := ""
	if len(or.Choices) > 0 {
		text = or.Choices[0].Message.Content
	}
	return &Response{
		Content:      text,
		InputTokens:  or.Usage.PromptTokens,
		OutputTokens: or.Usage.CompletionTokens,
	}, nil
}

// NewClient creates the appropriate AI client based on provider name.
func NewClient(provider, apiKey, model string) (Client, error) {
	switch provider {
	case "claude":
		return NewClaudeClient(apiKey, model), nil
	case "openai":
		return NewOpenAIClient(apiKey, model), nil
	default:
		return nil, fmt.Errorf("ai: unsupported provider %q (use \"claude\" or \"openai\")", provider)
	}
}
