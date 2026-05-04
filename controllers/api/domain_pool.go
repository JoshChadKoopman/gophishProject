package api

import (
	"encoding/json"
	"net/http"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── Sending Domain Pool ─────────────────────────────────────────
// Pre-configured realistic spoofing domains with automatic rotation,
// warm-up tracking, and health monitoring.

// DomainPool handles GET/POST /api/domain-pool/.
func (as *Server) DomainPool(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		domains, err := models.GetSendingDomains(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching domains"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, domains, http.StatusOK)

	case http.MethodPost:
		var d models.SendingDomain
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		d.OrgId = user.OrgId
		if d.Category == "" {
			d.Category = models.DomainCategoryCustom
		}
		if !models.ValidDomainCategories[d.Category] {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid domain category"}, http.StatusBadRequest)
			return
		}
		if err := models.CreateSendingDomain(&d); err != nil {
			if err == models.ErrDomainExists {
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusConflict)
				return
			}
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error creating domain"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, d, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// DomainPoolItem handles GET/PUT/DELETE /api/domain-pool/{id}.
func (as *Server) DomainPoolItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		d, err := models.GetSendingDomain(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Domain not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, d, http.StatusOK)

	case http.MethodPut:
		d, err := models.GetSendingDomain(id)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Domain not found"}, http.StatusNotFound)
			return
		}
		var update models.SendingDomain
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		update.Id = d.Id
		update.OrgId = d.OrgId
		update.CreatedDate = d.CreatedDate
		update.IsBuiltIn = d.IsBuiltIn
		if err := models.UpdateSendingDomain(&update); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error updating domain"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, update, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteSendingDomain(id); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting domain"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Domain deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// DomainPoolConfig handles GET/PUT /api/domain-pool/config.
func (as *Server) DomainPoolConfigHandler(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	switch r.Method {
	case http.MethodGet:
		cfg := models.GetDomainPoolConfig(user.OrgId)
		JSONResponse(w, cfg, http.StatusOK)
	case http.MethodPut:
		var cfg models.DomainPoolConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		cfg.OrgId = user.OrgId
		if cfg.RotationStrategy != "" && !models.ValidRotationStrategies[cfg.RotationStrategy] {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid rotation strategy"}, http.StatusBadRequest)
			return
		}
		if err := models.SaveDomainPoolConfig(&cfg); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving config"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cfg, http.StatusOK)
	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// DomainPoolSummary handles GET /api/domain-pool/summary.
func (as *Server) DomainPoolSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	summary, err := models.GetDomainPoolSummary(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching summary"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}

// DomainPoolSeed handles POST /api/domain-pool/seed.
// Seeds built-in domains into the org's pool.
func (as *Server) DomainPoolSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	seeded, err := models.SeedBuiltInDomains(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error seeding domains"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Seeded  int    `json:"seeded"`
	}{true, "Domains seeded", seeded}, http.StatusOK)
}

// DomainPoolSelect handles POST /api/domain-pool/select.
// Selects the next domain based on the rotation strategy.
func (as *Server) DomainPoolSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	domain, err := models.SelectNextDomain(user.OrgId)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, domain, http.StatusOK)
}

// DomainPoolWarmup handles POST /api/domain-pool/{id}/warmup.
// Advances a domain's warmup stage.
func (as *Server) DomainPoolWarmup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	id, ok := parseIDParam(w, vars, "id")
	if !ok {
		return
	}
	if err := models.AdvanceWarmup(id); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Warmup stage advanced"}, http.StatusOK)
}
