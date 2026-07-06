package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/esteban/rbchat/internal/db"
	"github.com/esteban/rbchat/internal/network"
	"github.com/esteban/rbchat/internal/tui"
)

const multicastAddr = "224.0.0.1:9999"

var (
	rbchatSecret string
	version      = "dev"
	commit       = "none"
	date         = "unknown"
)

func printVersion() {
	if version == "dev" {
		fmt.Println("rbchat dev")
		return
	}
	fmt.Printf("rbchat v%s (commit %s, built %s)\n", version, commit, date)
}

func dataDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "rbchat")
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			printVersion()
			return
		}
	}

	if len(os.Args) > 1 && os.Args[1] == "update" {
		cmdUpdate()
		return
	}

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

	os.MkdirAll(lockPath, 0755)
	pidFile := filepath.Join(lockPath, fmt.Sprintf("%d.pid", os.Getpid()))

	entries, _ := os.ReadDir(lockPath)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(lockPath, e.Name()))
		if err != nil {
			os.Remove(filepath.Join(lockPath, e.Name()))
			continue
		}
		var pid int
		fmt.Sscanf(string(data), "%d", &pid)
		if pid <= 0 || !isProcessAlive(pid) {
			os.Remove(filepath.Join(lockPath, e.Name()))
			continue
		}
		otherInstanceRunning = true
	}

	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	defer os.Remove(pidFile)

	network.SetSecret(rbchatSecret)

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

	networkID := network.ComputeNetworkID()

	model := tui.NewModel(database, username, team, listener, broadcaster, msgCh, ctx, cancel, notificationsEnabled, otherInstanceRunning, networkID, version)

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

func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
