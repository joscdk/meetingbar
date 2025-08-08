//go:build !gtk

package ui

import (
	"context"
	"log"

	"meetingbar/config"
)

type NativeSettingsManager struct {
	webManager *WebSettingsManager
}

func NewNativeSettingsManager(cfg *config.Config, ctx context.Context, onRefresh func()) *NativeSettingsManager {
	return &NativeSettingsManager{
		webManager: NewWebSettingsManager(cfg, ctx),
	}
}

func (nsm *NativeSettingsManager) ShowSettings() error {
	log.Printf("NativeSettingsManager: Using web settings fallback")
	return nsm.webManager.ShowSettings()
}