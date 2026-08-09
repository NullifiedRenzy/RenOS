package main

import (
	"fmt"
	"os"
	"strings"
)

type QuestionType int

const (
	QuestionText QuestionType = iota
	QuestionSecret
	QuestionSelect
	QuestionMultiSelect
	QuestionConfirm
)

type Question struct {
	Key       string
	Type      QuestionType
	Label     string
	Default   string
	Options   []string
	Validate  func(string) error
	DependsOn func(*InstallConfig) bool
}

func GetQuestions() []Question {
	return []Question{
		{
			Key:   "hostname",
			Type:  QuestionText,
			Label: "Enter the hostname for RenOS",
			Default: func() string {
				host, _ := getHostname()
				return host
			}(),
			Validate: ValidateHostname,
		},
		{
			Key:      "tls_mode",
			Type:     QuestionSelect,
			Label:    "Select TLS mode",
			Default:  "self-signed",
			Options:  []string{"none", "self-signed", "letsencrypt", "custom"},
		},
		{
			Key:       "tls_cert_path",
			Type:      QuestionText,
			Label:     "Path to TLS certificate file",
			Default:   "/etc/renos/certs/cert.pem",
			Validate:  ValidateDirectory,
			DependsOn: func(c *InstallConfig) bool { return c.TLSMode == "custom" },
		},
		{
			Key:       "tls_key_path",
			Type:      QuestionText,
			Label:     "Path to TLS key file",
			Default:   "/etc/renos/certs/key.pem",
			Validate:  ValidateDirectory,
			DependsOn: func(c *InstallConfig) bool { return c.TLSMode == "custom" },
		},
		{
			Key:      "admin_name",
			Type:     QuestionText,
			Label:    "Enter the admin display name",
			Default:  "Administrator",
		},
		{
			Key:      "admin_email",
			Type:     QuestionText,
			Label:    "Enter the admin email address",
			Validate: ValidateEmail,
		},
		{
			Key:   "admin_password",
			Type:  QuestionSecret,
			Label: "Enter the admin password (min 8 characters)",
			Validate: func(v string) error {
				if len(v) < 8 {
					return fmt.Errorf("password must be at least 8 characters")
				}
				return nil
			},
		},
		{
			Key:      "data_dir",
			Type:     QuestionText,
			Label:    "Enter the data directory path",
			Default:  "/var/lib/renos/data",
			Validate: ValidateDirectory,
		},
		{
			Key:      "backup_dir",
			Type:     QuestionText,
			Label:    "Enter the backup directory path",
			Default:  "/var/lib/renos/backups",
			Validate: ValidateDirectory,
		},
		{
			Key:      "backup_retention",
			Type:     QuestionText,
			Label:    "Enter backup retention period in days",
			Default:  "30",
			Validate: ValidatePositiveInt,
		},
		{
			Key:      "ai_provider",
			Type:     QuestionSelect,
			Label:    "Select AI provider",
			Default:  "openai",
			Options:  []string{"openai", "anthropic", "ollama", "none"},
		},
		{
			Key:       "ai_model",
			Type:      QuestionText,
			Label:     "Enter the AI model name",
			Default:   "gpt-4",
			DependsOn: func(c *InstallConfig) bool { return c.AIProvider != "none" },
		},
		{
			Key:      "marketplace_enabled",
			Type:     QuestionConfirm,
			Label:    "Enable the RenOS Marketplace?",
			Default:  "yes",
		},
		{
			Key:       "marketplace_type",
			Type:      QuestionSelect,
			Label:     "Select marketplace type",
			Default:   "public",
			Options:   []string{"public", "private", "hybrid"},
			DependsOn: func(c *InstallConfig) bool { return c.MarketplaceEnabled },
		},
		{
			Key:   "plugins",
			Type:  QuestionMultiSelect,
			Label: "Select plugins to install",
			Options: []string{
				"monitoring", "logging", "backup",
				"sso", "webhook", "analytics",
				"cdn", "email", "notifications", "search",
			},
		},
		{
			Key:     "cluster_mode",
			Type:    QuestionSelect,
			Label:   "Select cluster mode",
			Default: "single",
			Options: []string{"single", "cluster"},
		},
		{
			Key:       "cluster_node_count",
			Type:      QuestionText,
			Label:     "Enter the number of cluster nodes",
			Default:   "3",
			Validate:  ValidatePositiveInt,
			DependsOn: func(c *InstallConfig) bool { return c.ClusterMode == "cluster" },
		},
		{
			Key:     "install_method",
			Type:    QuestionSelect,
			Label:   "Select installation method",
			Default: "docker",
			Options: []string{"docker", "kubernetes", "bare-metal"},
		},
	}
}

func getHostname() (string, error) {
	data, err := readFile("/etc/hostname")
	if err != nil {
		return "renos.local", nil
	}
	return strings.TrimSpace(data), nil
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
