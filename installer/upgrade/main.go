package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const version = "1.0.0"

type Command struct {
	Name        string
	Description string
	Run         func(ctx context.Context, args []string) error
}

var commands = map[string]Command{
	"upgrade": {
		Name: "upgrade", Description: "Perform a full upgrade of a component",
		Run: runUpgrade,
	},
	"dry-run": {
		Name: "dry-run", Description: "Validate upgrade without making changes",
		Run: runDryRun,
	},
	"rollback": {
		Name: "rollback", Description: "Rollback a previous upgrade by ID",
		Run: runRollback,
	},
	"status": {
		Name: "status", Description: "Show current upgrade state and component versions",
		Run: runStatus,
	},
	"history": {
		Name: "history", Description: "Show upgrade history",
		Run: runHistory,
	},
	"check": {
		Name: "check", Description: "Run comprehensive health checks on all components",
		Run: runCheck,
	},
}

type GlobalFlags struct {
	Component string
	Version   string
	Force     bool
	Yes       bool
}

func parseGlobalFlags() GlobalFlags {
	component := flag.String("component", "all", "Component to target (core|hub|agent|all)")
	targetVersion := flag.String("version", "", "Target version for upgrade")
	force := flag.Bool("force", false, "Force upgrade even if checks warn")
	yes := flag.Bool("yes", false, "Auto-confirm prompts")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "RenOS Upgrade Manager v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: renos-upgrade <command> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		for _, cmd := range commands {
			fmt.Fprintf(os.Stderr, "  %-12s %s\n", cmd.Name, cmd.Description)
		}
		fmt.Fprintf(os.Stderr, "\nGlobal Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  renos-upgrade upgrade --component core --version 1.2.0\n")
		fmt.Fprintf(os.Stderr, "  renos-upgrade dry-run --component hub --version 2.0.0\n")
		fmt.Fprintf(os.Stderr, "  renos-upgrade rollback upg-1234567890\n")
		fmt.Fprintf(os.Stderr, "  renos-upgrade status\n")
		fmt.Fprintf(os.Stderr, "  renos-upgrade check\n")
	}

	flag.Parse()

	return GlobalFlags{
		Component: *component,
		Version:   *targetVersion,
		Force:     *force,
		Yes:       *yes,
	}
}

func validComponent(c string) bool {
	switch c {
	case "core", "hub", "agent", "all":
		return true
	}
	return false
}

func main() {
	flags := parseGlobalFlags()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	cmdName := flag.Arg(0)
	cmd, ok := commands[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmdName)
		flag.Usage()
		os.Exit(1)
	}

	os.Args = append([]string{os.Args[0]}, flag.Args()[1:]...)

	if !validComponent(flags.Component) {
		fmt.Fprintf(os.Stderr, "invalid component: %s (use core|hub|agent|all)\n", flags.Component)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\ninterrupt received, shutting down...")
		cancel()
	}()

	if err := cmd.Run(ctx, []string{flags.Component, flags.Version}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runUpgrade(ctx context.Context, args []string) error {
	component := args[0]
	targetVersion := args[1]

	if targetVersion == "" {
		return fmt.Errorf("--version is required for upgrade command")
	}

	mgr := NewUpgradeManager()
	mgr.Verbose = true

	go func() {
		for evt := range mgr.Events {
			printEvent(evt)
		}
	}()

	return mgr.Upgrade(ctx, component, targetVersion)
}

func runDryRun(ctx context.Context, args []string) error {
	component := args[0]
	targetVersion := args[1]

	if targetVersion == "" {
		return fmt.Errorf("--version is required for dry-run command")
	}

	mgr := NewUpgradeManager()
	return mgr.DryRun(ctx, component, targetVersion)
}

func runRollback(ctx context.Context, args []string) error {
	upgradeID := ""
	if len(args) > 0 {
		upgradeID = args[0]
	}

	if upgradeID == "" {
		rollbacks := ListAvailableRollbacks()
		if len(rollbacks) == 0 {
			fmt.Println("No rollbacks available.")
			return nil
		}
		fmt.Println("Available rollbacks:")
		for _, r := range rollbacks {
			fmt.Printf("  %s: %s %s -> %s (%s)\n",
				r.ID, r.Component, r.FromVersion, r.ToVersion,
				r.Timestamp.Format("2006-01-02 15:04:05"))
		}
		return nil
	}

	result := Rollback(ctx, upgradeID)
	if result.Status != "success" {
		return fmt.Errorf("rollback failed: %s", result.Message)
	}
	fmt.Printf("Rollback successful: %s\n", result.Message)
	return nil
}

func runStatus(ctx context.Context, args []string) error {
	state, err := Load()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	fmt.Printf("RenOS Upgrade Status\n")
	fmt.Printf("  Current Version:    %s\n", state.CurrentVersion)
	fmt.Printf("  Last Backup ID:     %s\n", state.LastBackupID)
	fmt.Printf("  Total Upgrades:     %d\n", len(state.History))
	fmt.Println()
	fmt.Println("Component Versions:")
	for comp, ver := range state.ComponentVersion {
		fmt.Printf("  %-12s %s\n", comp+":", ver)
	}
	fmt.Println()
	fmt.Println("Recent History (last 5):")
	history := state.ListHistory()
	start := 0
	if len(history) > 5 {
		start = len(history) - 5
	}
	for _, r := range history[start:] {
		fmt.Printf("  %s: %s %s -> %s [%s]\n",
			r.ID[:min(len(r.ID), 20)], r.Component, r.FromVersion, r.ToVersion, r.Status)
	}
	return nil
}

func runHistory(ctx context.Context, args []string) error {
	state, err := Load()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	history := state.ListHistory()
	if len(history) == 0 {
		fmt.Println("No upgrade history available.")
		return nil
	}

	fmt.Printf("%-30s %-12s %-16s %-16s %-10s %s\n",
		"ID", "Component", "From", "To", "Status", "Timestamp")
	fmt.Println(strings.Repeat("-", 120))
	for _, r := range history {
		fmt.Printf("%-30s %-12s %-16s %-16s %-10s %s\n",
			r.ID, r.Component, r.FromVersion, r.ToVersion, r.Status,
			r.Timestamp.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func runCheck(ctx context.Context, args []string) error {
	fmt.Println("RenOS Health Check")
	fmt.Println(strings.Repeat("=", 80))

	health := AllHealthChecks()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	for _, status := range health.Statuses {
		icon := "OK"
		if !status.Healthy {
			icon = "FAIL"
		}
		fmt.Printf("[%s] %s/%s: %s\n", icon, status.Component, status.Service, status.Message)
		_ = enc.Encode(status)
	}

	fmt.Println(strings.Repeat("-", 80))
	if health.AllOK {
		fmt.Println("Result: ALL SYSTEMS HEALTHY")
	} else {
		fmt.Println("Result: SOME CHECKS FAILED")
		return fmt.Errorf("health checks failed")
	}
	return nil
}

func printEvent(evt UpgradeEvent) {
	icon := ""
	switch evt.Status {
	case "start":
		icon = ">>>"
	case "done":
		icon = "OK "
	case "error":
		icon = "ERR"
	case "warn":
		icon = "WRN"
	case "info":
		icon = "INF"
	default:
		icon = "---"
	}
	fmt.Fprintf(os.Stderr, "  %s [%s/%s] %s\n", icon, evt.Component, evt.Step, evt.Message)
}
