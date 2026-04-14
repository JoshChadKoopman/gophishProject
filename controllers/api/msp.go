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

// ──────────────────────────────────────────────────────────────────────────────
// MSP Partner management (superadmin only)
// ──────────────────────────────────────────────────────────────────────────────

// MSPPartners handles GET /api/msp/partners/ (list) and POST (create).
func (as *Server) MSPPartners(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		partners, err := models.GetMSPPartners()
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, partners, http.StatusOK)
	case http.MethodPost:
		p := models.MSPPartner{}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		if err := models.PostMSPPartner(&p); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, p, http.StatusCreated)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// MSPPartner handles GET/PUT/DELETE /api/msp/partners/{id}.
func (as *Server) MSPPartner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	p, err := models.GetMSPPartner(id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		JSONResponse(w, p, http.StatusOK)
	case http.MethodPut:
		update := models.MSPPartner{}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		update.Id = p.Id
		update.CreatedDate = p.CreatedDate
		if err := models.PutMSPPartner(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, update, http.StatusOK)
	case http.MethodDelete:
		if err := models.DeleteMSPPartner(id); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Partner deleted"}, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// MSP Partner-Client management
// ──────────────────────────────────────────────────────────────────────────────

// MSPPartnerClients handles GET /api/msp/partners/{id}/clients (list)
// and POST (add client org).
func (as *Server) MSPPartnerClients(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	partnerId, _ := strconv.ParseInt(vars["id"], 0, 64)

	if err := requirePartnerAccess(r, partnerId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		clients, err := models.GetMSPPartnerClients(partnerId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, clients, http.StatusOK)
	case http.MethodPost:
		var req struct {
			OrgId int64 `json:"org_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		mapping, err := models.AddMSPPartnerClient(partnerId, req.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		// Auto-apply white-label branding to the new client
		_ = models.ApplyWhiteLabelToOrg(partnerId, req.OrgId)
		JSONResponse(w, mapping, http.StatusCreated)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// MSPPartnerClientRemove handles DELETE /api/msp/partners/{id}/clients/{oid}.
func (as *Server) MSPPartnerClientRemove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	partnerId, _ := strconv.ParseInt(vars["id"], 0, 64)
	orgId, _ := strconv.ParseInt(vars["oid"], 0, 64)

	if err := requirePartnerAccess(r, partnerId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusForbidden)
		return
	}

	if r.Method != http.MethodDelete {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	if err := models.RemoveMSPPartnerClient(partnerId, orgId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Client removed from partner"}, http.StatusOK)
}

// ──────────────────────────────────────────────────────────────────────────────
// White-label branding
// ──────────────────────────────────────────────────────────────────────────────

// MSPWhiteLabelConfig handles GET (retrieve) and PUT (save) for the
// authenticated user's org white-label configuration.
// Route: /api/msp/whitelabel/config
func (as *Server) MSPWhiteLabelConfig(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		cfg, err := models.GetWhiteLabelConfig(user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	case http.MethodPut:
		cfg := models.WhiteLabelConfig{}
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		cfg.OrgId = user.OrgId
		if err := models.SaveWhiteLabelConfig(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// MSPWhiteLabelPartnerConfig handles GET/PUT for a partner-level default
// white-label config.
// Route: /api/msp/partners/{id}/whitelabel
func (as *Server) MSPWhiteLabelPartnerConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	partnerId, _ := strconv.ParseInt(vars["id"], 0, 64)

	if err := requirePartnerAccess(r, partnerId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		configs, err := models.GetWhiteLabelConfigsByPartner(partnerId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, configs, http.StatusOK)
	case http.MethodPut:
		cfg := models.WhiteLabelConfig{}
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		cfg.PartnerId = partnerId
		// OrgId=0 means partner-level default
		if cfg.OrgId == 0 {
			cfg.OrgId = 0
		}
		if err := models.SaveWhiteLabelConfig(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// MSPWhiteLabelDelete handles DELETE /api/msp/whitelabel/{id}.
func (as *Server) MSPWhiteLabelDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	if r.Method != http.MethodDelete {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	if err := models.DeleteWhiteLabelConfig(id); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "White-label config deleted"}, http.StatusOK)
}

// ──────────────────────────────────────────────────────────────────────────────
// Partner Portal
// ──────────────────────────────────────────────────────────────────────────────

// MSPPortalDashboard returns the partner portal dashboard for the
// authenticated partner user.
// Route: /api/msp/portal/dashboard
func (as *Server) MSPPortalDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	partner, err := models.GetPartnerForUser(user.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusForbidden)
		return
	}

	dash, err := models.GetMSPPortalDashboard(partner.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, dash, http.StatusOK)
}

// MSPPortalReport returns the cross-client report for the authenticated
// partner user.
// Route: /api/msp/portal/report
func (as *Server) MSPPortalReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	partner, err := models.GetPartnerForUser(user.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusForbidden)
		return
	}

	report, err := models.GetMSPCrossClientReport(partner.Id)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, report, http.StatusOK)
}

// MSPPortalClientDetail returns full detail for a single client managed by
// the authenticated partner.
// Route: /api/msp/portal/clients/{oid}
func (as *Server) MSPPortalClientDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	orgId, _ := strconv.ParseInt(vars["oid"], 0, 64)

	user := ctx.Get(r, "user").(models.User)
	partner, err := models.GetPartnerForUser(user.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusForbidden)
		return
	}

	if !models.IsOrgManagedByPartner(partner.Id, orgId) {
		JSONResponse(w, models.Response{Success: false, Message: "Organization is not managed by your partner account"}, http.StatusForbidden)
		return
	}

	org, err := models.GetOrganization(orgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}

	// Build rich detail
	type clientDetail struct {
		Organization models.Organization      `json:"organization"`
		Features     map[string]bool          `json:"features"`
		UserCount    int                      `json:"user_count"`
		CampaignCnt  int                      `json:"campaign_count"`
		WhiteLabel   *models.WhiteLabelConfig `json:"white_label,omitempty"`
	}

	detail := clientDetail{Organization: org}
	detail.Features = models.GetOrgFeatures(orgId)
	detail.UserCount, _ = models.GetOrgUserCount(orgId)
	detail.CampaignCnt, _ = models.GetOrgCampaignCount(orgId)

	if wl, wErr := models.GetWhiteLabelConfig(orgId); wErr == nil {
		detail.WhiteLabel = &wl
	}

	JSONResponse(w, detail, http.StatusOK)
}

// ──────────────────────────────────────────────────────────────────────────────
// Partner-scoped "impersonate org" — switch context to a client org
// ──────────────────────────────────────────────────────────────────────────────

// MSPSwitchOrg allows a partner admin to switch their working context to one
// of their managed client orgs. Returns a success message; the frontend uses
// this to set a session-level org override.
// Route: POST /api/msp/portal/switch-org
func (as *Server) MSPSwitchOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	user := ctx.Get(r, "user").(models.User)
	partner, err := models.GetPartnerForUser(user.Id)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusForbidden)
		return
	}

	var req struct {
		OrgId int64 `json:"org_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}

	if !models.IsOrgManagedByPartner(partner.Id, req.OrgId) {
		JSONResponse(w, models.Response{
			Success: false,
			Message: "Organization is not managed by your partner account",
		}, http.StatusForbidden)
		return
	}

	// Return org details so frontend can update context
	org, err := models.GetOrganization(req.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}

	JSONResponse(w, struct {
		Success bool                `json:"success"`
		Message string              `json:"message"`
		Org     models.Organization `json:"org"`
	}{
		Success: true,
		Message: "Switched to organization context",
		Org:     org,
	}, http.StatusOK)
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// requirePartnerAccess checks that the authenticated user is either a
// superadmin or the primary admin of the requested partner.
func requirePartnerAccess(r *http.Request, partnerId int64) error {
	user := ctx.Get(r, "user").(models.User)
	if user.Role.Slug == models.RoleSuperAdmin {
		return nil
	}
	p, err := models.GetMSPPartnerByUserId(user.Id)
	if err != nil {
		return models.ErrMSPNotPartner
	}
	if p.Id != partnerId {
		return models.ErrMSPNotPartner
	}
	return nil
}
