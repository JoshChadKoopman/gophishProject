package models

import (
	"encoding/json"
	"sync"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── WebSocket Event Hub ─────────────────────────────────────────
// Centralised pub/sub bus that workers publish events to and that
// the WebSocket handler broadcasts to connected admin clients.
// Events are scoped to an org so that each admin only receives data
// relevant to their organisation.

// ── Event types ──

// WSEventType identifies the kind of real-time event.
type WSEventType string

const (
	// Campaign events
	WSEventCampaignProgress  WSEventType = "campaign.progress"
	WSEventCampaignCompleted WSEventType = "campaign.completed"
	WSEventEmailSent         WSEventType = "email.sent"
	WSEventEmailOpened       WSEventType = "email.opened"
	WSEventLinkClicked       WSEventType = "link.clicked"
	WSEventDataSubmitted     WSEventType = "data.submitted"
	WSEventEmailReported     WSEventType = "email.reported"

	// Training events
	WSEventTrainingCompleted WSEventType = "training.completed"
	WSEventQuizPassed        WSEventType = "quiz.passed"
	WSEventAssignmentOverdue WSEventType = "assignment.overdue"

	// Ticket / reported-email events
	WSEventTicketCreated  WSEventType = "ticket.created"
	WSEventTicketResolved WSEventType = "ticket.resolved"

	// Compliance / misc
	WSEventComplianceUpdate WSEventType = "compliance.update"
	WSEventRiskScoreChange  WSEventType = "risk_score.change"

	// Dashboard-specific heartbeat (carries current metrics snapshot)
	WSEventDashboardPulse WSEventType = "dashboard.pulse"
)

// WSEvent is a single real-time event published to the hub.
type WSEvent struct {
	Type      WSEventType    `json:"type"`
	OrgId     int64          `json:"org_id"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   interface{}    `json:"payload,omitempty"`
}

// MarshalJSON is a convenience so that Payload stays dynamic.
func (e WSEvent) MarshalJSON() ([]byte, error) {
	type Alias WSEvent
	return json.Marshal(&struct{ Alias }{Alias(e)})
}

// ── Hub ──

// WSHub is a thread-safe publish/subscribe hub.
// Workers publish WSEvent values; WebSocket handlers subscribe.
type WSHub struct {
	mu          sync.RWMutex
	subscribers map[int64]map[chan WSEvent]struct{} // orgId -> set of channels
}

// globalHub is the singleton event hub.
var globalHub = &WSHub{
	subscribers: make(map[int64]map[chan WSEvent]struct{}),
}

// GetWSHub returns the package-level event hub.
func GetWSHub() *WSHub { return globalHub }

// Subscribe returns a channel that will receive events for the given org.
// The caller MUST call Unsubscribe when done (typically via defer).
func (h *WSHub) Subscribe(orgId int64) chan WSEvent {
	ch := make(chan WSEvent, 64) // buffered so slow readers don't block publishers
	h.mu.Lock()
	if h.subscribers[orgId] == nil {
		h.subscribers[orgId] = make(map[chan WSEvent]struct{})
	}
	h.subscribers[orgId][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a channel from the hub and closes it.
func (h *WSHub) Unsubscribe(orgId int64, ch chan WSEvent) {
	h.mu.Lock()
	if subs, ok := h.subscribers[orgId]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(h.subscribers, orgId)
		}
	}
	h.mu.Unlock()
	// Drain remaining messages so publishers never block.
	go func() {
		for range ch {
		}
	}()
	close(ch)
}

// Publish sends an event to all subscribers of the event's org.
// It is non-blocking: if a subscriber's buffer is full the event is dropped
// for that subscriber with a warning.
func (h *WSHub) Publish(evt WSEvent) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	h.mu.RLock()
	subs := h.subscribers[evt.OrgId]
	h.mu.RUnlock()

	for ch := range subs {
		select {
		case ch <- evt:
		default:
			log.Warnf("ws_event: dropping event %s for org %d (subscriber buffer full)", evt.Type, evt.OrgId)
		}
	}
}

// PublishToAll sends an event to every org's subscribers (e.g. system-wide alerts).
func (h *WSHub) PublishToAll(evt WSEvent) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, subs := range h.subscribers {
		for ch := range subs {
			select {
			case ch <- evt:
			default:
			}
		}
	}
}

// SubscriberCount returns the number of active subscribers for an org.
func (h *WSHub) SubscriberCount(orgId int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers[orgId])
}

// TotalSubscribers returns the total number of connected clients across all orgs.
func (h *WSHub) TotalSubscribers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for _, subs := range h.subscribers {
		n += len(subs)
	}
	return n
}

// ── Convenience publisher helpers ──

// PublishCampaignEvent is a shorthand used by campaign processing code.
func PublishCampaignEvent(orgId int64, evtType WSEventType, payload interface{}) {
	GetWSHub().Publish(WSEvent{
		Type:    evtType,
		OrgId:   orgId,
		Payload: payload,
	})
}

// PublishTrainingEvent is a shorthand used by training workers.
func PublishTrainingEvent(orgId int64, evtType WSEventType, payload interface{}) {
	GetWSHub().Publish(WSEvent{
		Type:    evtType,
		OrgId:   orgId,
		Payload: payload,
	})
}

// PublishTicketEvent is a shorthand used by ticket/report-button code.
func PublishTicketEvent(orgId int64, evtType WSEventType, payload interface{}) {
	GetWSHub().Publish(WSEvent{
		Type:    evtType,
		OrgId:   orgId,
		Payload: payload,
	})
}

// campaignOrgCache provides a lightweight in-memory cache for campaign→orgId
// lookups so that event publishing doesn't hit the DB on every single event.
var campaignOrgCache = struct {
	sync.RWMutex
	m map[int64]int64
}{m: make(map[int64]int64)}

// getCampaignOrgId returns the org_id for a given campaign, using a cache.
func getCampaignOrgId(campaignId int64) int64 {
	campaignOrgCache.RLock()
	if orgId, ok := campaignOrgCache.m[campaignId]; ok {
		campaignOrgCache.RUnlock()
		return orgId
	}
	campaignOrgCache.RUnlock()

	var c struct{ OrgId int64 }
	if err := db.Table("campaigns").Select("org_id").Where("id = ?", campaignId).Scan(&c).Error; err != nil {
		return 0
	}
	campaignOrgCache.Lock()
	campaignOrgCache.m[campaignId] = c.OrgId
	campaignOrgCache.Unlock()
	return c.OrgId
}

// PublishResultEvent publishes a WebSocket event for a campaign result action.
// It resolves the campaign's org_id automatically and runs in a goroutine
// so it never blocks the caller.
func PublishResultEvent(campaignId int64, evtType WSEventType, payload interface{}) {
	go func() {
		orgId := getCampaignOrgId(campaignId)
		if orgId == 0 {
			return
		}
		GetWSHub().Publish(WSEvent{
			Type:    evtType,
			OrgId:   orgId,
			Payload: payload,
		})
	}()
}
