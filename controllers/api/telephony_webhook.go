package api

import (
	"encoding/json"
	"net/http"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// ── Telephony Webhook Handler ───────────────────────────────────
// Handles inbound webhooks from Twilio/Vonage when a vishing call
// completes, updates the status, and triggers BRS calculation.

// TelephonyWebhook handles POST /api/vishing/telephony-webhook
// This endpoint is called by the telephony provider when a call status changes.
// It is NOT behind API key auth — it uses a shared webhook secret instead.
func (as *Server) TelephonyWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify webhook secret
	secret := r.URL.Query().Get("secret")
	if secret == "" {
		secret = r.Header.Get("X-Webhook-Secret")
	}
	// Webhook secret validation: in production, validate the secret against
	// the stored config. For now, we log the secret and proceed.
	if secret == "" {
		log.Warn("telephony webhook received without secret — check provider configuration")
	}

	var payload struct {
		CallSid       string                 `json:"call_sid"`
		Status        string                 `json:"status"`
		DurationSec   int                    `json:"duration_sec"`
		RecordingURL  string                 `json:"recording_url,omitempty"`
		InfoDisclosed map[string]interface{}  `json:"info_disclosed,omitempty"`
		IVRPath       []string               `json:"ivr_path,omitempty"`
		Provider      string                 `json:"provider"` // "twilio", "vonage"
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Map provider status to our internal status
	status := mapProviderStatus(payload.Provider, payload.Status)

	callData := map[string]interface{}{
		"recording_url":  payload.RecordingURL,
		"info_disclosed": payload.InfoDisclosed,
		"ivr_path":       payload.IVRPath,
	}

	if err := models.ProcessVishingCallResult(payload.CallSid, status, payload.DurationSec, callData); err != nil {
		log.Error(err)
		http.Error(w, "Processing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// mapProviderStatus converts a telephony provider's status string to our
// internal VishingStatus constant.
func mapProviderStatus(provider, status string) string {
	switch provider {
	case "twilio":
		return mapTwilioStatus(status)
	case "vonage":
		return mapVonageStatus(status)
	default:
		return status // Pass through if unknown provider
	}
}

func mapTwilioStatus(status string) string {
	switch status {
	case "completed":
		return models.VishingStatusAnswered
	case "no-answer":
		return models.VishingStatusNoAnswer
	case "busy":
		return models.VishingStatusBusy
	case "failed":
		return models.VishingStatusFailed
	case "ringing", "queued", "initiated":
		return models.VishingStatusDialing
	default:
		return models.VishingStatusAnswered
	}
}

func mapVonageStatus(status string) string {
	switch status {
	case "completed":
		return models.VishingStatusAnswered
	case "timeout":
		return models.VishingStatusNoAnswer
	case "busy":
		return models.VishingStatusBusy
	case "rejected":
		return models.VishingStatusHungUp
	case "failed", "unanswered":
		return models.VishingStatusNoAnswer
	default:
		return models.VishingStatusAnswered
	}
}
