package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const installScriptURL = "https://raw.githubusercontent.com/Esteban-Bermudez/rbchat/main/install.sh"

const releaseURL = "https://github.com/Esteban-Bermudez/rbchat/releases/latest"

func cmdUpdate() {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("sh"); err != nil {
			fmt.Println("Download the latest release from:")
			fmt.Println(releaseURL)
			return
		}
	}

	cmd := exec.Command("sh", "-c", "curl -sfL "+installScriptURL+" | sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}
}
