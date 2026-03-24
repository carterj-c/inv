package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/tui"
)

func main() {
	if err := config.EnsureDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	app := tui.NewApp()
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
