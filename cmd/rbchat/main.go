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
	notificationsEnabled := true
	for _, arg := range os.Args[1:] {
		if arg == "--no-notify" {
			notificationsEnabled = false
		}
	}

	dd := dataDir()
	database, err := db.Init(dd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	lockPath := filepath.Join(dd, "rbchat.lock")
	otherInstanceRunning := false
	lf, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err == nil {
		fmt.Fprintf(lf, "%d", os.Getpid())
		lf.Close()
		defer os.Remove(lockPath)
	} else {
		otherInstanceRunning = true
	}

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

	model := tui.NewModel(database, username, team, listener, broadcaster, msgCh, ctx, cancel, notificationsEnabled, otherInstanceRunning)

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
