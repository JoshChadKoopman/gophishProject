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

// EscalationPolicies handles GET/POST /api/escalation/policies.
func (as *Server) EscalationPolicies(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		policies, err := models.GetEscalationPolicies(user.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching policies"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, policies, http.StatusOK)

	case http.MethodPost:
		p := models.EscalationPolicy{}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		p.OrgId = user.OrgId
		if err := models.PostEscalationPolicy(&p); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error creating policy"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, p, http.StatusCreated)
	}
}

// EscalationPolicy handles GET/PUT/DELETE /api/escalation/policies/{id}.
func (as *Server) EscalationPolicy(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	switch r.Method {
	case http.MethodGet:
		p, err := models.GetEscalationPolicy(id, user.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Policy not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, p, http.StatusOK)

	case http.MethodPut:
		p := models.EscalationPolicy{}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		p.Id = id
		p.OrgId = user.OrgId
		if err := models.PutEscalationPolicy(&p); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error updating policy"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, p, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteEscalationPolicy(id, user.OrgId); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting policy"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Policy deleted"}, http.StatusOK)
	}
}

// EscalationOffenders handles GET /api/escalation/offenders — list repeat offenders.
func (as *Server) EscalationOffenders(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	minFails := 2
	if v := r.URL.Query().Get("min_fails"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			minFails = parsed
		}
	}
	lookback := 90
	if v := r.URL.Query().Get("lookback_days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			lookback = parsed
		}
	}

	offenders, err := models.GetRepeatOffenders(user.OrgId, minFails, lookback)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching offenders"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, offenders, http.StatusOK)
}

// EscalationEvaluate handles POST /api/escalation/evaluate — runs escalation evaluation.
func (as *Server) EscalationEvaluate(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	events, err := models.EvaluateAndEscalate(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error running escalation"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"success":         true,
		"events_created":  len(events),
		"events":          events,
	}, http.StatusOK)
}

// EscalationEvents handles GET /api/escalation/events — list escalation events.
func (as *Server) EscalationEvents(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	status := r.URL.Query().Get("status")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events, err := models.GetEscalationEvents(user.OrgId, status, limit)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching events"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, events, http.StatusOK)
}

// EscalationResolve handles POST /api/escalation/events/{id}/resolve.
func (as *Server) EscalationResolve(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)

	if err := models.ResolveEscalation(id, user.OrgId, user.Id); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error resolving escalation"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Escalation resolved"}, http.StatusOK)
}

// EscalationDashboard handles GET /api/escalation/summary.
func (as *Server) EscalationDashboard(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	summary, err := models.GetEscalationSummary(user.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching summary"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, summary, http.StatusOK)
}
