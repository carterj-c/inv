package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/git"
	"github.com/carter/inv/internal/pdfsync"
	"github.com/carter/inv/internal/tui"
)

func main() {
	if err := config.EnsureDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "sync" {
		if err := runSync(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	app := tui.NewApp()
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSync() error {
	if !config.Exists() {
		return fmt.Errorf("config not found; run inv first to complete setup")
	}

	global, err := config.LoadGlobal()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	result, err := pdfsync.Sync(global)
	if err != nil {
		return err
	}

	if result.ArchiveUpdated > 0 && git.IsRepo(config.Dir()) {
		if err := git.CommitAll(config.Dir(), "sync pdf archive"); err != nil {
			return fmt.Errorf("committing synced archive: %w", err)
		}
		if git.HasRemote(config.Dir()) {
			if err := git.Push(config.Dir()); err != nil {
				return fmt.Errorf("pushing synced archive: %w", err)
			}
		}
	}

	fmt.Printf(
		"sync complete: updated %d export files and %d archived files\n",
		result.ExportUpdated,
		result.ArchiveUpdated,
	)
	return nil
}
