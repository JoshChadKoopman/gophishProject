package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── Vishing Scenario Endpoints ──────────────────────────────────

// VishingScenarios handles GET (list) and POST (create) for /api/vishing/scenarios/.
func (as *Server) VishingScenarios(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		scenarios, err := models.GetVishingScenarios(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching vishing scenarios"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, scenarios, http.StatusOK)

	case http.MethodPost:
		var s models.VishingScenario
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		s.OrgId = user.OrgId
		s.UserId = user.Id
		if err := models.PostVishingScenario(&s); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, s, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// VishingScenario handles GET, PUT, DELETE for /api/vishing/scenarios/{id}.
func (as *Server) VishingScenario(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	switch r.Method {
	case http.MethodGet:
		s, err := models.GetVishingScenario(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Scenario not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, s, http.StatusOK)

	case http.MethodPut:
		s, err := models.GetVishingScenario(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Scenario not found"}, http.StatusNotFound)
			return
		}
		var update models.VishingScenario
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		update.Id = s.Id
		update.OrgId = s.OrgId
		update.UserId = s.UserId
		update.CreatedDate = s.CreatedDate
		if err := models.PutVishingScenario(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, update, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteVishingScenario(id, user.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Scenario deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// VishingScenarioLibrary handles GET /api/vishing/scenarios/library
// Returns built-in vishing scenario templates.
func (as *Server) VishingScenarioLibrary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	JSONResponse(w, models.GetVishingScenarioLibrary(), http.StatusOK)
}

// ── Vishing Campaign Endpoints ──────────────────────────────────

// VishingCampaigns handles GET (list) and POST (create) for /api/vishing/campaigns/.
func (as *Server) VishingCampaigns(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		campaigns, err := models.GetVishingCampaigns(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching vishing campaigns"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, campaigns, http.StatusOK)

	case http.MethodPost:
		var c models.VishingCampaign
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		c.OrgId = user.OrgId
		c.UserId = user.Id
		if err := models.PostVishingCampaign(&c); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, c, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// VishingCampaign handles GET, DELETE for /api/vishing/campaigns/{id}.
func (as *Server) VishingCampaign(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	switch r.Method {
	case http.MethodGet:
		c, err := models.GetVishingCampaign(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, c, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteVishingCampaign(id, user.OrgId); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Campaign deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// VishingCampaignStats handles GET /api/vishing/campaigns/{id}/stats.
func (as *Server) VishingCampaignStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)
	stats := models.GetVishingCampaignStats(id)
	JSONResponse(w, stats, http.StatusOK)
}

// VishingResultRecord handles POST /api/vishing/campaigns/{id}/results
// Used by the telephony webhook to record call outcomes.
func (as *Server) VishingResultRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	campaignId, _ := strconv.ParseInt(vars["id"], 10, 64)

	var result models.VishingResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	result.CampaignId = campaignId
	result.OrgId = user.OrgId

	if err := models.RecordVishingResult(&result); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	// Apply BRS penalty/reward based on the call outcome
	go func() {
		// Find user ID from email
		var targetUser models.Target
		if err := models.GetDB().Where("email = ?", result.Email).First(&targetUser).Error; err == nil {
			models.ApplyVishingBRSPenalty(targetUser.Id, result.Status)
		}
	}()

	JSONResponse(w, result, http.StatusCreated)
}

// VishingReportCall handles POST /api/vishing/report
// Called when a user reports a suspicious phone call.
func (as *Server) VishingReportCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)

	type reportReq struct {
		PhoneNumber string `json:"phone_number"`
		Notes       string `json:"notes"`
	}
	var req reportReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	// Check if there's an active vishing result for this user
	var result models.VishingResult
	err := models.GetDB().
		Where("email = ? AND org_id = ? AND reported = ?", user.Username, user.OrgId, false).
		Order("created_date DESC").
		First(&result).Error

	if err != nil {
		// No vishing result found — may be a real phishing call report
		JSONResponse(w, models.Response{Success: true, Message: "Report recorded. No matching simulation found — this may be a real threat."}, http.StatusOK)
		return
	}

	// Mark as reported
	result.Reported = true
	result.Status = models.VishingStatusReported
	if saveErr := models.RecordVishingResult(&result); saveErr != nil {
		log.Error(saveErr)
	}

	// Apply BRS reward
	models.ApplyVishingBRSPenalty(user.Id, models.VishingStatusReported)

	JSONResponse(w, models.Response{
		Success: true,
		Message: "Great catch! This was a vishing simulation. You correctly identified and reported a suspicious call.",
	}, http.StatusOK)
}

// VishingCampaignLaunch handles POST /api/vishing/campaigns/{id}/launch.
func (as *Server) VishingCampaignLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	if err := models.LaunchVishingCampaign(id, user.OrgId); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Vishing campaign launched"}, http.StatusOK)
}

// VishingCampaignComplete handles POST /api/vishing/campaigns/{id}/complete.
func (as *Server) VishingCampaignComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	if err := models.CompleteVishingCampaign(id, user.OrgId); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Campaign marked as completed"}, http.StatusOK)
}

// VishingOrgStats handles GET /api/vishing/stats.
func (as *Server) VishingOrgStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	stats := models.GetVishingOrgStats(user.OrgId)
	JSONResponse(w, stats, http.StatusOK)
}
