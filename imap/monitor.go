package imap

// Future improvements for IMAP monitoring:
//   - Implement a per-config counter for consecutive login errors with exponential backoff
//     (e.g. if supplied credentials are incorrect).
//   - Add a "last_login_error" field in the database to surface IMAP failures in the UI.
//   - Add a DB counter for non-campaign emails that the admin should investigate.
//   - Track the number of non-campaign reported emails per User.

import (
	"bytes"
	"context"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jordan-wright/email"

	"github.com/gophish/gophish/models"
)

// Pattern for GoPhish emails e.g ?rid=AbC1234
// We include the optional quoted-printable 3D at the front, just in case decoding fails. e.g ?rid=3DAbC1234
// We also include alternative URL encoded representations of '=' and '?' to handle Microsoft ATP URLs e.g %3Frid%3DAbC1234
var goPhishRegex = regexp.MustCompile(`((\?|%3F)rid(=|%3D)(3D)?([A-Za-z0-9]{7}))`)

// Monitor is a worker that monitors IMAP servers for reported campaign emails
type Monitor struct {
	cancel func()
}

// Monitor.start() checks for campaign emails
// As each account can have its own polling frequency set we need to run one Go routine for
// each, as well as keeping an eye on newly created user accounts.
func (im *Monitor) start(ctx context.Context) {
	usermap := make(map[int64]int) // Keep track of running go routines, one per user. We assume incrementing non-repeating UIDs (for the case where users are deleted and re-added).

	for {
		select {
		case <-ctx.Done():
			return
		default:
			dbusers, err := models.GetUsers() //Slice of all user ids. Each user gets their own IMAP monitor routine.
			if err != nil {
				log.Error(err)
				break
			}
			for _, dbuser := range dbusers {
				if _, ok := usermap[dbuser.Id]; !ok { // If we don't currently have a running Go routine for this user, start one.
					log.Info("Starting new IMAP monitor for user ", dbuser.Username)
					usermap[dbuser.Id] = 1
					go monitor(dbuser.Id, ctx)
				}
			}
			time.Sleep(10 * time.Second) // Every ten seconds we check if a new user has been created
		}
	}
}

// monitor will continuously login to the IMAP settings associated to the supplied user id (if the user account has IMAP settings, and they're enabled.)
// It also verifies the user account exists, and returns if not (for the case of a user being deleted).
func monitor(uid int64, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 1. Check if user exists, if not, return.
			_, err := models.GetUser(uid)
			if err != nil { // Not sure if there's a better way to determine user existence via id.
				log.Info("User ", uid, " seems to have been deleted. Stopping IMAP monitor for this user.")
				return
			}
			// 2. Check if user has IMAP settings.
			imapSettings, err := models.GetIMAP(uid)
			if err != nil {
				log.Error(err)
				break
			}
			if len(imapSettings) > 0 {
				im := imapSettings[0]
				// 3. Check if IMAP is enabled
				if im.Enabled {
					log.Debug("Checking IMAP for user ", uid, ": ", im.Username, " -> ", im.Host)
					checkForNewEmails(im)
					time.Sleep((time.Duration(im.IMAPFreq) - 10) * time.Second) // Subtract 10 to compensate for the default sleep of 10 at the bottom
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
}

// NewMonitor returns a new instance of imap.Monitor
func NewMonitor() *Monitor {
	im := &Monitor{}
	return im
}

// Start launches the IMAP campaign monitor
func (im *Monitor) Start() error {
	log.Info("Starting IMAP monitor manager")
	ctx, cancel := context.WithCancel(context.Background()) // ctx is the derivedContext
	im.cancel = cancel
	go im.start(ctx)
	return nil
}

// Shutdown attempts to gracefully shutdown the IMAP monitor.
func (im *Monitor) Shutdown() error {
	log.Info("Shutting down IMAP monitor manager")
	im.cancel()
	return nil
}

// checkForNewEmails logs into an IMAP account and checks unread emails for the
// rid campaign identifier.
func checkForNewEmails(im models.IMAP) {
	im.Host = im.Host + ":" + strconv.Itoa(int(im.Port)) // Append port
	mailServer := Mailbox{
		Host:             im.Host,
		TLS:              im.TLS,
		IgnoreCertErrors: im.IgnoreCertErrors,
		User:             im.Username,
		Pwd:              im.Password,
		Folder:           im.Folder,
	}

	msgs, err := mailServer.GetUnread(true, false)
	if err != nil {
		log.Error(err)
		return
	}
	// Update last_succesful_login here via im.Host
	err = models.SuccessfulLogin(&im)

	if len(msgs) == 0 {
		log.Debug("No new emails for ", im.Username)
		return
	}

	log.Debugf("%d new emails for %s", len(msgs), im.Username)
	reportingFailed, deleteEmails := processMessages(msgs, im)
	handlePostProcessing(mailServer, reportingFailed, deleteEmails)
}

// processMessages iterates through unread messages and returns SeqNums for
// emails that failed to report and campaign emails to delete.
func processMessages(msgs []Email, im models.IMAP) ([]uint32, []uint32) {
	var reportingFailed []uint32
	var deleteEmails []uint32

	for _, m := range msgs {
		if shouldSkipByDomain(m, im.RestrictDomain) {
			continue
		}

		rids, err := matchEmail(m.Email)
		if err != nil {
			log.Errorf("Error searching email for rids from user '%s': %s", m.Email.From, err.Error())
			continue
		}
		if len(rids) < 1 {
			log.Infof("User '%s' reported email with subject '%s'. This is not a GoPhish campaign; you should investigate it.", m.Email.From, m.Email.Subject)
		}
		failed, toDelete := processRIDs(rids, m, im.DeleteReportedCampaignEmail)
		reportingFailed = append(reportingFailed, failed...)
		deleteEmails = append(deleteEmails, toDelete...)
	}
	return reportingFailed, deleteEmails
}

// shouldSkipByDomain returns true if domain restriction is enabled and the
// sender does not match.
func shouldSkipByDomain(m Email, restrictDomain string) bool {
	if restrictDomain == "" {
		return false
	}
	splitEmail := strings.Split(m.Email.From, "@")
	senderDomain := splitEmail[len(splitEmail)-1]
	if senderDomain != restrictDomain {
		log.Debug("Ignoring email as not from company domain: ", senderDomain)
		return true
	}
	return false
}

// processRIDs reports each rid found in an email and returns SeqNums that
// failed and SeqNums that should be deleted.
func processRIDs(rids map[string]bool, m Email, deleteCampaignEmail bool) ([]uint32, []uint32) {
	var failed, toDelete []uint32
	for rid := range rids {
		log.Infof("User '%s' reported email with rid %s", m.Email.From, rid)
		result, err := models.GetResult(rid)
		if err != nil {
			log.Error("Error reporting GoPhish email with rid ", rid, ": ", err.Error())
			failed = append(failed, m.SeqNum)
			continue
		}
		if err = result.HandleEmailReport(models.EventDetails{}); err != nil {
			log.Error("Error updating GoPhish email with rid ", rid, ": ", err.Error())
			continue
		}
		if deleteCampaignEmail {
			toDelete = append(toDelete, m.SeqNum)
		}
	}
	return failed, toDelete
}

// handlePostProcessing marks failed emails as unread and deletes campaign emails.
func handlePostProcessing(mailServer Mailbox, reportingFailed, deleteEmails []uint32) {
	if len(reportingFailed) > 0 {
		log.Debugf("Marking %d emails as unread as failed to report", len(reportingFailed))
		if err := mailServer.MarkAsUnread(reportingFailed); err != nil {
			log.Error("Unable to mark emails as unread: ", err.Error())
		}
	}
	if len(deleteEmails) > 0 {
		log.Debugf("Deleting %d campaign emails", len(deleteEmails))
		if err := mailServer.DeleteEmails(deleteEmails); err != nil {
			log.Error("Failed to delete emails: ", err.Error())
		}
	}
}

func checkRIDs(em *email.Email, rids map[string]bool) {
	// Check Text and HTML
	emailContent := string(em.Text) + string(em.HTML)
	for _, r := range goPhishRegex.FindAllStringSubmatch(emailContent, -1) {
		newrid := r[len(r)-1]
		if !rids[newrid] {
			rids[newrid] = true
		}
	}
}

// returns a slice of gophish rid paramters found in the email HTML, Text, and attachments
func matchEmail(em *email.Email) (map[string]bool, error) {
	rids := make(map[string]bool)
	checkRIDs(em, rids)

	// Next check each attachment
	for _, a := range em.Attachments {
		ext := filepath.Ext(a.Filename)
		if a.Header.Get("Content-Type") == "message/rfc822" || ext == ".eml" {

			// Let's decode the email
			rawBodyStream := bytes.NewReader(a.Content)
			attachmentEmail, err := email.NewEmailFromReader(rawBodyStream)
			if err != nil {
				return rids, err
			}

			checkRIDs(attachmentEmail, rids)
		}
	}

	return rids, nil
}
