package main

import (
	"fmt"
	"os"
	"strings"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "create":
		if len(os.Args) < 3 {
			fmt.Println("Usage: renos-plugin create <name>")
			os.Exit(1)
		}
		name := os.Args[2]
		if err := ScaffoldPlugin(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "build":
		if err := CmdBuild(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "test":
		if err := CmdTest(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "package":
		if err := CmdPackage(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "publish":
		if err := CmdPublish(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "install":
		if len(os.Args) < 3 {
			fmt.Println("Usage: renos-plugin install <path>")
			os.Exit(1)
		}
		path := os.Args[2]
		if err := CmdInstall(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "validate":
		if err := CmdValidate(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "generate":
		if len(os.Args) < 3 {
			fmt.Println("Usage: renos-plugin generate <type>")
			fmt.Println()
			listTemplates()
			os.Exit(1)
		}
		tmplType := os.Args[2]
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := GenerateTemplate(tmplType, dir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated %q template\n", tmplType)

	case "version":
		fmt.Printf("renos-plugin version %s\n", version)

	case "help", "--help", "-h":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		fmt.Println()
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`RenOS Plugin SDK CLI %s

Usage:
  renos-plugin create <name>    Generate a new plugin scaffold
  renos-plugin build            Build the plugin
  renos-plugin test             Run plugin tests
  renos-plugin package          Package plugin for distribution
  renos-plugin publish          Publish to marketplace
  renos-plugin install <path>   Install a plugin (.rpk or directory)
  renos-plugin validate         Validate plugin structure
  renos-plugin generate <type>  Generate a starter template
  renos-plugin version          Print version
  renos-plugin help             Show this help

Templates:
  dashboard-page   A TypeScript React dashboard page component
  api-route        A Go HTTP handler with auth middleware
  worker           A background worker with NATS subscription
  schedule         A cron-based scheduled task
  sidebar-item     A navigation item for the RenOS hub UI
  webhook          A webhook receiver with signature verification
  workflow-action  A custom workflow step action

Examples:
  renos-plugin create my-plugin
  cd my-plugin && renos-plugin build
  cd my-plugin && renos-plugin generate dashboard-page
`, version)

	_ = strings.Join // avoid unused import
}
