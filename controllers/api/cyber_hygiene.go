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

// HygieneDevices handles GET (list user's devices) and POST (register device)
// for /api/hygiene/devices/.
func (as *Server) HygieneDevices(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		devices, err := models.GetUserDevices(user.Id, user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching devices"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, devices, http.StatusOK)

	case http.MethodPost:
		d := &models.UserDevice{}
		if err := json.NewDecoder(r.Body).Decode(d); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		d.UserId = user.Id
		d.OrgId = user.OrgId
		if err := d.Validate(); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		if err := models.PostDevice(d); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error creating device"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, d, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// HygieneDevice handles GET, PUT, DELETE for /api/hygiene/devices/{id}.
func (as *Server) HygieneDevice(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	device, err := models.GetDevice(id, user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Device not found"}, http.StatusNotFound)
		return
	}
	// Users can only manage their own devices (admins can see all via /hygiene/admin/devices)
	modifySystem, _ := user.HasPermission(models.PermissionModifySystem)
	if device.UserId != user.Id && !modifySystem {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		JSONResponse(w, device, http.StatusOK)

	case http.MethodPut:
		var req models.UserDevice
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		device.Name = req.Name
		device.DeviceType = req.DeviceType
		device.OS = req.OS
		if err := device.Validate(); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		if err := models.PutDevice(&device); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error updating device"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, device, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteDevice(id, user.OrgId); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting device"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Device deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// HygieneDeviceChecks handles POST /api/hygiene/devices/{id}/checks
// to upsert a hygiene check result for a device.
func (as *Server) HygieneDeviceChecks(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	// Verify device belongs to this user
	device, err := models.GetDevice(id, user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Device not found"}, http.StatusNotFound)
		return
	}
	if device.UserId != user.Id {
		JSONResponse(w, models.Response{Success: false, Message: "Access denied"}, http.StatusForbidden)
		return
	}

	var req struct {
		CheckType string `json:"check_type"`
		Status    string `json:"status"`
		Note      string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if req.CheckType == "" {
		JSONResponse(w, models.Response{Success: false, Message: "check_type is required"}, http.StatusBadRequest)
		return
	}
	validStatuses := map[string]bool{
		models.HygieneStatusPass:    true,
		models.HygieneStatusFail:    true,
		models.HygieneStatusUnknown: true,
	}
	if !validStatuses[req.Status] {
		JSONResponse(w, models.Response{Success: false, Message: "status must be pass, fail, or unknown"}, http.StatusBadRequest)
		return
	}

	if err := models.UpsertDeviceCheck(id, user.OrgId, req.CheckType, req.Status, req.Note); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error saving check"}, http.StatusInternalServerError)
		return
	}

	// Return the updated device with refreshed checks and score
	updated, _ := models.GetDevice(id, user.OrgId)
	JSONResponse(w, updated, http.StatusOK)
}

// HygieneAdminDevices handles GET /api/hygiene/admin/devices — org-wide device list (admin only).
func (as *Server) HygieneAdminDevices(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	devices, err := models.GetOrgDevices(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching devices"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, devices, http.StatusOK)
}

// HygieneAdminSummary handles GET /api/hygiene/admin/summary — org-wide hygiene stats.
func (as *Server) HygieneAdminSummary(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}

	summary, err := models.GetOrgHygieneEnrichedSummary(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching hygiene summary"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// HygieneTechStack handles GET and POST /api/hygiene/tech-stack.
func (as *Server) HygieneTechStack(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		profile, err := models.GetTechStackProfile(user.Id, user.OrgId)
		if err != nil {
			// No profile yet — return empty
			JSONResponse(w, map[string]interface{}{
				"has_profile": false,
				"profile":     nil,
			}, http.StatusOK)
			return
		}
		JSONResponse(w, map[string]interface{}{
			"has_profile": true,
			"profile":     profile,
		}, http.StatusOK)

	case http.MethodPost:
		p := &models.TechStackProfile{}
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		p.UserId = user.Id
		p.OrgId = user.OrgId
		if err := models.UpsertTechStackProfile(p); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving tech stack profile"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, p, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// HygienePersonalizedChecks handles GET /api/hygiene/personalized-checks.
func (as *Server) HygienePersonalizedChecks(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	checks := models.GetPersonalizedChecks(user.Id, user.OrgId)
	JSONResponse(w, checks, http.StatusOK)
}

// HygieneAdminDevicesEnriched handles GET /api/hygiene/admin/devices-enriched.
func (as *Server) HygieneAdminDevicesEnriched(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	views, err := models.GetOrgDevicesEnriched(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching devices"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, views, http.StatusOK)
}
