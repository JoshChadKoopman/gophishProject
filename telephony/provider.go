package telephony

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ── Telephony Provider Interface ────────────────────────────────
// Abstracts voice call operations for vishing simulations.

// CallRequest contains the parameters for placing an outbound call.
type CallRequest struct {
	To              string `json:"to"`               // Target phone number (E.164)
	From            string `json:"from"`              // Caller ID number (E.164)
	CallerIdName    string `json:"caller_id_name"`    // Spoofed caller name
	ScriptURL       string `json:"script_url"`        // URL for TwiML/IVR script
	StatusCallback  string `json:"status_callback"`   // Webhook for call status updates
	RecordingEnabled bool  `json:"recording_enabled"`
	MaxDurationSec  int    `json:"max_duration_sec"`
	MachineDetection string `json:"machine_detection"` // "Enable", "DetectMessageEnd", ""
}

// CallResult contains the response from placing a call.
type CallResult struct {
	CallSid    string `json:"call_sid"`    // Provider's unique call identifier
	Status     string `json:"status"`      // queued, ringing, in-progress, completed, failed
	Duration   int    `json:"duration"`    // Duration in seconds (populated after completion)
	ProviderName string `json:"provider"`
}

// CallStatusUpdate represents a webhook callback from the telephony provider.
type CallStatusUpdate struct {
	CallSid      string `json:"call_sid"`
	Status       string `json:"status"`       // initiated, ringing, answered, completed
	Duration     int    `json:"duration"`
	AnsweredBy   string `json:"answered_by"`  // human, machine
	RecordingURL string `json:"recording_url"`
	Digits       string `json:"digits"`       // DTMF input from user
	SpeechResult string `json:"speech_result"`
}

// Provider is the interface for telephony providers.
type Provider interface {
	// PlaceCall initiates an outbound call.
	PlaceCall(req CallRequest) (*CallResult, error)
	// GetCallStatus retrieves the current status of a call.
	GetCallStatus(callSid string) (*CallResult, error)
	// CancelCall terminates an in-progress call.
	CancelCall(callSid string) error
	// ProviderName returns the provider identifier.
	ProviderName() string
}

// Shared error format strings.
const (
	errTwilioCreateReq = "twilio: create request: %w"
	errTwilioSendReq   = "twilio: send request: %w"
	errTwilioReadResp  = "twilio: read response: %w"
	errTwilioParseResp = "twilio: parse response: %w"
	contentTypeForm    = "application/x-www-form-urlencoded"
)

// ── Twilio Provider ─────────────────────────────────────────────

// TwilioProvider implements Provider using the Twilio REST API.
type TwilioProvider struct {
	AccountSid string
	AuthToken  string
	httpClient *http.Client
}

// NewTwilioProvider creates a new Twilio provider.
func NewTwilioProvider(accountSid, authToken string) *TwilioProvider {
	return &TwilioProvider{
		AccountSid: accountSid,
		AuthToken:  authToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *TwilioProvider) ProviderName() string { return "twilio" }

// twilioCallResponse is the JSON response from Twilio's Calls API.
type twilioCallResponse struct {
	Sid       string `json:"sid"`
	Status    string `json:"status"`
	Duration  string `json:"duration"`
	ErrorCode *int   `json:"error_code"`
	ErrorMsg  string `json:"error_message"`
}

func (t *TwilioProvider) PlaceCall(req CallRequest) (*CallResult, error) {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls.json", t.AccountSid)

	data := url.Values{}
	data.Set("To", req.To)
	data.Set("From", req.From)
	data.Set("Url", req.ScriptURL)

	if req.StatusCallback != "" {
		data.Set("StatusCallback", req.StatusCallback)
		data.Set("StatusCallbackEvent", "initiated ringing answered completed")
	}
	if req.RecordingEnabled {
		data.Set("Record", "true")
	}
	if req.MaxDurationSec > 0 {
		data.Set("Timeout", fmt.Sprintf("%d", req.MaxDurationSec))
	}
	if req.MachineDetection != "" {
		data.Set("MachineDetection", req.MachineDetection)
	}
	if req.CallerIdName != "" {
		data.Set("CallerIdName", req.CallerIdName) // Note: requires verified number
	}

	httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf(errTwilioCreateReq, err)
	}
	httpReq.Header.Set("Content-Type", contentTypeForm)
	httpReq.SetBasicAuth(t.AccountSid, t.AuthToken)

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf(errTwilioSendReq, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(errTwilioReadResp, err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twilio: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var cr twilioCallResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, fmt.Errorf(errTwilioParseResp, err)
	}

	return &CallResult{
		CallSid:      cr.Sid,
		Status:       cr.Status,
		ProviderName: "twilio",
	}, nil
}

func (t *TwilioProvider) GetCallStatus(callSid string) (*CallResult, error) {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls/%s.json",
		t.AccountSid, callSid)

	httpReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf(errTwilioCreateReq, err)
	}
	httpReq.SetBasicAuth(t.AccountSid, t.AuthToken)

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf(errTwilioSendReq, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(errTwilioReadResp, err)
	}

	var cr twilioCallResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, fmt.Errorf(errTwilioParseResp, err)
	}

	return &CallResult{
		CallSid:      cr.Sid,
		Status:       cr.Status,
		ProviderName: "twilio",
	}, nil
}

func (t *TwilioProvider) CancelCall(callSid string) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls/%s.json",
		t.AccountSid, callSid)

	data := url.Values{}
	data.Set("Status", "completed")

	httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf(errTwilioCreateReq, err)
	}
	httpReq.Header.Set("Content-Type", contentTypeForm)
	httpReq.SetBasicAuth(t.AccountSid, t.AuthToken)

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf(errTwilioSendReq, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio: cancel failed status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ── TwiML Script Generation ─────────────────────────────────────
// Generates TwiML XML from vishing scenario scripts.

// ScriptStep is a single step in a vishing IVR script.
type ScriptStep struct {
	Type    string `json:"type"`    // "say", "gather", "pause", "play"
	Text    string `json:"text"`
	Input   string `json:"input"`   // "dtmf", "speech", "dtmf speech"
	Timeout int    `json:"timeout"`
	URL     string `json:"url"`     // For "play" type
}

type twiMLScript struct {
	Steps []ScriptStep `json:"steps"`
}

// GenerateTwiML converts a vishing scenario's JSON script to TwiML XML.
func GenerateTwiML(scriptJSON string, variables map[string]string) (string, error) {
	var script twiMLScript
	if err := json.Unmarshal([]byte(scriptJSON), &script); err != nil {
		return "", fmt.Errorf("parse script: %w", err)
	}

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Response>\n")

	for _, step := range script.Steps {
		text := substituteVariables(step.Text, variables)
		renderTwiMLStep(&b, step, text)
	}

	b.WriteString("</Response>")
	return b.String(), nil
}

// renderTwiMLStep writes a single TwiML step to the builder.
func renderTwiMLStep(b *strings.Builder, step ScriptStep, text string) {
	switch step.Type {
	case "say":
		b.WriteString(fmt.Sprintf("  <Say voice=\"alice\">%s</Say>\n", escapeXML(text)))
	case "gather":
		timeout := step.Timeout
		if timeout <= 0 {
			timeout = 10
		}
		input := step.Input
		if input == "" {
			input = "dtmf speech"
		}
		b.WriteString(fmt.Sprintf("  <Gather input=\"%s\" timeout=\"%d\" speechTimeout=\"auto\">\n", input, timeout))
		if text != "" {
			b.WriteString(fmt.Sprintf("    <Say voice=\"alice\">%s</Say>\n", escapeXML(text)))
		}
		b.WriteString("  </Gather>\n")
	case "pause":
		timeout := step.Timeout
		if timeout <= 0 {
			timeout = 2
		}
		b.WriteString(fmt.Sprintf("  <Pause length=\"%d\"/>\n", timeout))
	case "play":
		if step.URL != "" {
			b.WriteString(fmt.Sprintf("  <Play>%s</Play>\n", escapeXML(step.URL)))
		}
	}
}

func substituteVariables(text string, vars map[string]string) string {
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{{."+k+"}}", v)
	}
	return text
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// ── Provider Factory ────────────────────────────────────────────

// NewProvider creates a telephony provider from the given configuration.
func NewProvider(providerType, accountSid, authToken string) (Provider, error) {
	switch providerType {
	case "twilio":
		return NewTwilioProvider(accountSid, authToken), nil
	default:
		return nil, fmt.Errorf("unsupported telephony provider: %s", providerType)
	}
}
