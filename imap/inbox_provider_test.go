package imap

import (
	"testing"
	"time"
)

// ─── InboxMessage tests ───

func TestInboxMessageFieldsInitialized(t *testing.T) {
	msg := InboxMessage{
		MessageId:    "test-id-123",
		SenderEmail:  "sender@example.com",
		SenderName:   "Test Sender",
		Subject:      "Test Subject",
		Headers:      "From: sender@example.com",
		Body:         "Hello world",
		ReceivedDate: time.Now(),
		ProviderUID:  "42",
	}
	if msg.MessageId != "test-id-123" {
		t.Fatal("message id mismatch")
	}
	if msg.SenderEmail != "sender@example.com" {
		t.Fatal("sender email mismatch")
	}
	if msg.ProviderUID != "42" {
		t.Fatal("provider uid mismatch")
	}
}

// ─── IMAP Provider tests (unit, no server) ───

func TestIMAPProviderName(t *testing.T) {
	p := &IMAPProvider{}
	if p.ProviderName() != "imap" {
		t.Fatalf("expected 'imap', got %q", p.ProviderName())
	}
}

func TestIMAPProviderQuarantineFolderDefault(t *testing.T) {
	p := &IMAPProvider{}
	if p.QuarantineFolder != "" {
		t.Fatal("quarantine folder should be empty by default (resolved to 'Junk' at runtime)")
	}
}

// ─── Graph Provider tests (unit, no server) ───

func TestGraphProviderName(t *testing.T) {
	p := &GraphProvider{}
	if p.ProviderName() != "microsoft_graph" {
		t.Fatalf("expected 'microsoft_graph', got %q", p.ProviderName())
	}
}

func TestGraphProviderAuthRequiresTenant(t *testing.T) {
	p := &GraphProvider{
		TenantId:     "",
		ClientId:     "test",
		ClientSecret: "test",
	}
	// With empty tenant, auth URL will be malformed and fail
	err := p.authenticate()
	if err == nil {
		t.Fatal("expected auth error with empty tenant")
	}
}

// ─── Gmail Provider tests (unit, no server) ───

func TestGmailProviderName(t *testing.T) {
	p := &GmailProvider{}
	if p.ProviderName() != "gmail" {
		t.Fatalf("expected 'gmail', got %q", p.ProviderName())
	}
}

func TestGmailProviderHTTPClientInit(t *testing.T) {
	p := &GmailProvider{}
	c := p.getHTTPClient()
	if c == nil {
		t.Fatal("expected non-nil http client")
	}
	// Calling again should return the same instance
	c2 := p.getHTTPClient()
	if c != c2 {
		t.Fatal("expected same http client instance")
	}
}

// ─── parseFromHeader tests ───

func TestParseFromHeaderFull(t *testing.T) {
	email, name := parseFromHeader("John Doe <john@example.com>")
	if email != "john@example.com" {
		t.Fatalf("expected john@example.com, got %q", email)
	}
	if name != "John Doe" {
		t.Fatalf("expected 'John Doe', got %q", name)
	}
}

func TestParseFromHeaderQuoted(t *testing.T) {
	email, name := parseFromHeader(`"Jane Smith" <jane@example.com>`)
	if email != "jane@example.com" {
		t.Fatalf("expected jane@example.com, got %q", email)
	}
	if name != "Jane Smith" {
		t.Fatalf("expected 'Jane Smith', got %q", name)
	}
}

func TestParseFromHeaderEmailOnly(t *testing.T) {
	email, name := parseFromHeader("user@example.com")
	if email != "user@example.com" {
		t.Fatalf("expected user@example.com, got %q", email)
	}
	if name != "" {
		t.Fatalf("expected empty name, got %q", name)
	}
}

// ─── decodeBase64URL tests ───

func TestDecodeBase64URLValid(t *testing.T) {
	// "Hello World" in base64url
	encoded := "SGVsbG8gV29ybGQ"
	decoded := decodeBase64URL(encoded)
	if decoded != "Hello World" {
		t.Fatalf("expected 'Hello World', got %q", decoded)
	}
}

func TestDecodeBase64URLInvalid(t *testing.T) {
	// Invalid base64 should return as-is
	result := decodeBase64URL("!!!not-base64!!!")
	if result != "!!!not-base64!!!" {
		t.Fatalf("expected original string, got %q", result)
	}
}

// ─── imapFolderFromMailbox tests ───

func TestImapFolderFromMailboxDefault(t *testing.T) {
	f := imapFolderFromMailbox("user@example.com")
	if f != "INBOX" {
		t.Fatalf("expected INBOX, got %q", f)
	}
}

func TestImapFolderFromMailboxEmpty(t *testing.T) {
	f := imapFolderFromMailbox("")
	if f != "INBOX" {
		t.Fatalf("expected INBOX, got %q", f)
	}
}

func TestImapFolderFromMailboxWithFolder(t *testing.T) {
	f := imapFolderFromMailbox("user@example.com/Sent Items")
	if f != "Sent Items" {
		t.Fatalf("expected 'Sent Items', got %q", f)
	}
}

// ─── parseSeqNum tests ───

func TestParseSeqNumValid(t *testing.T) {
	seqSet, err := parseSeqNum("42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seqSet == nil {
		t.Fatal("expected non-nil SeqSet")
	}
}

func TestParseSeqNumInvalid(t *testing.T) {
	_, err := parseSeqNum("notanumber")
	if err == nil {
		t.Fatal("expected error for invalid sequence number")
	}
}

func TestParseSeqNumEmpty(t *testing.T) {
	_, err := parseSeqNum("")
	if err == nil {
		t.Fatal("expected error for empty sequence number")
	}
}

// ─── extractGmailBody tests ───

func TestExtractGmailBodyFromDirectBody(t *testing.T) {
	payload := gmailPayload{
		Body: struct {
			Size int    `json:"size"`
			Data string `json:"data"`
		}{
			Size: 11,
			Data: "SGVsbG8gV29ybGQ", // "Hello World"
		},
	}
	body := extractGmailBody(payload)
	if body != "Hello World" {
		t.Fatalf("expected 'Hello World', got %q", body)
	}
}

func TestExtractGmailBodyFromTextPart(t *testing.T) {
	payload := gmailPayload{
		Parts: []gmailPart{
			{
				MimeType: "text/plain",
				Body: struct {
					Size int    `json:"size"`
					Data string `json:"data"`
				}{
					Size: 11,
					Data: "SGVsbG8gV29ybGQ",
				},
			},
		},
	}
	body := extractGmailBody(payload)
	if body != "Hello World" {
		t.Fatalf("expected 'Hello World', got %q", body)
	}
}

func TestExtractGmailBodyEmpty(t *testing.T) {
	payload := gmailPayload{}
	body := extractGmailBody(payload)
	if body != "" {
		t.Fatalf("expected empty, got %q", body)
	}
}
