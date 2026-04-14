package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gophish/gophish/config"
	"github.com/gophish/gophish/mailer"
	"github.com/gophish/gophish/models"
)

type logMailer struct {
	queue chan []mailer.Mail
}

func (m *logMailer) Start(ctx context.Context) {
	// No-op: logMailer is a test double that only captures queued mail.
}

func (m *logMailer) Queue(ms []mailer.Mail) {
	m.queue <- ms
}

// testContext is context to cover API related functions
type testContext struct {
	config *config.Config
}

func setupTest(t *testing.T) *testContext {
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	err := models.Setup(conf)
	if err != nil {
		t.Fatalf("Failed creating database: %v", err)
	}
	ctx := &testContext{}
	ctx.config = conf
	createTestData(t, ctx)
	return ctx
}

func createTestData(t *testing.T, ctx *testContext) {
	ctx.config.TestFlag = true
	// Add a group
	group := models.Group{Name: "Test Group"}
	for i := 0; i < 10; i++ {
		group.Targets = append(group.Targets, models.Target{
			BaseRecipient: models.BaseRecipient{
				Email:     fmt.Sprintf("test%d@example.com", i),
				FirstName: "First",
				LastName:  "Example"}})
	}
	group.UserId = 1
	models.PostGroup(&group)

	// Add a template
	template := models.Template{Name: "Test Template"}
	template.Subject = "Test subject"
	template.Text = "Text text"
	template.HTML = "<html>Test</html>"
	template.UserId = 1
	models.PostTemplate(&template)

	// Add a landing page
	p := models.Page{Name: "Test Page"}
	p.HTML = "<html>Test</html>"
	p.UserId = 1
	models.PostPage(&p)

	// Add a sending profile
	smtp := models.SMTP{Name: "Test Page"}
	smtp.UserId = 1
	smtp.Host = "example.com"
	smtp.FromAddress = "test@test.com"
	models.PostSMTP(&smtp)
}

func workerTestScope(uid int64) models.OrgScope {
	return models.OrgScope{OrgId: 0, UserId: uid, IsSuperAdmin: true}
}

func setupCampaign(id int) (*models.Campaign, error) {
	// Setup and "launch" our campaign
	// Set the status such that no emails are attempted
	c := models.Campaign{Name: fmt.Sprintf("Test campaign - %d", id)}
	c.UserId = 1
	scope := workerTestScope(1)
	template, err := models.GetTemplate(1, scope)
	if err != nil {
		return nil, err
	}
	c.Template = template

	page, err := models.GetPage(1, scope)
	if err != nil {
		return nil, err
	}
	c.Page = page

	smtp, err := models.GetSMTP(1, scope)
	if err != nil {
		return nil, err
	}
	c.SMTP = smtp

	group, err := models.GetGroup(1, scope)
	if err != nil {
		return nil, err
	}
	c.Groups = []models.Group{group}
	err = models.PostCampaign(&c, scope)
	if err != nil {
		return nil, err
	}
	err = c.UpdateStatus(models.CampaignEmailsSent)
	return &c, err
}

// createAndUnlockCampaigns creates n campaigns and unlocks their maillogs.
func createAndUnlockCampaigns(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		campaign, err := setupCampaign(i)
		if err != nil {
			t.Fatalf("error creating campaign: %v", err)
		}
		ms, err := models.GetMailLogsByCampaign(campaign.Id)
		if err != nil {
			t.Fatalf("error getting maillogs for campaign: %v", err)
		}
		for _, m := range ms {
			m.Unlock()
		}
	}
}

// verifyMaillogGrouping reads n batches from the queue and asserts all maillogs
// in each batch share the same campaign ID.
func verifyMaillogGrouping(t *testing.T, queue <-chan []mailer.Mail, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		ms := <-queue
		maillog, ok := ms[0].(*models.MailLog)
		if !ok {
			t.Fatalf("unable to cast mail to models.MailLog")
		}
		expected := maillog.CampaignId
		for _, m := range ms {
			maillog, ok = m.(*models.MailLog)
			if !ok {
				t.Fatalf("unable to cast mail to models.MailLog")
			}
			if got := maillog.CampaignId; got != expected {
				t.Fatalf("unexpected campaign ID received for maillog: got %d expected %d", got, expected)
			}
		}
	}
}

func TestMailLogGrouping(t *testing.T) {
	setupTest(t)

	const campaignCount = 10
	createAndUnlockCampaigns(t, campaignCount)

	lm := &logMailer{queue: make(chan []mailer.Mail)}
	worker := &DefaultWorker{}
	worker.mailer = lm

	// Trigger the worker, generating the maillogs and sending them to the mailer
	worker.processCampaigns(time.Now())

	verifyMaillogGrouping(t, lm.queue, campaignCount)
}
