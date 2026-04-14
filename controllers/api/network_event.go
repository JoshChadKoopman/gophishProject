package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// NetworkEvents handles GET /api/network-events/ — returns a filtered list
// of network events for the caller's organization.
func (as *Server) NetworkEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	q := r.URL.Query()

	filter := models.NetworkEventFilter{
		Source:    q.Get("source"),
		EventType: q.Get("event_type"),
		Severity:  q.Get("severity"),
		Status:    q.Get("status"),
		UserEmail: q.Get("user_email"),
	}

	// Parse limit (default 100, max 500).
	filter.Limit = 100
	if v := q.Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			filter.Limit = parsed
		}
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}

	// Parse offset.
	if v := q.Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			filter.Offset = parsed
		}
	}

	// Parse date range.
	if v := q.Get("start_date"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.StartDate = t
		}
	}
	if v := q.Get("end_date"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.EndDate = t
		}
	}

	events, err := models.GetNetworkEvents(scope.OrgId, filter)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching network events"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, events, http.StatusOK)
}

// NetworkEvent handles GET/PUT /api/network-events/{id}.
// GET returns a single event with its notes.
// PUT updates the event status.
func (as *Server) NetworkEvent(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		event, err := models.GetNetworkEvent(id, scope.OrgId)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Network event not found"}, http.StatusNotFound)
			return
		}
		JSONResponse(w, event, http.StatusOK)

	case http.MethodPut:
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		if req.Status == "" {
			JSONResponse(w, models.Response{Success: false, Message: "status is required"}, http.StatusBadRequest)
			return
		}
		if err := models.UpdateNetworkEventStatus(id, scope.OrgId, req.Status, scope.UserId); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error updating network event"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Event status updated"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// NetworkEventIngest handles POST /api/network-events/ingest — ingests a
// single network event.
func (as *Server) NetworkEventIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)

	event := models.NetworkEvent{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	event.OrgId = scope.OrgId

	if err := models.PostNetworkEvent(&event); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error ingesting network event"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, event, http.StatusCreated)
}

// NetworkEventBulkIngest handles POST /api/network-events/bulk-ingest —
// ingests a batch of network events and returns the count of events
// successfully created.
func (as *Server) NetworkEventBulkIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)

	var req struct {
		Events []models.NetworkEvent `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if len(req.Events) == 0 {
		JSONResponse(w, models.Response{Success: false, Message: "No events provided"}, http.StatusBadRequest)
		return
	}

	created, err := models.BulkIngestNetworkEvents(scope.OrgId, req.Events)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error during bulk ingest"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"success":        true,
		"events_created": created,
		"events_total":   len(req.Events),
	}, http.StatusCreated)
}

// NetworkEventAddNote handles POST /api/network-events/{id}/notes — adds an
// analyst note to a network event.
func (as *Server) NetworkEventAddNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}

	// Verify the event exists and belongs to this org.
	if _, err := models.GetNetworkEvent(id, scope.OrgId); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Network event not found"}, http.StatusNotFound)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		JSONResponse(w, models.Response{Success: false, Message: "content is required"}, http.StatusBadRequest)
		return
	}

	note := models.NetworkEventNote{
		EventId: id,
		UserId:  scope.UserId,
		Content: req.Content,
	}
	if err := models.AddNetworkEventNote(&note); err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error adding note"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, note, http.StatusCreated)
}

// NetworkEventDashboard handles GET /api/network-events/dashboard — returns
// the full dashboard aggregation for the caller's organization.
func (as *Server) NetworkEventDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)

	dash, err := models.GetNetworkEventDashboard(scope.OrgId)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching dashboard"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, dash, http.StatusOK)
}

// NetworkEventTrend handles GET /api/network-events/trend?days=30 — returns
// daily event counts. Defaults to 30 days, max 365.
func (as *Server) NetworkEventTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			days = parsed
		}
	}
	if days > 365 {
		days = 365
	}

	trend, err := models.GetNetworkEventTrend(scope.OrgId, days)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching trend data"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, trend, http.StatusOK)
}

// NetworkEventRules handles GET/POST /api/network-events/rules.
// GET lists all automation rules for the org.
// POST creates a new automation rule.
func (as *Server) NetworkEventRules(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)

	switch r.Method {
	case http.MethodGet:
		rules, err := models.GetNetworkEventRules(scope.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching rules"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, rules, http.StatusOK)

	case http.MethodPost:
		rule := models.NetworkEventRule{}
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		rule.OrgId = scope.OrgId
		if err := models.PostNetworkEventRule(&rule); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error creating rule"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, rule, http.StatusCreated)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// NetworkEventRule handles GET/PUT/DELETE /api/network-events/rules/{id}.
func (as *Server) NetworkEventRule(w http.ResponseWriter, r *http.Request) {
	scope := getOrgScope(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid ID"}, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rules, err := models.GetNetworkEventRules(scope.OrgId)
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error fetching rules"}, http.StatusInternalServerError)
			return
		}
		for _, rule := range rules {
			if rule.Id == id {
				JSONResponse(w, rule, http.StatusOK)
				return
			}
		}
		JSONResponse(w, models.Response{Success: false, Message: "Rule not found"}, http.StatusNotFound)

	case http.MethodPut:
		rule := models.NetworkEventRule{}
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		rule.Id = id
		rule.OrgId = scope.OrgId
		if err := models.PutNetworkEventRule(&rule); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error updating rule"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, rule, http.StatusOK)

	case http.MethodDelete:
		if err := models.DeleteNetworkEventRule(id, scope.OrgId); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting rule"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Rule deleted"}, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}
