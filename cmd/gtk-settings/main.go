package main

import (
	"context"
	"log"
	"os"

	"meetingbar/config"
	"meetingbar/ui/gtk"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	ctx := context.Background()
	
	// Create GTK settings manager - this runs in separate process to avoid conflicts
	settingsMgr := gtk.NewGTKSettingsManager(cfg, ctx, nil)
	
	// Show settings and block until closed
	if err := settingsMgr.ShowSettingsBlocking(); err != nil {
		log.Printf("Settings error: %v", err)
		os.Exit(1)
	}
}