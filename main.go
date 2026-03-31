package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	internalaws "tui-ssm/internal/aws"
	"tui-ssm/internal/config"
	"tui-ssm/internal/store"
	"tui-ssm/internal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("tui-ssm %s\n", version)
		os.Exit(0)
	}

	// Prerequisite checks
	results := internalaws.CheckPrerequisites()
	for _, r := range results {
		if !r.OK {
			fmt.Fprintf(os.Stderr, "ERROR: %s — %s\n", r.Name, r.Message)
			os.Exit(1)
		}
	}

	// Load config
	cfg, err := config.Load(config.Path())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Ensure data directory exists
	os.MkdirAll(config.Dir(), 0o755)

	// Load stores
	favs, err := store.LoadFavorites(store.FavoritesPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load favorites: %v\n", err)
		os.Exit(1)
	}

	hist, err := store.LoadHistory(store.HistoryPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load history: %v\n", err)
		os.Exit(1)
	}

	// Parse AWS profiles (always includes "(instance role)" for EC2 IAM roles)
	profiles := internalaws.ParseProfiles(
		internalaws.DefaultCredentialsPath(),
		internalaws.DefaultConfigPath(),
	)

	// Create and run TUI
	model := ui.NewModel(cfg, profiles, favs, hist)
	p := tea.NewProgram(model, tea.WithFilter(ui.InterruptFilter))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
