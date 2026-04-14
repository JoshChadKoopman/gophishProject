package imap

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	"github.com/gophish/gophish/dialer"
	log "github.com/gophish/gophish/logger"
	"github.com/jordan-wright/email"
)

// Shared error format strings to avoid literal duplication.
const (
	errFmtInvalidSeqNum = "invalid IMAP sequence number %q: %w"
	errFmtStoreDeleted  = "imap store deleted flag: %w"
	errFmtExpunge       = "imap expunge: %w"
	headerContentType   = "Content-Type"
)

// ────────────────────────────────────────────────────────────────
// InboxMessage — universal representation of a fetched email
// ────────────────────────────────────────────────────────────────

// InboxMessage is the unified representation of an email message retrieved
// from any mail provider (IMAP, Microsoft Graph, Gmail API).
type InboxMessage struct {
	MessageId    string    `json:"message_id"`
	SenderEmail  string    `json:"sender_email"`
	SenderName   string    `json:"sender_name"`
	Subject      string    `json:"subject"`
	Headers      string    `json:"headers"`
	Body         string    `json:"body"`
	ReceivedDate time.Time `json:"received_date"`
	// Provider-specific identifier used for remediation actions.
	// IMAP: sequence number; Graph/Gmail: provider message ID.
	ProviderUID string `json:"provider_uid"`
}

// ────────────────────────────────────────────────────────────────
// InboxProvider interface
// ────────────────────────────────────────────────────────────────

// InboxProvider abstracts the operations needed to scan and remediate mailboxes
// across IMAP, Microsoft Graph API, and Google Gmail API.
type InboxProvider interface {
	// FetchNewMessages retrieves unread/unseen messages since the given time.
	FetchNewMessages(mailbox string, since time.Time) ([]InboxMessage, error)
	// DeleteMessage permanently removes a message by its provider UID.
	DeleteMessage(mailbox, providerUID string) error
	// QuarantineMessage moves a message to a quarantine/junk folder.
	QuarantineMessage(mailbox, providerUID string) error
	// RestoreMessage moves a message from quarantine back to inbox.
	RestoreMessage(mailbox, providerUID string) error
	// ProviderName returns a human-readable name for logging.
	ProviderName() string
}

// ────────────────────────────────────────────────────────────────
// IMAP Provider
// ────────────────────────────────────────────────────────────────

// IMAPProvider implements InboxProvider using standard IMAP.
type IMAPProvider struct {
	Host             string
	Port             int
	Username         string
	Password         string
	TLS              bool
	IgnoreCertErrors bool
	QuarantineFolder string // defaults to "Junk"
}

func (p *IMAPProvider) ProviderName() string { return "imap" }

func (p *IMAPProvider) dial() (*client.Client, error) {
	addr := p.Host + ":" + strconv.Itoa(p.Port)
	restrictedDialer := dialer.Dialer()
	var imapClient *client.Client
	var err error

	if p.TLS {
		tlsCfg := &tls.Config{InsecureSkipVerify: p.IgnoreCertErrors}
		imapClient, err = client.DialWithDialerTLS(restrictedDialer, addr, tlsCfg)
	} else {
		imapClient, err = client.DialWithDialer(restrictedDialer, addr)
	}
	if err != nil {
		return nil, fmt.Errorf("imap dial %s: %w", addr, err)
	}

	if err = imapClient.Login(p.Username, p.Password); err != nil {
		imapClient.Logout()
		return nil, fmt.Errorf("imap login %s: %w", p.Username, err)
	}
	return imapClient, nil
}

func (p *IMAPProvider) FetchNewMessages(mailbox string, since time.Time) ([]InboxMessage, error) {
	imap.CharsetReader = charset.Reader

	imapClient, err := p.dial()
	if err != nil {
		return nil, err
	}
	defer imapClient.Logout()

	folder := imapFolderFromMailbox(mailbox)
	if _, err := imapClient.Select(folder, true); err != nil {
		return nil, fmt.Errorf("imap select %s: %w", folder, err)
	}

	seqs, err := imapSearchUnseen(imapClient, since)
	if err != nil {
		return nil, err
	}
	if len(seqs) == 0 {
		return nil, nil
	}

	messages, fetchErr := imapFetchMessages(imapClient, seqs)
	results := imapParseMessages(messages)

	if fErr := <-fetchErr; fErr != nil {
		return results, fmt.Errorf("imap fetch error: %w", fErr)
	}
	return results, nil
}

// imapFolderFromMailbox extracts the IMAP folder from a mailbox string.
func imapFolderFromMailbox(mailbox string) string {
	if mailbox != "" && strings.Contains(mailbox, "/") {
		parts := strings.SplitN(mailbox, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return "INBOX"
}

// imapSearchUnseen searches for unseen messages since the given time.
func imapSearchUnseen(c *client.Client, since time.Time) ([]uint32, error) {
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	if !since.IsZero() {
		criteria.Since = since
	}
	seqs, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("imap search: %w", err)
	}
	return seqs, nil
}

// imapFetchMessages starts a background fetch and returns the message channel
// and an error channel.
func imapFetchMessages(c *client.Client, seqs []uint32) (chan *imap.Message, chan error) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(seqs...)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{
		imap.FetchEnvelope, imap.FetchFlags, imap.FetchInternalDate, section.FetchItem(),
	}
	messages := make(chan *imap.Message, 20)
	fetchErr := make(chan error, 1)
	go func() {
		if err := c.Fetch(seqset, items, messages); err != nil {
			fetchErr <- err
		}
		close(fetchErr)
	}()
	return messages, fetchErr
}

// imapParseMessages converts raw IMAP messages to InboxMessage structs.
func imapParseMessages(messages chan *imap.Message) []InboxMessage {
	crRegex := regexp.MustCompile(`\r`)
	var results []InboxMessage

	for msg := range messages {
		parsed := imapParseOneMessage(msg, crRegex)
		if parsed != nil {
			results = append(results, *parsed)
		}
	}
	return results
}

// imapParseOneMessage parses a single IMAP message into an InboxMessage.
func imapParseOneMessage(msg *imap.Message, crRegex *regexp.Regexp) *InboxMessage {
	var rawBuf []byte
	for _, v := range msg.Body {
		rawBuf = make([]byte, v.Len())
		v.Read(rawBuf)
		break
	}

	cleaned := crRegex.ReplaceAllString(string(rawBuf), "")
	em, err := email.NewEmailFromReader(bytes.NewReader([]byte(cleaned)))
	if err != nil {
		log.Warnf("IMAP: failed to parse message seq=%d: %v", msg.SeqNum, err)
		return nil
	}

	headerStr := imapBuildHeaders(msg)
	senderEmail, senderName := imapExtractSender(msg)
	msgId := ""
	if msg.Envelope != nil {
		msgId = msg.Envelope.MessageId
	}

	bodyStr := string(em.Text)
	if bodyStr == "" {
		bodyStr = string(em.HTML)
	}

	return &InboxMessage{
		MessageId:    msgId,
		SenderEmail:  senderEmail,
		SenderName:   senderName,
		Subject:      msg.Envelope.Subject,
		Headers:      headerStr,
		Body:         bodyStr,
		ReceivedDate: msg.InternalDate,
		ProviderUID:  strconv.FormatUint(uint64(msg.SeqNum), 10),
	}
}

// imapBuildHeaders builds a headers string from an IMAP message envelope.
func imapBuildHeaders(msg *imap.Message) string {
	if msg.Envelope == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Subject: %s\n", msg.Envelope.Subject)
	fmt.Fprintf(&b, "Date: %s\n", msg.Envelope.Date.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: %s\n", msg.Envelope.MessageId)
	if len(msg.Envelope.From) > 0 {
		fmt.Fprintf(&b, "From: %s <%s>\n", msg.Envelope.From[0].PersonalName, msg.Envelope.From[0].Address())
	}
	if len(msg.Envelope.To) > 0 {
		fmt.Fprintf(&b, "To: %s <%s>\n", msg.Envelope.To[0].PersonalName, msg.Envelope.To[0].Address())
	}
	return b.String()
}

// imapExtractSender extracts the sender email and name from an IMAP envelope.
func imapExtractSender(msg *imap.Message) (string, string) {
	if msg.Envelope != nil && len(msg.Envelope.From) > 0 {
		return msg.Envelope.From[0].Address(), msg.Envelope.From[0].PersonalName
	}
	return "", ""
}

func (p *IMAPProvider) DeleteMessage(mailbox, providerUID string) error {
	imapClient, err := p.dial()
	if err != nil {
		return err
	}
	defer imapClient.Logout()

	if _, err := imapClient.Select("INBOX", false); err != nil {
		return fmt.Errorf("imap select INBOX: %w", err)
	}

	seqSet, err := parseSeqNum(providerUID)
	if err != nil {
		return err
	}

	// Mark as deleted
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	if err := imapClient.Store(seqSet, item, imap.DeletedFlag, nil); err != nil {
		return fmt.Errorf(errFmtStoreDeleted, err)
	}

	// Expunge
	if err := imapClient.Expunge(nil); err != nil {
		return fmt.Errorf(errFmtExpunge, err)
	}

	return nil
}

func (p *IMAPProvider) QuarantineMessage(mailbox, providerUID string) error {
	imapClient, err := p.dial()
	if err != nil {
		return err
	}
	defer imapClient.Logout()

	if _, err := imapClient.Select("INBOX", false); err != nil {
		return fmt.Errorf("imap select INBOX: %w", err)
	}

	seqSet, err := parseSeqNum(providerUID)
	if err != nil {
		return err
	}

	targetFolder := p.QuarantineFolder
	if targetFolder == "" {
		targetFolder = "Junk"
	}

	// Copy to quarantine folder
	if err := imapClient.Copy(seqSet, targetFolder); err != nil {
		return fmt.Errorf("imap copy to %s: %w", targetFolder, err)
	}

	// Delete from INBOX
	return imapDeleteAndExpunge(imapClient, seqSet)
}

func (p *IMAPProvider) RestoreMessage(mailbox, providerUID string) error {
	imapClient, err := p.dial()
	if err != nil {
		return err
	}
	defer imapClient.Logout()

	quarantineFolder := p.QuarantineFolder
	if quarantineFolder == "" {
		quarantineFolder = "Junk"
	}

	if _, err := imapClient.Select(quarantineFolder, false); err != nil {
		return fmt.Errorf("imap select %s: %w", quarantineFolder, err)
	}

	seqSet, err := parseSeqNum(providerUID)
	if err != nil {
		return err
	}

	// Copy back to INBOX
	if err := imapClient.Copy(seqSet, "INBOX"); err != nil {
		return fmt.Errorf("imap copy to INBOX: %w", err)
	}

	// Delete from quarantine
	return imapDeleteAndExpunge(imapClient, seqSet)
}

// parseSeqNum converts a provider UID string to an IMAP SeqSet.
func parseSeqNum(providerUID string) (*imap.SeqSet, error) {
	seqNum, err := strconv.ParseUint(providerUID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf(errFmtInvalidSeqNum, providerUID, err)
	}
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uint32(seqNum))
	return seqSet, nil
}

// imapDeleteAndExpunge marks messages in seqSet as deleted and expunges.
func imapDeleteAndExpunge(c *client.Client, seqSet *imap.SeqSet) error {
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	if err := c.Store(seqSet, item, imap.DeletedFlag, nil); err != nil {
		return fmt.Errorf(errFmtStoreDeleted, err)
	}
	if err := c.Expunge(nil); err != nil {
		return fmt.Errorf(errFmtExpunge, err)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────
// Microsoft Graph API Provider (Microsoft 365)
// ────────────────────────────────────────────────────────────────

// GraphProvider implements InboxProvider using the Microsoft Graph API.
type GraphProvider struct {
	TenantId     string
	ClientId     string
	ClientSecret string
	httpClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

func (p *GraphProvider) ProviderName() string { return "microsoft_graph" }

// graphTokenResponse holds the OAuth2 token response from Azure AD.
type graphTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// authenticate obtains a client_credentials access token from Azure AD.
func (p *GraphProvider) authenticate() error {
	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return nil // Token still valid
	}

	if p.httpClient == nil {
		p.httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", p.TenantId)
	data := fmt.Sprintf(
		"client_id=%s&client_secret=%s&scope=https%%3A%%2F%%2Fgraph.microsoft.com%%2F.default&grant_type=client_credentials",
		p.ClientId, p.ClientSecret,
	)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("graph auth: create request: %w", err)
	}
	req.Header.Set(headerContentType, "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("graph auth: send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("graph auth: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graph auth: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp graphTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("graph auth: unmarshal token: %w", err)
	}

	p.accessToken = tokenResp.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second) // refresh 60s early
	return nil
}

// graphDo sends an authenticated request to the Graph API.
func (p *GraphProvider) graphDo(method, url string, body io.Reader) ([]byte, int, error) {
	if err := p.authenticate(); err != nil {
		return nil, 0, err
	}

	if p.httpClient == nil {
		p.httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("graph request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set(headerContentType, "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("graph request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("graph read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// graphMessage represents a simplified Microsoft Graph message object.
type graphMessage struct {
	Id               string `json:"id"`
	Subject          string `json:"subject"`
	ReceivedDateTime string `json:"receivedDateTime"`
	IsRead           bool   `json:"isRead"`
	Body             struct {
		ContentType string `json:"contentType"`
		Content     string `json:"content"`
	} `json:"body"`
	From struct {
		EmailAddress struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"emailAddress"`
	} `json:"from"`
	InternetMessageHeaders []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"internetMessageHeaders"`
	InternetMessageId string `json:"internetMessageId"`
}

type graphMessagesResponse struct {
	Value    []graphMessage `json:"value"`
	NextLink string         `json:"@odata.nextLink"`
}

func (p *GraphProvider) FetchNewMessages(mailbox string, since time.Time) ([]InboxMessage, error) {
	if err := p.authenticate(); err != nil {
		return nil, err
	}

	sinceStr := since.UTC().Format("2006-01-02T15:04:05Z")
	url := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/messages?$filter=isRead eq false and receivedDateTime ge %s&$top=50&$select=id,subject,receivedDateTime,isRead,body,from,internetMessageHeaders,internetMessageId&$orderby=receivedDateTime desc",
		mailbox, sinceStr,
	)

	respBody, status, err := p.graphDo("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("graph fetch messages: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("graph fetch messages: status %d: %s", status, string(respBody))
	}

	var msgResp graphMessagesResponse
	if err := json.Unmarshal(respBody, &msgResp); err != nil {
		return nil, fmt.Errorf("graph parse messages: %w", err)
	}

	var results []InboxMessage
	for _, gm := range msgResp.Value {
		// Build headers string from internet message headers
		var headersBuf strings.Builder
		for _, h := range gm.InternetMessageHeaders {
			fmt.Fprintf(&headersBuf, "%s: %s\n", h.Name, h.Value)
		}

		receivedDate, _ := time.Parse(time.RFC3339, gm.ReceivedDateTime)

		msgId := gm.InternetMessageId
		if msgId == "" {
			msgId = gm.Id
		}

		results = append(results, InboxMessage{
			MessageId:    msgId,
			SenderEmail:  gm.From.EmailAddress.Address,
			SenderName:   gm.From.EmailAddress.Name,
			Subject:      gm.Subject,
			Headers:      headersBuf.String(),
			Body:         gm.Body.Content,
			ReceivedDate: receivedDate,
			ProviderUID:  gm.Id,
		})
	}

	return results, nil
}

func (p *GraphProvider) DeleteMessage(mailbox, providerUID string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/messages/%s", mailbox, providerUID)
	_, status, err := p.graphDo("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("graph delete message: %w", err)
	}
	if status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("graph delete message: unexpected status %d", status)
	}
	return nil
}

func (p *GraphProvider) QuarantineMessage(mailbox, providerUID string) error {
	// Move to "junkemail" well-known folder
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/messages/%s/move", mailbox, providerUID)
	payload := `{"destinationId":"junkemail"}`
	_, status, err := p.graphDo("POST", url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("graph quarantine message: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("graph quarantine message: unexpected status %d", status)
	}
	return nil
}

func (p *GraphProvider) RestoreMessage(mailbox, providerUID string) error {
	// Move back to inbox
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/messages/%s/move", mailbox, providerUID)
	payload := `{"destinationId":"inbox"}`
	_, status, err := p.graphDo("POST", url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("graph restore message: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("graph restore message: unexpected status %d", status)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────
// Google Gmail API Provider (Google Workspace)
// ────────────────────────────────────────────────────────────────

// GmailProvider implements InboxProvider using the Gmail API with
// domain-wide delegation (service account).
type GmailProvider struct {
	AdminEmail   string // Google Workspace admin email for domain-wide delegation
	AccessToken  string // Pre-obtained OAuth2 access token
	httpClient   *http.Client
}

func (p *GmailProvider) ProviderName() string { return "gmail" }

func (p *GmailProvider) getHTTPClient() *http.Client {
	if p.httpClient == nil {
		p.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return p.httpClient
}

// gmailDo sends an authenticated request to the Gmail API.
func (p *GmailProvider) gmailDo(method, url string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("gmail request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.AccessToken)
	req.Header.Set(headerContentType, "application/json")

	resp, err := p.getHTTPClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("gmail request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("gmail read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// gmailListResponse represents the response from Gmail users.messages.list.
type gmailListResponse struct {
	Messages []struct {
		Id       string `json:"id"`
		ThreadId string `json:"threadId"`
	} `json:"messages"`
	NextPageToken  string `json:"nextPageToken"`
	ResultSizeEstimate int `json:"resultSizeEstimate"`
}

// gmailMessageResponse represents a full Gmail message.
type gmailMessageResponse struct {
	Id        string `json:"id"`
	ThreadId  string `json:"threadId"`
	LabelIds  []string `json:"labelIds"`
	Snippet   string `json:"snippet"`
	Payload   gmailPayload `json:"payload"`
	InternalDate string `json:"internalDate"`
}

type gmailPayload struct {
	MimeType string        `json:"mimeType"`
	Headers  []gmailHeader `json:"headers"`
	Body     struct {
		Size int    `json:"size"`
		Data string `json:"data"`
	} `json:"body"`
	Parts []gmailPart `json:"parts"`
}

type gmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type gmailPart struct {
	MimeType string `json:"mimeType"`
	Body     struct {
		Size int    `json:"size"`
		Data string `json:"data"`
	} `json:"body"`
}

func (p *GmailProvider) FetchNewMessages(mailbox string, since time.Time) ([]InboxMessage, error) {
	sinceEpoch := since.Unix()
	query := fmt.Sprintf("is:unread after:%d", sinceEpoch)
	listURL := fmt.Sprintf(
		"https://gmail.googleapis.com/gmail/v1/users/%s/messages?q=%s&maxResults=50",
		mailbox, query,
	)

	listBody, status, err := p.gmailDo("GET", listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gmail list messages: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("gmail list messages: status %d: %s", status, string(listBody))
	}

	var listResp gmailListResponse
	if err := json.Unmarshal(listBody, &listResp); err != nil {
		return nil, fmt.Errorf("gmail parse list: %w", err)
	}

	var results []InboxMessage
	for _, m := range listResp.Messages {
		msg, err := p.fetchFullMessage(mailbox, m.Id)
		if err != nil {
			log.Warnf("Gmail: failed to fetch message %s: %v", m.Id, err)
			continue
		}
		results = append(results, *msg)
	}

	return results, nil
}

func (p *GmailProvider) fetchFullMessage(mailbox, messageId string) (*InboxMessage, error) {
	url := fmt.Sprintf(
		"https://gmail.googleapis.com/gmail/v1/users/%s/messages/%s?format=full",
		mailbox, messageId,
	)

	body, status, err := p.gmailDo("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("gmail get message: status %d", status)
	}

	var msg gmailMessageResponse
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("gmail parse message: %w", err)
	}

	// Extract headers
	var headersBuf strings.Builder
	var subject, senderEmail, senderName, msgIdHeader string
	for _, h := range msg.Payload.Headers {
		fmt.Fprintf(&headersBuf, "%s: %s\n", h.Name, h.Value)
		switch strings.ToLower(h.Name) {
		case "subject":
			subject = h.Value
		case "from":
			senderEmail, senderName = parseFromHeader(h.Value)
		case "message-id":
			msgIdHeader = h.Value
		}
	}

	// Extract body from parts
	bodyText := extractGmailBody(msg.Payload)

	// Parse internal date (epoch milliseconds)
	var receivedDate time.Time
	if ms, err := strconv.ParseInt(msg.InternalDate, 10, 64); err == nil {
		receivedDate = time.Unix(ms/1000, 0)
	}

	if msgIdHeader == "" {
		msgIdHeader = msg.Id
	}

	return &InboxMessage{
		MessageId:    msgIdHeader,
		SenderEmail:  senderEmail,
		SenderName:   senderName,
		Subject:      subject,
		Headers:      headersBuf.String(),
		Body:         bodyText,
		ReceivedDate: receivedDate,
		ProviderUID:  msg.Id,
	}, nil
}

func (p *GmailProvider) DeleteMessage(mailbox, providerUID string) error {
	// Use trash instead of permanent delete for safety
	url := fmt.Sprintf(
		"https://gmail.googleapis.com/gmail/v1/users/%s/messages/%s/trash",
		mailbox, providerUID,
	)
	_, status, err := p.gmailDo("POST", url, nil)
	if err != nil {
		return fmt.Errorf("gmail trash message: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("gmail trash message: unexpected status %d", status)
	}
	return nil
}

func (p *GmailProvider) QuarantineMessage(mailbox, providerUID string) error {
	// Add SPAM label and remove INBOX label
	url := fmt.Sprintf(
		"https://gmail.googleapis.com/gmail/v1/users/%s/messages/%s/modify",
		mailbox, providerUID,
	)
	payload := `{"addLabelIds":["SPAM"],"removeLabelIds":["INBOX"]}`
	_, status, err := p.gmailDo("POST", url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("gmail quarantine message: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("gmail quarantine message: unexpected status %d", status)
	}
	return nil
}

func (p *GmailProvider) RestoreMessage(mailbox, providerUID string) error {
	// Remove SPAM label and add INBOX label
	url := fmt.Sprintf(
		"https://gmail.googleapis.com/gmail/v1/users/%s/messages/%s/modify",
		mailbox, providerUID,
	)
	payload := `{"addLabelIds":["INBOX"],"removeLabelIds":["SPAM","TRASH"]}`
	_, status, err := p.gmailDo("POST", url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("gmail restore message: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("gmail restore message: unexpected status %d", status)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────

// parseFromHeader extracts email and name from a "From" header value like
// "John Doe <john@example.com>" or "john@example.com".
func parseFromHeader(from string) (emailAddr, name string) {
	from = strings.TrimSpace(from)
	if idx := strings.Index(from, "<"); idx >= 0 {
		name = strings.TrimSpace(from[:idx])
		emailAddr = strings.Trim(from[idx:], "<>")
	} else {
		emailAddr = from
	}
	// Remove surrounding quotes from name
	name = strings.Trim(name, `"'`)
	return
}

// extractGmailBody walks a Gmail message payload and returns the best
// text representation (preferring text/plain, falling back to text/html).
func extractGmailBody(payload gmailPayload) string {
	// Direct body
	if payload.Body.Data != "" && payload.Body.Size > 0 {
		decoded := decodeBase64URL(payload.Body.Data)
		return decoded
	}

	// Check parts for text/plain first, then text/html
	var htmlBody string
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" && part.Body.Data != "" {
			return decodeBase64URL(part.Body.Data)
		}
		if part.MimeType == "text/html" && part.Body.Data != "" {
			htmlBody = decodeBase64URL(part.Body.Data)
		}
	}
	return htmlBody
}

// decodeBase64URL decodes a base64url-encoded string (Gmail API encoding).
func decodeBase64URL(data string) string {
	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		// Try with padding stripped (base64.RawURLEncoding)
		decoded, err = base64.RawURLEncoding.DecodeString(data)
		if err != nil {
			return data // Return as-is if decode fails
		}
	}
	return string(decoded)
}
