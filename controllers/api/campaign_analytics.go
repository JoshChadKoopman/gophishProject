package api

import (
	"net/http"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// CampaignAnalyticsFunnel returns the phishing funnel for a campaign.
// GET /api/campaigns/:id/analytics/funnel
func (as *Server) CampaignAnalyticsFunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	// Verify org access
	_, err := models.GetCampaignResults(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}

	funnel, err := models.GetCampaignFunnel(id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, funnel, http.StatusOK)
}

// CampaignAnalyticsTimeToClick returns the time-to-click distribution for a campaign.
// GET /api/campaigns/:id/analytics/time-to-click
func (as *Server) CampaignAnalyticsTimeToClick(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	_, err := models.GetCampaignResults(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}

	dist, err := models.GetTimeToClickDistribution(id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, dist, http.StatusOK)
}

// CampaignAnalyticsRepeatOffenders returns repeat offenders for the org, contextualised to a campaign.
// GET /api/campaigns/:id/analytics/repeat-offenders
func (as *Server) CampaignAnalyticsRepeatOffenders(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}
	user := ctx.Get(r, "user").(models.User)

	// Verify org access
	_, err := models.GetCampaignResults(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}

	offenders, err := models.GetCampaignRepeatOffenders(user.OrgId, id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, offenders, http.StatusOK)
}

// CampaignAnalyticsDeviceBreakdown returns device/browser/OS breakdown for a campaign.
// GET /api/campaigns/:id/analytics/devices
func (as *Server) CampaignAnalyticsDeviceBreakdown(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	_, err := models.GetCampaignResults(id, getOrgScope(r))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Campaign not found"}, http.StatusNotFound)
		return
	}

	bd, err := models.GetDeviceBreakdown(id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, bd, http.StatusOK)
}
