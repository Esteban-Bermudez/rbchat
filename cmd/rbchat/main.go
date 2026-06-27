package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/esteban/rbchat/internal/db"
	"github.com/esteban/rbchat/internal/network"
	"github.com/esteban/rbchat/internal/tui"
)

const multicastAddr = "224.0.0.1:9999"

func dataDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "rbchat")
}

func main() {
	dd := dataDir()
	database, err := db.Init(dd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	username, team, err := tui.RunSetup(database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
		os.Exit(1)
	}

	listener, err := network.NewListener(multicastAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating network listener: %v\n", err)
		os.Exit(1)
	}

	broadcaster, err := network.NewBroadcaster(multicastAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating network broadcaster: %v\n", err)
		os.Exit(1)
	}

	msgCh := make(chan network.IncomingMessage, 100)
	ctx, cancel := context.WithCancel(context.Background())

	model := tui.NewModel(database, username, team, listener, broadcaster, msgCh, ctx, cancel)

	go listener.Listen(ctx, msgCh)

	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	cancel()
	listener.Close()
	broadcaster.Close()
}
