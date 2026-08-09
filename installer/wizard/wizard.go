package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func RunWizard(cfg *InstallConfig, outputPath string, dryRun bool) (*InstallConfig, error) {
	if cfg == nil {
		cfg = &InstallConfig{}
	}

	PrintBanner()

	questions := GetQuestions()
	totalSteps := countActiveSteps(questions, cfg) + 1 // +1 for Welcome
	currentStep := 1

	PrintProgress(currentStep, totalSteps)
	PrintSection("Welcome")
	welcomeStep()
	currentStep++

	for _, q := range questions {
		if q.DependsOn != nil && !q.DependsOn(cfg) {
			continue
		}

		PrintProgress(currentStep, totalSteps)

		stepTitle := questionStepTitle(q.Key)
		PrintSection(stepTitle)

		switch q.Type {
		case QuestionText:
			val := Prompt(q.Label, q.Default, q.Validate)
			setConfigValue(cfg, q.Key, val)

		case QuestionSecret:
			val := PromptSecret(q.Label)
			setConfigValue(cfg, q.Key, val)

		case QuestionSelect:
			val := PromptSelect(q.Label, q.Options, q.Default)
			setConfigValue(cfg, q.Key, val)

		case QuestionMultiSelect:
			selected := PromptMultiSelect(q.Label, q.Options)
			setConfigSlice(cfg, q.Key, selected)

		case QuestionConfirm:
			defaultYes := strings.ToLower(q.Default) == "yes" || q.Default == "true"
			val := Confirm(q.Label, defaultYes)
			setConfigBool(cfg, q.Key, val)
		}

		currentStep++
	}

	PrintProgress(totalSteps, totalSteps)
	PrintSection("Configuration Review")

	reviewConfig(cfg)

	if !Confirm("\nProceed with this configuration?", true) {
		fmt.Println(colorize(ColorYellow, "\nWizard cancelled by user."))
		return cfg, fmt.Errorf("cancelled")
	}

	PrintSection("Generating Configuration")

	if dryRun {
		fmt.Println(colorize(ColorYellow, "  DRY-RUN MODE - No files will be written"))
		fmt.Println()
		previewOutputs(cfg)
		return cfg, nil
	}

	if outputPath != "" {
		if err := cfg.SaveToFile(outputPath); err != nil {
			return nil, fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("  %s Configuration saved to %s\n", colorize(ColorGreen, "✓"), outputPath)
	}

	generateAll(cfg, outputPath)

	fmt.Println()
	fmt.Println(colorize(ColorGreen, "  ✓ Configuration generated successfully!"))
	fmt.Println()
	fmt.Println(colorize(ColorCyan, "  Next steps:"))
	fmt.Println("   1. Review the generated configuration files")
	fmt.Println("   2. Run the installation with your preferred method:")
	fmt.Printf("      - Docker:     docker-compose up -d\n")
	fmt.Printf("      - Kubernetes: helm install renos -f values.yaml renos/renos\n")
	fmt.Printf("      - Bare Metal: ./renos serve --config %s\n", outputPath)
	fmt.Println()

	return cfg, nil
}

func welcomeStep() {
	fmt.Println("  This wizard will guide you through configuring RenOS,")
	fmt.Println("  the modern AI platform for deploying and managing")
	fmt.Println("  AI-powered applications at scale.")
	fmt.Println()
	fmt.Println("  You will be asked about:")
	fmt.Println("    • Hostname and TLS configuration")
	fmt.Println("    • Admin user setup")
	fmt.Println("    • Storage and backup preferences")
	fmt.Println("    • AI provider integration")
	fmt.Println("    • Marketplace and plugin selection")
	fmt.Println("    • Cluster configuration")
	fmt.Println("    • Installation method")
	fmt.Println()
	Confirm("Ready to begin?", true)
}

func questionStepTitle(key string) string {
	titles := map[string]string{
		"hostname":             "Hostname Configuration",
		"tls_mode":             "TLS Mode",
		"tls_cert_path":        "TLS Certificate Path",
		"tls_key_path":         "TLS Key Path",
		"admin_name":           "Admin User - Name",
		"admin_email":          "Admin User - Email",
		"admin_password":       "Admin User - Password",
		"data_dir":             "Storage - Data Directory",
		"backup_dir":           "Storage - Backup Directory",
		"backup_retention":     "Storage - Backup Retention",
		"ai_provider":          "AI Provider",
		"ai_model":             "AI Model",
		"marketplace_enabled":  "Marketplace",
		"marketplace_type":     "Marketplace Type",
		"plugins":              "Plugins",
		"cluster_mode":         "Cluster Configuration",
		"cluster_node_count":   "Cluster Node Count",
		"install_method":       "Installation Method",
	}
	if title, ok := titles[key]; ok {
		return title
	}
	return strings.ReplaceAll(key, "_", " ")
}

func setConfigValue(cfg *InstallConfig, key, value string) {
	switch key {
	case "hostname":
		cfg.Hostname = value
	case "tls_mode":
		cfg.TLSMode = value
	case "tls_cert_path":
		cfg.TLSCertPath = value
	case "tls_key_path":
		cfg.TLSKeyPath = value
	case "admin_name":
		cfg.AdminName = value
	case "admin_email":
		cfg.AdminEmail = value
	case "admin_password":
		cfg.AdminPassword = value
	case "data_dir":
		cfg.DataDir = value
	case "backup_dir":
		cfg.BackupDir = value
	case "backup_retention":
		if n, err := strconv.Atoi(value); err == nil {
			cfg.BackupRetention = n
		}
	case "ai_provider":
		cfg.AIProvider = value
	case "ai_model":
		cfg.AIModel = value
	case "marketplace_type":
		cfg.MarketplaceType = value
	case "cluster_mode":
		cfg.ClusterMode = value
	case "cluster_node_count":
		if n, err := strconv.Atoi(value); err == nil {
			cfg.ClusterNodeCount = n
		}
	case "install_method":
		cfg.InstallMethod = value
	}
}

func setConfigSlice(cfg *InstallConfig, key string, values []string) {
	switch key {
	case "plugins":
		cfg.Plugins = values
	}
}

func setConfigBool(cfg *InstallConfig, key string, value bool) {
	switch key {
	case "marketplace_enabled":
		cfg.MarketplaceEnabled = value
	}
}

func reviewConfig(cfg *InstallConfig) {
	headers := []string{"Setting", "Value"}
	rows := cfg.SummaryRows()
	PrintTable(headers, rows)
}

func previewOutputs(cfg *InstallConfig) {
	fmt.Println(colorize(ColorYellow, "  === Environment File Preview ==="))
	fmt.Println(cfg.GenerateEnvFile())
	fmt.Println()
	fmt.Println(colorize(ColorYellow, "  === Docker Compose Preview ==="))
	fmt.Println(cfg.GenerateDockerCompose())
	fmt.Println()
	fmt.Println(colorize(ColorYellow, "  === Kubernetes Values Preview ==="))
	fmt.Println(cfg.GenerateK8sValues())
}

func countActiveSteps(questions []Question, cfg *InstallConfig) int {
	depCfg := *cfg
	if depCfg.TLSMode == "" {
		depCfg.TLSMode = "self-signed"
	}
	if depCfg.AIProvider == "" {
		depCfg.AIProvider = "openai"
	}
	if !depCfg.MarketplaceEnabled {
		depCfg.MarketplaceEnabled = true
	}
	if depCfg.ClusterMode == "" {
		depCfg.ClusterMode = "single"
	}

	count := 0
	for _, q := range questions {
		if q.DependsOn == nil || q.DependsOn(&depCfg) {
			count++
		}
	}
	return count
}

func generateAll(cfg *InstallConfig, outputPath string) {
	baseDir := strings.TrimSuffix(outputPath, ".json")
	if baseDir == outputPath {
		baseDir = outputPath + "_output"
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Printf("  %s Warning: could not create output directory: %v\n", colorize(ColorYellow, "!"), err)
	}

	envPath := baseDir + "/renos.env"
	envContent := cfg.GenerateEnvFile()
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		fmt.Printf("  %s Failed to write %s: %v\n", colorize(ColorRed, "✗"), envPath, err)
	} else {
		fmt.Printf("  %s Environment file: %s\n", colorize(ColorGreen, "✓"), envPath)
	}

	composePath := baseDir + "/docker-compose.yml"
	composeContent := cfg.GenerateDockerCompose()
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		fmt.Printf("  %s Failed to write %s: %v\n", colorize(ColorRed, "✗"), composePath, err)
	} else {
		fmt.Printf("  %s Docker Compose file: %s\n", colorize(ColorGreen, "✓"), composePath)
	}

	k8sPath := baseDir + "/values.yaml"
	k8sContent := cfg.GenerateK8sValues()
	if err := os.WriteFile(k8sPath, []byte(k8sContent), 0644); err != nil {
		fmt.Printf("  %s Failed to write %s: %v\n", colorize(ColorRed, "✗"), k8sPath, err)
	} else {
		fmt.Printf("  %s Kubernetes values file: %s\n", colorize(ColorGreen, "✓"), k8sPath)
	}
}
