package notify

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseSMTPConfigDefaultsPort(t *testing.T) {
	cfg, err := parseSMTPConfig(json.RawMessage(`{"host":"smtp.example.test","from":"renos@example.test"}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 587 {
		t.Fatalf("Port = %d, want 587", cfg.Port)
	}
	if !cfg.useStartTLS() {
		t.Fatal("expected default STARTTLS on port 587")
	}
}

func TestSMTPRecipientsDeduplicate(t *testing.T) {
	cfg := smtpConfig{To: []string{"a@example.test", "A@example.test", ""}, Recipients: []string{"b@example.test"}}
	got := cfg.recipients()
	if len(got) != 2 || got[0] != "a@example.test" || got[1] != "b@example.test" {
		t.Fatalf("recipients = %#v", got)
	}
}

func TestBuildEmailMessageSanitizesSubject(t *testing.T) {
	msg := buildEmailMessage(smtpConfig{From: "renos@example.test"}, []string{"admin@example.test"}, "hello\r\nBcc: bad", "body")
	if strings.Contains(msg, "\r\nBcc:") {
		t.Fatalf("message contains injected header: %q", msg)
	}
	if !strings.Contains(msg, "\r\n\r\nbody\r\n") {
		t.Fatalf("message missing body separator: %q", msg)
	}
}
