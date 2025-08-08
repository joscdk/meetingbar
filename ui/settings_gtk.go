//go:build gtk

package ui

import (
	"context"
	"log"
	"runtime"

	"meetingbar/config"
	"meetingbar/ui/gtk"
)

type NativeSettingsManager struct {
	gtkManager *gtk.GTKSettingsManager
}

func NewNativeSettingsManager(cfg *config.Config, ctx context.Context, onRefresh func()) *NativeSettingsManager {
	return &NativeSettingsManager{
		gtkManager: gtk.NewGTKSettingsManager(cfg, ctx, onRefresh),
	}
}

func (nsm *NativeSettingsManager) ShowSettings() error {
	// Run GTK in a separate goroutine with proper OS thread locking
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		
		if err := nsm.gtkManager.ShowSettingsBlocking(); err != nil {
			log.Printf("GTK settings error: %v", err)
		}
	}()
	
	return nil
}