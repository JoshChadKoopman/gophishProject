package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test constants to avoid duplicate literal warnings (SonarLint S1192).
const (
	testAPIKey       = "sk-test"
	testHTTPKey      = "test-key"
	testModel        = "test-model"
	errUnexpected    = "unexpected error: %v"
	errTokenMismatch = "token counts mismatch"
)

// ---------- Mock client for unit tests ----------

// mockClient implements the Client interface with a configurable response.
type mockClient struct {
	provider   string
	model      string
	response   *Response
	err        error
	lastSystem string
	lastUser   string
}

func (m *mockClient) Generate(systemPrompt, userPrompt string) (*Response, error) {
	m.lastSystem = systemPrompt
	m.lastUser = userPrompt
	return m.response, m.err
}

func (m *mockClient) Provider() string { return m.provider }

// ---------- Client construction tests ----------

func TestNewClaudeClientDefaults(t *testing.T) {
	c := NewClaudeClient(testAPIKey, "")
	if c.Model != "claude-sonnet-4-20250514" {
		t.Fatalf("expected default model claude-sonnet-4-20250514, got %s", c.Model)
	}
	if c.APIKey != testAPIKey {
		t.Fatalf("expected API key %s, got %s", testAPIKey, c.APIKey)
	}
	if c.Endpoint != "https://api.anthropic.com/v1/messages" {
		t.Fatalf("unexpected endpoint: %s", c.Endpoint)
	}
	if c.Provider() != "claude" {
		t.Fatalf("expected provider 'claude', got %s", c.Provider())
	}
}

func TestNewClaudeClientCustomModel(t *testing.T) {
	c := NewClaudeClient(testAPIKey, "claude-3-opus")
	if c.Model != "claude-3-opus" {
		t.Fatalf("expected model claude-3-opus, got %s", c.Model)
	}
}

func TestNewOpenAIClientDefaults(t *testing.T) {
	c := NewOpenAIClient(testAPIKey, "")
	if c.Model != "gpt-4o" {
		t.Fatalf("expected default model gpt-4o, got %s", c.Model)
	}
	if c.APIKey != testAPIKey {
		t.Fatalf("expected API key %s, got %s", testAPIKey, c.APIKey)
	}
	if c.Endpoint != "https://api.openai.com/v1/chat/completions" {
		t.Fatalf("unexpected endpoint: %s", c.Endpoint)
	}
	if c.Provider() != "openai" {
		t.Fatalf("expected provider 'openai', got %s", c.Provider())
	}
}

func TestNewOpenAIClientCustomModel(t *testing.T) {
	c := NewOpenAIClient(testAPIKey, "gpt-3.5-turbo")
	if c.Model != "gpt-3.5-turbo" {
		t.Fatalf("expected model gpt-3.5-turbo, got %s", c.Model)
	}
}

// ---------- NewClient factory tests ----------

func TestNewClientClaude(t *testing.T) {
	c, err := NewClient("claude", "key", "model")
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if c.Provider() != "claude" {
		t.Fatalf("expected claude provider, got %s", c.Provider())
	}
}

func TestNewClientOpenAI(t *testing.T) {
	c, err := NewClient("openai", "key", "model")
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if c.Provider() != "openai" {
		t.Fatalf("expected openai provider, got %s", c.Provider())
	}
}

func TestNewClientUnsupported(t *testing.T) {
	_, err := NewClient("gemini", "key", "model")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

// ---------- Prompt tests ----------

func TestGetSystemPromptAllLevels(t *testing.T) {
	tests := []struct {
		level    int
		contains string
	}{
		{DifficultyEasy, "LEVEL 1"},
		{DifficultyMedium, "LEVEL 2"},
		{DifficultyHard, "LEVEL 3"},
		{DifficultySophisticated, "LEVEL 4"},
	}
	for _, tt := range tests {
		prompt := GetSystemPrompt(tt.level)
		if prompt == "" {
			t.Fatalf("empty prompt for level %d", tt.level)
		}
		if !containsStr(prompt, tt.contains) {
			t.Fatalf("level %d prompt missing %q", tt.level, tt.contains)
		}
	}
}

func TestGetSystemPromptInvalidLevel(t *testing.T) {
	// Invalid levels should default to Medium
	for _, level := range []int{0, -1, 5, 100} {
		prompt := GetSystemPrompt(level)
		expected := GetSystemPrompt(DifficultyMedium)
		if prompt != expected {
			t.Fatalf("level %d: expected Medium prompt as fallback", level)
		}
	}
}

func TestBuildUserPromptBasic(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Password reset scenario",
	}
	prompt := BuildUserPrompt(req)
	if !containsStr(prompt, "Password reset scenario") {
		t.Fatal("expected prompt to include scenario text")
	}
	if !containsStr(prompt, "JSON") {
		t.Fatal("expected prompt to include JSON instruction")
	}
}

func TestBuildUserPromptAllFields(t *testing.T) {
	req := GenerateRequest{
		Prompt:           "Invoice scam",
		Language:         "nl",
		TargetRole:       "Engineer",
		TargetDepartment: "IT",
		TargetIndustry:   "Finance",
		SenderName:       "John CEO",
		CompanyName:      "Acme Corp",
	}
	prompt := BuildUserPrompt(req)
	for _, expected := range []string{"Invoice scam", "nl", "Engineer", "IT", "Finance", "John CEO", "Acme Corp"} {
		if !containsStr(prompt, expected) {
			t.Fatalf("expected prompt to contain %q", expected)
		}
	}
}

func TestBuildUserPromptSkipsEnglish(t *testing.T) {
	req := GenerateRequest{
		Prompt:   "Test",
		Language: "en",
	}
	prompt := BuildUserPrompt(req)
	// "en" should not produce a Language line since it's the default
	if containsStr(prompt, "Language: Write the email in en") {
		t.Fatal("should not include language line for English")
	}
}

// ---------- Template response parsing tests ----------

func TestParseTemplateResponseValid(t *testing.T) {
	tmpl := templateJSON{
		Subject: "Reset your password",
		HTML:    "<html>Test</html>",
		Text:    "Plain text",
	}
	data, _ := json.Marshal(tmpl)
	parsed, err := parseTemplateResponse(string(data))
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if parsed.Subject != tmpl.Subject {
		t.Fatalf("subject mismatch: %s vs %s", parsed.Subject, tmpl.Subject)
	}
	if parsed.HTML != tmpl.HTML {
		t.Fatal("HTML mismatch")
	}
	if parsed.Text != tmpl.Text {
		t.Fatal("Text mismatch")
	}
}

func TestParseTemplateResponseWithCodeFences(t *testing.T) {
	tmpl := `{"subject":"Test","html":"<b>hi</b>","text":"hi"}`
	wrapped := "```json\n" + tmpl + "\n```"
	parsed, err := parseTemplateResponse(wrapped)
	if err != nil {
		t.Fatalf("unexpected error parsing fenced JSON: %v", err)
	}
	if parsed.Subject != "Test" {
		t.Fatalf("expected subject 'Test', got %q", parsed.Subject)
	}
}

func TestParseTemplateResponseMissingSubject(t *testing.T) {
	data := `{"html":"<b>hi</b>","text":"hi"}`
	_, err := parseTemplateResponse(data)
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
}

func TestParseTemplateResponseMissingBothBodies(t *testing.T) {
	data := `{"subject":"Test"}`
	_, err := parseTemplateResponse(data)
	if err == nil {
		t.Fatal("expected error for missing html and text")
	}
}

func TestParseTemplateResponseInvalidJSON(t *testing.T) {
	_, err := parseTemplateResponse("not json at all")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseTemplateResponseHTMLOnly(t *testing.T) {
	data := `{"subject":"Test","html":"<b>hi</b>"}`
	parsed, err := parseTemplateResponse(data)
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if parsed.HTML != "<b>hi</b>" {
		t.Fatal("html mismatch")
	}
}

func TestParseTemplateResponseTextOnly(t *testing.T) {
	data := `{"subject":"Test","text":"hello"}`
	parsed, err := parseTemplateResponse(data)
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if parsed.Text != "hello" {
		t.Fatal("text mismatch")
	}
}

// ---------- GenerateTemplate integration tests (with mock) ----------

func TestGenerateTemplateSuccess(t *testing.T) {
	tmpl := `{"subject":"Urgent","html":"<p>Click {{.URL}}</p>","text":"Click {{.URL}}"}`
	mock := &mockClient{
		provider: "claude",
		model:    testModel,
		response: &Response{
			Content:      tmpl,
			InputTokens:  100,
			OutputTokens: 200,
		},
	}

	result, err := GenerateTemplate(mock, GenerateRequest{
		Prompt:          "Password reset",
		DifficultyLevel: DifficultyHard,
	})
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if result.Subject != "Urgent" {
		t.Fatalf("expected subject 'Urgent', got %q", result.Subject)
	}
	if result.InputTokens != 100 || result.OutputTokens != 200 {
		t.Fatal(errTokenMismatch)
	}
	if result.Provider != "claude" {
		t.Fatalf("expected provider 'claude', got %q", result.Provider)
	}
}

func TestGenerateTemplateDefaultsDifficulty(t *testing.T) {
	tmpl := `{"subject":"Test","html":"<p>hi</p>","text":"hi"}`
	mock := &mockClient{
		provider: "openai",
		response: &Response{Content: tmpl},
	}

	// Invalid difficulty should be corrected to Medium
	_, err := GenerateTemplate(mock, GenerateRequest{
		Prompt:          "Test",
		DifficultyLevel: 99,
	})
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	// Verify that the system prompt used was for Medium difficulty
	expected := GetSystemPrompt(DifficultyMedium)
	if mock.lastSystem != expected {
		t.Fatal("expected Medium system prompt for invalid difficulty")
	}
}

func TestGenerateTemplateDefaultsLanguage(t *testing.T) {
	tmpl := `{"subject":"Test","html":"<p>hi</p>","text":"hi"}`
	mock := &mockClient{
		provider: "openai",
		response: &Response{Content: tmpl},
	}
	_, err := GenerateTemplate(mock, GenerateRequest{
		Prompt:          "Test",
		DifficultyLevel: DifficultyEasy,
		Language:        "",
	})
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	// Language should not appear as "en" (default, so skipped in prompt)
}

func TestGenerateTemplateClientError(t *testing.T) {
	mock := &mockClient{
		provider: "claude",
		err:      fmt.Errorf("API rate limited"),
	}
	_, err := GenerateTemplate(mock, GenerateRequest{
		Prompt:          "Test",
		DifficultyLevel: DifficultyEasy,
	})
	if err == nil {
		t.Fatal("expected error from client")
	}
	if !containsStr(err.Error(), "generation failed") {
		t.Fatalf("expected 'generation failed' in error, got: %v", err)
	}
}

func TestGenerateTemplateBadResponse(t *testing.T) {
	mock := &mockClient{
		provider: "claude",
		response: &Response{Content: "not valid json"},
	}
	_, err := GenerateTemplate(mock, GenerateRequest{
		Prompt:          "Test",
		DifficultyLevel: DifficultyEasy,
	})
	if err == nil {
		t.Fatal("expected error for bad LLM response")
	}
	if !containsStr(err.Error(), "parse LLM response") {
		t.Fatalf("expected 'parse LLM response' in error, got: %v", err)
	}
}

// ---------- clientModel tests ----------

func TestClientModelClaude(t *testing.T) {
	c := NewClaudeClient("key", "my-claude-model")
	if m := clientModel(c); m != "my-claude-model" {
		t.Fatalf("expected 'my-claude-model', got %q", m)
	}
}

func TestClientModelOpenAI(t *testing.T) {
	c := NewOpenAIClient("key", "my-openai-model")
	if m := clientModel(c); m != "my-openai-model" {
		t.Fatalf("expected 'my-openai-model', got %q", m)
	}
}

func TestClientModelUnknown(t *testing.T) {
	mock := &mockClient{provider: "test"}
	if m := clientModel(mock); m != "unknown" {
		t.Fatalf("expected 'unknown', got %q", m)
	}
}

// ---------- HTTP mock server tests for real API clients ----------

func TestClaudeClientGenerateHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != testHTTPKey {
			t.Error("missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("missing anthropic-version header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
		}

		resp := claudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{{Text: "Hello from Claude"}},
		}
		resp.Usage.InputTokens = 10
		resp.Usage.OutputTokens = 20
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := NewClaudeClient(testHTTPKey, testModel)
	c.Endpoint = ts.URL

	result, err := c.Generate("system", "user")
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if result.Content != "Hello from Claude" {
		t.Fatalf("expected 'Hello from Claude', got %q", result.Content)
	}
	if result.InputTokens != 10 || result.OutputTokens != 20 {
		t.Fatal(errTokenMismatch)
	}
}

func TestClaudeClientGenerateAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{}
		resp.Error = &struct {
			Message string `json:"message"`
		}{Message: "rate limit exceeded"}
		w.WriteHeader(http.StatusOK) // Anthropic returns 200 with error in body sometimes
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := NewClaudeClient(testHTTPKey, testModel)
	c.Endpoint = ts.URL

	_, err := c.Generate("system", "user")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !containsStr(err.Error(), "rate limit exceeded") {
		t.Fatalf("expected rate limit error, got: %v", err)
	}
}

func TestOpenAIClientGenerateHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer "+testHTTPKey {
			t.Error("missing Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
		}

		resp := openaiResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: "Hello from OpenAI"}}},
		}
		resp.Usage.PromptTokens = 15
		resp.Usage.CompletionTokens = 25
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := NewOpenAIClient(testHTTPKey, testModel)
	c.Endpoint = ts.URL

	result, err := c.Generate("system", "user")
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if result.Content != "Hello from OpenAI" {
		t.Fatalf("expected 'Hello from OpenAI', got %q", result.Content)
	}
	if result.InputTokens != 15 || result.OutputTokens != 25 {
		t.Fatal(errTokenMismatch)
	}
}

func TestOpenAIClientGenerateAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openaiResponse{}
		resp.Error = &struct {
			Message string `json:"message"`
		}{Message: "invalid api key"}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := NewOpenAIClient("bad-key", testModel)
	c.Endpoint = ts.URL

	_, err := c.Generate("system", "user")
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !containsStr(err.Error(), "invalid api key") {
		t.Fatalf("expected 'invalid api key' error, got: %v", err)
	}
}

func TestClaudeClientEmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := NewClaudeClient(testHTTPKey, testModel)
	c.Endpoint = ts.URL

	result, err := c.Generate("system", "user")
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if result.Content != "" {
		t.Fatalf("expected empty content, got %q", result.Content)
	}
}

func TestOpenAIClientEmptyChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openaiResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := NewOpenAIClient(testHTTPKey, testModel)
	c.Endpoint = ts.URL

	result, err := c.Generate("system", "user")
	if err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if result.Content != "" {
		t.Fatalf("expected empty content, got %q", result.Content)
	}
}

// ---------- helpers ----------

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
