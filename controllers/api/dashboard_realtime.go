package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/websocket"
)

// ── Real-Time Dashboard — WebSocket & Metrics Endpoints ─────────
// Provides:
//   - WebSocket endpoint for live event push
//   - Dashboard metrics with configurable time windows
//   - Dashboard preference save/load
//   - Sparkline data for summary cards

// upgrader configures the WebSocket handshake.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	// Allow connections from the admin SPA origin.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// DashboardWS handles GET /api/dashboard/ws — upgrades to WebSocket.
// The connection streams real-time events scoped to the admin's org.
// The client can send JSON messages to set the desired time window,
// e.g. {"action":"set_window","window":"7d"}.
func (as *Server) DashboardWS(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()

	scope := models.OrgScope{
		OrgId:        user.OrgId,
		UserId:       user.Id,
		IsSuperAdmin: user.Role.Slug == models.RoleSuperAdmin,
	}

	hub := models.GetWSHub()
	ch := hub.Subscribe(user.OrgId)
	defer hub.Unsubscribe(user.OrgId, ch)

	log.Infof("ws: admin %d (org %d) connected — %d subscribers", user.Id, user.OrgId, hub.SubscriberCount(user.OrgId))

	// Send an initial pulse so the client has data immediately.
	sendPulse(conn, scope)

	// Periodic pulse ticker (every 15 s).
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Read pump — handle incoming commands from the client.
	// We run this in a goroutine so we can also select on the event channel.
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// Parse optional client commands.
			var cmd struct {
				Action string `json:"action"`
				Window string `json:"window,omitempty"`
			}
			if json.Unmarshal(msg, &cmd) == nil && cmd.Action == "set_window" {
				tw := models.TimeWindow(cmd.Window)
				if models.ValidTimeWindows[tw] {
					pref := &models.DashboardPreference{
						UserId:     user.Id,
						OrgId:      user.OrgId,
						TimeWindow: cmd.Window,
					}
					models.SaveDashboardPreference(pref)
				}
			}
		}
	}()

	// Write pump — forward hub events + periodic pulses.
	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(evt)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			sendPulse(conn, scope)
		case <-clientDone:
			return
		}
	}
}

// sendPulse writes a dashboard.pulse event with live counts.
func sendPulse(conn *websocket.Conn, scope models.OrgScope) {
	counts := models.GetDashboardLiveCounts(scope)
	evt := models.WSEvent{
		Type:      models.WSEventDashboardPulse,
		OrgId:     scope.OrgId,
		Timestamp: time.Now().UTC(),
		Payload:   counts,
	}
	data, _ := json.Marshal(evt)
	conn.WriteMessage(websocket.TextMessage, data)
}

// DashboardMetrics handles GET /api/dashboard/metrics?window=7d.
// Returns all dashboard cards with sparkline data.
func (as *Server) DashboardMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	scope := getOrgScope(r)

	tw := models.TimeWindow(r.URL.Query().Get("window"))
	if !models.ValidTimeWindows[tw] {
		// Fall back to user's saved preference.
		pref := models.GetDashboardPreference(user.Id)
		tw = models.TimeWindow(pref.TimeWindow)
		if !models.ValidTimeWindows[tw] {
			tw = models.TimeWindow30D
		}
	}

	metrics, err := models.GetDashboardMetrics(scope, tw)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error fetching dashboard metrics"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, metrics, http.StatusOK)
}

// DashboardSparkline handles GET /api/dashboard/sparkline?metric=click_rate&window=7d.
// Returns a single sparkline for a specific metric — useful for lazy-loading cards.
func (as *Server) DashboardSparkline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	metric := r.URL.Query().Get("metric")
	tw := models.TimeWindow(r.URL.Query().Get("window"))
	if !models.ValidTimeWindows[tw] {
		tw = models.TimeWindow7D
	}
	days := models.TimeWindowDays(tw)

	var points []models.SparklinePoint
	switch metric {
	case "emails_sent":
		points = models.BuildEventSparklinePublic(scope, models.EventSent, days)
	case "click_rate":
		points = models.BuildRateSparklinePublic(scope, models.EventClicked, models.EventSent, days)
	case "report_rate":
		points = models.BuildRateSparklinePublic(scope, models.EventReported, models.EventSent, days)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Invalid metric. Use: emails_sent, click_rate, report_rate"}, http.StatusBadRequest)
		return
	}

	JSONResponse(w, struct {
		Metric    string                  `json:"metric"`
		Window    models.TimeWindow       `json:"window"`
		Sparkline []models.SparklinePoint `json:"sparkline"`
	}{metric, tw, points}, http.StatusOK)
}

// DashboardPreference handles GET/PUT /api/dashboard/preference.
func (as *Server) DashboardPreference(w http.ResponseWriter, r *http.Request) {
	user := ctx.Get(r, "user").(models.User)

	switch r.Method {
	case http.MethodGet:
		pref := models.GetDashboardPreference(user.Id)
		JSONResponse(w, pref, http.StatusOK)

	case http.MethodPut:
		var req struct {
			TimeWindow string `json:"time_window"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: ErrInvalidJSON}, http.StatusBadRequest)
			return
		}
		tw := models.TimeWindow(req.TimeWindow)
		if !models.ValidTimeWindows[tw] {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid time window. Use: 7d, 30d, 90d, ytd"}, http.StatusBadRequest)
			return
		}
		pref := &models.DashboardPreference{
			UserId:     user.Id,
			OrgId:      user.OrgId,
			TimeWindow: req.TimeWindow,
		}
		if err := models.SaveDashboardPreference(pref); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error saving preference"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, pref, http.StatusOK)

	default:
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
	}
}

// DashboardLiveCounts handles GET /api/dashboard/live-counts.
// Lightweight endpoint for polling fallback when WebSocket is unavailable.
func (as *Server) DashboardLiveCounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	scope := getOrgScope(r)
	counts := models.GetDashboardLiveCounts(scope)
	JSONResponse(w, counts, http.StatusOK)
}

// DashboardWSStatus handles GET /api/dashboard/ws-status.
// Returns the number of connected WebSocket clients (admin-only).
func (as *Server) DashboardWSStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: ErrMethodNotAllowed}, http.StatusMethodNotAllowed)
		return
	}
	user := ctx.Get(r, "user").(models.User)
	hub := models.GetWSHub()
	JSONResponse(w, struct {
		OrgSubscribers   int `json:"org_subscribers"`
		TotalSubscribers int `json:"total_subscribers"`
	}{
		OrgSubscribers:   hub.SubscriberCount(user.OrgId),
		TotalSubscribers: hub.TotalSubscribers(),
	}, http.StatusOK)
}

// ── Helper: parse "days" query param ──

func parseDaysParam(r *http.Request, defaultVal int) int {
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			return parsed
		}
	}
	return defaultVal
}
