package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	configPath := flag.String("config", "", "Path to existing JSON configuration file to pre-load")
	outputPath := flag.String("output", "renos-config.json", "Path to save the generated configuration")
	dryRun := flag.Bool("dry-run", false, "Preview configuration without writing files")
	help := flag.Bool("help", false, "Show this help message")

	flag.Usage = printUsage
	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	var cfg *InstallConfig
	if *configPath != "" {
		var err error
		cfg, err = LoadFromFile(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s Error loading config from %s: %v\n",
				colorize(ColorRed, "!"), *configPath, err)
			os.Exit(1)
		}
		fmt.Printf("%s Loaded existing configuration from %s\n",
			colorize(ColorGreen, "✓"), *configPath)
		fmt.Println()
	}

	result, err := RunWizard(cfg, *outputPath, *dryRun)
	if err != nil {
		if err.Error() == "cancelled" {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "%s Wizard failed: %v\n", colorize(ColorRed, "✗"), err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println()
		fmt.Println(colorize(ColorYellow, "  Dry-run complete. No files were written."))
		fmt.Println(colorize(ColorYellow, "  Run without --dry-run to generate configuration files."))
		os.Exit(0)
	}

	if errs := result.Validate(); len(errs) > 0 {
		fmt.Printf("%s Configuration has validation warnings:\n", colorize(ColorYellow, "!"))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		fmt.Println()
	}
}

func printUsage() {
	name := os.Args[0]
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}

	fmt.Println(colorize(ColorCyan, "RenOS Interactive Installation Wizard"))
	fmt.Println(colorize(ColorCyan, "====================================="))
	fmt.Println()
	fmt.Printf("Usage: %s [options]\n", name)
	fmt.Println()
	fmt.Println(colorize(ColorBold, "Options:"))
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println(colorize(ColorBold, "Examples:"))
	fmt.Printf("  %s\n", name)
	fmt.Printf("  %s --config existing-config.json\n", name)
	fmt.Printf("  %s --output /etc/renos/config.json\n", name)
	fmt.Printf("  %s --dry-run\n", name)
	fmt.Printf("  %s --help\n", name)
	fmt.Println()
}
