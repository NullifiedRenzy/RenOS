package notify

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
)

type emailSender func(context.Context, smtpConfig, []string, string, string) error

type smtpConfig struct {
	Host               string   `json:"host"`
	Port               int      `json:"port"`
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	From               string   `json:"from"`
	FromName           string   `json:"from_name"`
	To                 []string `json:"to"`
	Recipients         []string `json:"recipients"`
	TLS                bool     `json:"tls"`
	StartTLS           *bool    `json:"start_tls"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify"`
}

func parseSMTPConfig(raw json.RawMessage) (smtpConfig, error) {
	var cfg smtpConfig
	if len(raw) == 0 {
		return cfg, errors.New("SMTP provider config is empty")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("parse SMTP provider config: %w", err)
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.From = strings.TrimSpace(cfg.From)
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	if cfg.Host == "" {
		return cfg, errors.New("SMTP host is required")
	}
	if cfg.From == "" {
		return cfg, errors.New("SMTP from address is required")
	}
	return cfg, nil
}

func (c smtpConfig) recipients() []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [][]string{c.To, c.Recipients} {
		for _, raw := range list {
			addr := strings.TrimSpace(raw)
			if addr == "" || seen[strings.ToLower(addr)] {
				continue
			}
			seen[strings.ToLower(addr)] = true
			out = append(out, addr)
		}
	}
	return out
}

func (c smtpConfig) useStartTLS() bool {
	if c.TLS {
		return false
	}
	if c.StartTLS != nil {
		return *c.StartTLS
	}
	return c.Port == 587
}

func sendSMTPEmail(ctx context.Context, cfg smtpConfig, recipients []string, subject, body string) error {
	if len(recipients) == 0 {
		return errors.New("no SMTP recipients configured")
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	client, err := dialSMTP(ctx, addr, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	if cfg.useStartTLS() {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: cfg.Host, InsecureSkipVerify: cfg.InsecureSkipVerify}); err != nil {
				return fmt.Errorf("start SMTP TLS: %w", err)
			}
		} else {
			return errors.New("SMTP server does not advertise STARTTLS")
		}
	}
	if cfg.Username != "" || cfg.Password != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
			return fmt.Errorf("authenticate SMTP: %w", err)
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	for _, to := range recipients {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("set SMTP recipient %s: %w", to, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP data: %w", err)
	}
	if _, err := io.WriteString(w, buildEmailMessage(cfg, recipients, subject, body)); err != nil {
		w.Close()
		return fmt.Errorf("write SMTP data: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close SMTP data: %w", err)
	}
	return client.Quit()
}

func dialSMTP(ctx context.Context, addr string, cfg smtpConfig) (*smtp.Client, error) {
	dialer := &net.Dialer{}
	if cfg.TLS || cfg.Port == 465 {
		conn, err := (&tls.Dialer{
			NetDialer: dialer,
			Config:    &tls.Config{ServerName: cfg.Host, InsecureSkipVerify: cfg.InsecureSkipVerify},
		}).DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("dial SMTP TLS: %w", err)
		}
		return smtp.NewClient(conn, cfg.Host)
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial SMTP: %w", err)
	}
	return smtp.NewClient(conn, cfg.Host)
}

func buildEmailMessage(cfg smtpConfig, recipients []string, subject, body string) string {
	headers := []string{
		"From: " + formatFrom(cfg),
		"To: " + strings.Join(recipients, ", "),
		"Subject: " + sanitizeHeader(subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body + "\r\n"
}

func formatFrom(cfg smtpConfig) string {
	if strings.TrimSpace(cfg.FromName) == "" {
		return cfg.From
	}
	return (&mail.Address{Name: cfg.FromName, Address: cfg.From}).String()
}

func sanitizeHeader(v string) string {
	return strings.ReplaceAll(strings.ReplaceAll(v, "\r", " "), "\n", " ")
}
