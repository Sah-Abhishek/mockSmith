package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/Sah-Abhishek/mockSmith/config"
	"github.com/Sah-Abhishek/mockSmith/server"
	"github.com/Sah-Abhishek/mockSmith/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data", 0o755); err != nil {
		log.Fatal(err)
	}

	// Load or create config
	cfg, err := config.Load("data/endpoints.json")
	if err != nil {
		log.Fatal(err)
	}

	// Channel for config updates from TUI to server
	updateChan := make(chan *config.Config, 10)

	// WaitGroup to manage goroutines
	var wg sync.WaitGroup

	// Start HTTP server in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Start(cfg, updateChan); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Start TUI
	p := tea.NewProgram(tui.NewModel(cfg, updateChan), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	wg.Wait()
}
