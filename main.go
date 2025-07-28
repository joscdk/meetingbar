package main

import (
	"log"
	"os"

	"meetingbar/config"
	"meetingbar/ui"

	"github.com/getlantern/systray"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	if !cfg.Debug {
		log.SetOutput(os.Stderr)
	}

	// Run system tray
	systray.Run(func() {
		ui.OnReady(cfg)
	}, func() {
		ui.OnExit()
	})
}