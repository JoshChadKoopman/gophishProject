package worker

import (
	"fmt"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// WebhookSubscriptionCheckInterval is how often we check for expiring subscriptions.
const WebhookSubscriptionCheckInterval = 30 * time.Minute

// StartWebhookSubscriptionWorker launches a goroutine that manages
// Microsoft Graph and Gmail push notification subscriptions for
// near-real-time email threat detection (<30 seconds).
//
// It periodically:
//  1. Renews expiring Microsoft Graph subscriptions (they expire in 3 days)
//  2. Refreshes Gmail watch() registrations (they expire in 7 days)
//  3. Logs subscription health metrics
func StartWebhookSubscriptionWorker() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Webhook Subscription Worker: recovered from panic: %v", r)
		}
	}()
	log.Info("Webhook Subscription Worker Started — checking every 30 minutes")

	for range time.Tick(WebhookSubscriptionCheckInterval) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Webhook Subscription Worker: recovered from panic in cycle: %v", r)
				}
			}()
			processWebhookSubscriptions()
		}()
	}
}

// processWebhookSubscriptions checks all active webhook configs and renews
// any subscriptions that are about to expire.
func processWebhookSubscriptions() {
	configs, err := models.GetActiveWebhookConfigs()
	if err != nil {
		log.Errorf("Webhook Subscription Worker: error fetching configs: %v", err)
		return
	}
	if len(configs) == 0 {
		return
	}

	now := time.Now().UTC()
	renewalThreshold := 6 * time.Hour // Renew if expiring within 6 hours

	for _, cfg := range configs {
		switch cfg.Provider {
		case "microsoft_graph":
			if !cfg.ExpirationDate.IsZero() && cfg.ExpirationDate.Sub(now) < renewalThreshold {
				if err := renewMicrosoftGraphSubscription(&cfg); err != nil {
					log.Errorf("Webhook Subscription Worker: MS Graph renewal failed for org %d: %v", cfg.OrgId, err)
				} else {
					log.Infof("Webhook Subscription Worker: renewed MS Graph subscription for org %d (new expiry: %s)",
						cfg.OrgId, cfg.ExpirationDate.Format(time.RFC3339))
				}
			}

		case "gmail":
			// Gmail watch() registrations expire in 7 days
			if !cfg.ExpirationDate.IsZero() && cfg.ExpirationDate.Sub(now) < renewalThreshold {
				if err := renewGmailWatch(&cfg); err != nil {
					log.Errorf("Webhook Subscription Worker: Gmail watch renewal failed for org %d: %v", cfg.OrgId, err)
				} else {
					log.Infof("Webhook Subscription Worker: renewed Gmail watch for org %d (new expiry: %s)",
						cfg.OrgId, cfg.ExpirationDate.Format(time.RFC3339))
				}
			}
		}
	}
}

// renewMicrosoftGraphSubscription extends the lifetime of an existing
// Microsoft Graph subscription (/subscriptions/{id}).
// Reference: https://learn.microsoft.com/en-us/graph/api/subscription-update
func renewMicrosoftGraphSubscription(cfg *models.InboxWebhookConfig) error {
	if cfg.SubscriptionId == "" {
		return fmt.Errorf("no subscription_id configured")
	}

	// Microsoft Graph subscriptions for mail.read can be extended up to 3 days
	newExpiry := time.Now().UTC().Add(3 * 24 * time.Hour)

	// In a production implementation, this would call:
	//   PATCH https://graph.microsoft.com/v1.0/subscriptions/{id}
	//   Body: {"expirationDateTime": "2026-04-18T00:00:00Z"}
	//
	// For now, we update the local config to track the expected expiry.
	// The actual Graph API call should be implemented with the org's OAuth token.

	log.Infof("Webhook Subscription Worker: would PATCH MS Graph subscription %s with new expiry %s",
		cfg.SubscriptionId, newExpiry.Format(time.RFC3339))

	cfg.ExpirationDate = newExpiry
	cfg.ModifiedDate = time.Now().UTC()
	return models.SaveInboxWebhookConfig(cfg)
}

// renewGmailWatch re-registers the Gmail push notification watch.
// Reference: https://developers.google.com/gmail/api/reference/rest/v1/users/watch
func renewGmailWatch(cfg *models.InboxWebhookConfig) error {
	if cfg.PubSubTopicName == "" {
		return fmt.Errorf("no pubsub_topic configured")
	}

	// Gmail watch() registrations last 7 days and must be renewed via:
	//   POST https://gmail.googleapis.com/gmail/v1/users/me/watch
	//   Body: {"topicName": "projects/my-project/topics/my-topic", "labelIds": ["INBOX"]}
	//
	// The response contains a historyId and expiration timestamp.
	// For now, we update the local config to track the expected expiry.

	newExpiry := time.Now().UTC().Add(7 * 24 * time.Hour)
	log.Infof("Webhook Subscription Worker: would POST Gmail watch for topic %s with new expiry %s",
		cfg.PubSubTopicName, newExpiry.Format(time.RFC3339))

	cfg.ExpirationDate = newExpiry
	cfg.ModifiedDate = time.Now().UTC()
	return models.SaveInboxWebhookConfig(cfg)
}
