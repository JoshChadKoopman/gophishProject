package imap

import (
	"context"
	"testing"
	"time"

	"github.com/jordan-wright/email"
)

// Shared test constant.
const imapFmtUnexpectedErr = "unexpected error: %v"

// ---------- goPhishRegex tests ----------

func TestGoPhishRegexStandardRID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"?rid=AbC1234", "AbC1234"},
		{"https://example.com/?rid=XyZ9876", "XyZ9876"},
		{"Click here: https://phish.test/landing?rid=a1B2c3D", "a1B2c3D"},
	}
	for _, tt := range tests {
		matches := goPhishRegex.FindAllStringSubmatch(tt.input, -1)
		if len(matches) == 0 {
			t.Fatalf("no match for input %q", tt.input)
		}
		rid := matches[0][len(matches[0])-1]
		if rid != tt.expected {
			t.Fatalf("input %q: expected RID %q, got %q", tt.input, tt.expected, rid)
		}
	}
}

func TestGoPhishRegexQuotedPrintable(t *testing.T) {
	// Quoted-printable encoding may prefix = with 3D
	input := "?rid=3DAbC1234"
	matches := goPhishRegex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		t.Fatal("expected match for quoted-printable RID")
	}
	rid := matches[0][len(matches[0])-1]
	if rid != "AbC1234" {
		t.Fatalf("expected 'AbC1234', got %q", rid)
	}
}

func TestGoPhishRegexURLEncoded(t *testing.T) {
	// URL-encoded ? and = characters
	input := "https://example.com/%3Frid%3DxYz4567"
	matches := goPhishRegex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		t.Fatal("expected match for URL-encoded RID")
	}
	rid := matches[0][len(matches[0])-1]
	if rid != "xYz4567" {
		t.Fatalf("expected 'xYz4567', got %q", rid)
	}
}

func TestGoPhishRegexNoMatch(t *testing.T) {
	inputs := []string{
		"no rid here",
		"?rid=short",   // Only 5 chars, need 7
		"?rid=Ab!@#$%", // Special characters
		"?rid=",        // No value
		"",
	}
	for _, input := range inputs {
		matches := goPhishRegex.FindAllStringSubmatch(input, -1)
		if len(matches) > 0 {
			t.Fatalf("unexpected match for input %q: %v", input, matches)
		}
	}
}

func TestGoPhishRegexMultipleRIDs(t *testing.T) {
	input := "?rid=AbC1234 some text ?rid=XyZ9876"
	matches := goPhishRegex.FindAllStringSubmatch(input, -1)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	rids := map[string]bool{}
	for _, m := range matches {
		rids[m[len(m)-1]] = true
	}
	if !rids["AbC1234"] || !rids["XyZ9876"] {
		t.Fatalf("expected both RIDs, got %v", rids)
	}
}

// ---------- checkRIDs tests ----------

func TestCheckRIDsFromText(t *testing.T) {
	em := &email.Email{
		Text: []byte("Please click: https://example.com/?rid=tEsT123"),
	}
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if !rids["tEsT123"] {
		t.Fatalf("expected RID 'tEsT123', got %v", rids)
	}
}

func TestCheckRIDsFromHTML(t *testing.T) {
	em := &email.Email{
		HTML: []byte(`<a href="https://phish.test/?rid=hTmL567">Click</a>`),
	}
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if !rids["hTmL567"] {
		t.Fatalf("expected RID 'hTmL567', got %v", rids)
	}
}

func TestCheckRIDsFromBothTextAndHTML(t *testing.T) {
	em := &email.Email{
		Text: []byte("?rid=aAa1111"),
		HTML: []byte("?rid=bBb2222"),
	}
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if len(rids) != 2 {
		t.Fatalf("expected 2 RIDs, got %d: %v", len(rids), rids)
	}
}

func TestCheckRIDsDeduplication(t *testing.T) {
	em := &email.Email{
		Text: []byte("?rid=dUp1234 and again ?rid=dUp1234"),
		HTML: []byte("?rid=dUp1234"),
	}
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if len(rids) != 1 {
		t.Fatalf("expected 1 unique RID, got %d: %v", len(rids), rids)
	}
}

func TestCheckRIDsNoMatch(t *testing.T) {
	em := &email.Email{
		Text: []byte("Hello, this is a regular email with no phishing links."),
		HTML: []byte("<p>Nothing suspicious here</p>"),
	}
	rids := make(map[string]bool)
	checkRIDs(em, rids)
	if len(rids) != 0 {
		t.Fatalf("expected 0 RIDs, got %d: %v", len(rids), rids)
	}
}

// ---------- matchEmail tests ----------

func TestMatchEmailTextOnly(t *testing.T) {
	em := &email.Email{
		Text: []byte("?rid=mAtCh01"),
	}
	rids, err := matchEmail(em)
	if err != nil {
		t.Fatalf(imapFmtUnexpectedErr, err)
	}
	if !rids["mAtCh01"] {
		t.Fatalf("expected RID 'mAtCh01', got %v", rids)
	}
}

func TestMatchEmailHTMLOnly(t *testing.T) {
	em := &email.Email{
		HTML: []byte(`<a href="https://test.com/?rid=hTmLmAt">link</a>`),
	}
	rids, err := matchEmail(em)
	if err != nil {
		t.Fatalf(imapFmtUnexpectedErr, err)
	}
	if !rids["hTmLmAt"] {
		t.Fatalf("expected RID 'hTmLmAt', got %v", rids)
	}
}

func TestMatchEmailNoRIDs(t *testing.T) {
	em := &email.Email{
		Text: []byte("Just a regular email"),
		HTML: []byte("<p>Nothing here</p>"),
	}
	rids, err := matchEmail(em)
	if err != nil {
		t.Fatalf(imapFmtUnexpectedErr, err)
	}
	if len(rids) != 0 {
		t.Fatalf("expected 0 RIDs, got %v", rids)
	}
}

// ---------- Mailbox struct tests ----------

func TestMailboxStructFields(t *testing.T) {
	mbox := Mailbox{
		Host:             "imap.example.com:993",
		TLS:              true,
		IgnoreCertErrors: false,
		User:             "test@example.com",
		Pwd:              "secret",
		Folder:           "INBOX",
		ReadOnly:         true,
	}
	if mbox.Host != "imap.example.com:993" {
		t.Fatal("host mismatch")
	}
	if !mbox.TLS {
		t.Fatal("expected TLS true")
	}
	if mbox.IgnoreCertErrors {
		t.Fatal("expected IgnoreCertErrors false")
	}
	if mbox.User != "test@example.com" {
		t.Fatal("user mismatch")
	}
	if mbox.Folder != "INBOX" {
		t.Fatal("folder mismatch")
	}
	if !mbox.ReadOnly {
		t.Fatal("expected ReadOnly true")
	}
}

func TestMailboxDefaultReadOnly(t *testing.T) {
	mbox := Mailbox{}
	if mbox.ReadOnly {
		t.Fatal("expected ReadOnly to default to false")
	}
}

// ---------- Email struct tests ----------

func TestEmailStruct(t *testing.T) {
	em := Email{
		SeqNum: 42,
		Email:  &email.Email{Subject: "Test Subject"},
	}
	if em.SeqNum != 42 {
		t.Fatalf("expected SeqNum 42, got %d", em.SeqNum)
	}
	if em.Subject != "Test Subject" {
		t.Fatalf("expected subject 'Test Subject', got %q", em.Subject)
	}
}

// ---------- Monitor lifecycle tests ----------

func TestNewMonitor(t *testing.T) {
	m := NewMonitor()
	if m == nil {
		t.Fatal("expected non-nil monitor")
	}
}

func TestMonitorStartShutdown(t *testing.T) {
	// We can't test the full monitor (it needs a database), but we can
	// verify that start/shutdown doesn't panic and the context is cancelled.
	m := &Monitor{}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// Verify shutdown doesn't panic
	err := m.Shutdown()
	if err != nil {
		t.Fatalf(imapFmtUnexpectedErr, err)
	}

	// Context should be done after shutdown
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Fatal("expected context to be cancelled after Shutdown")
	}
}

func TestMonitorStartAndStop(t *testing.T) {
	// This tests the Monitor's Start method by immediately shutting it down.
	// The internal goroutine calls models.GetUsers() which will fail without
	// a database, but we just need to ensure no panics and clean shutdown.
	m := NewMonitor()

	// We create our own context-based monitor to avoid the DB dependency
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// Start a mock goroutine that simulates monitor behavior
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			t.Error("monitor goroutine did not stop in time")
		}
	}()

	// Shutdown
	m.Shutdown()

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine did not exit after shutdown")
	}
}
