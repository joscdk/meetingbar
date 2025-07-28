package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Accounts                []Account    `mapstructure:"accounts"`
	EnabledCalendars        []string     `mapstructure:"enabled_calendars"`
	RefreshInterval         int          `mapstructure:"refresh_interval"` // minutes
	NotificationTime        int          `mapstructure:"notification_time"` // minutes before meeting
	EnableNotifications     bool         `mapstructure:"enable_notifications"`
	ShowMeetingLinks        bool         `mapstructure:"show_meeting_links"`
	PersistentNotifications bool         `mapstructure:"persistent_notifications"`
	NotificationSound       bool         `mapstructure:"notification_sound"`
	ShowDuration            bool         `mapstructure:"show_duration"`
	MaxMeetings             int          `mapstructure:"max_meetings"`
	StartWithSystem         bool         `mapstructure:"start_with_system"`
	AutoRefreshStartup      bool         `mapstructure:"auto_refresh_startup"`
	LaunchAtLogin           bool         `mapstructure:"launch_at_login"`
	Debug                   bool         `mapstructure:"debug"`
	OAuth2                  OAuth2Config `mapstructure:"oauth2"`
}

type OAuth2Config struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

type Account struct {
	ID      string    `mapstructure:"id"`
	Email   string    `mapstructure:"email"`
	AddedAt time.Time `mapstructure:"added_at"`
}

type Calendar struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AccountID string `json:"account_id"`
	Enabled   bool   `json:"enabled"`
	Color     string `json:"color"`
}

const (
	DefaultRefreshInterval          = 5     // minutes
	DefaultNotificationTime         = 5     // minutes
	DefaultEnableNotifications      = true
	DefaultShowMeetingLinks         = true
	DefaultPersistentNotifications  = false
	DefaultNotificationSound        = true
	DefaultShowDuration             = false
	DefaultMaxMeetings              = 5
	DefaultStartWithSystem          = false
	DefaultAutoRefreshStartup       = true
	DefaultLaunchAtLogin            = false
)

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}
	
	viper.AddConfigPath(configDir)
	
	// Set defaults
	viper.SetDefault("refresh_interval", DefaultRefreshInterval)
	viper.SetDefault("notification_time", DefaultNotificationTime)
	viper.SetDefault("enable_notifications", DefaultEnableNotifications)
	viper.SetDefault("show_meeting_links", DefaultShowMeetingLinks)
	viper.SetDefault("persistent_notifications", DefaultPersistentNotifications)
	viper.SetDefault("notification_sound", DefaultNotificationSound)
	viper.SetDefault("show_duration", DefaultShowDuration)
	viper.SetDefault("max_meetings", DefaultMaxMeetings)
	viper.SetDefault("start_with_system", DefaultStartWithSystem)
	viper.SetDefault("auto_refresh_startup", DefaultAutoRefreshStartup)
	viper.SetDefault("launch_at_login", DefaultLaunchAtLogin)
	viper.SetDefault("debug", false)
	viper.SetDefault("accounts", []Account{})
	viper.SetDefault("enabled_calendars", []string{})
	viper.SetDefault("oauth2", OAuth2Config{})
	
	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, create with defaults
		if err := ensureConfigDir(); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
	}
	
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

func (c *Config) Save() error {
	if err := ensureConfigDir(); err != nil {
		return fmt.Errorf("failed to ensure config directory: %w", err)
	}
	
	viper.Set("accounts", c.Accounts)
	viper.Set("enabled_calendars", c.EnabledCalendars)
	viper.Set("refresh_interval", c.RefreshInterval)
	viper.Set("notification_time", c.NotificationTime)
	viper.Set("enable_notifications", c.EnableNotifications)
	viper.Set("show_meeting_links", c.ShowMeetingLinks)
	viper.Set("persistent_notifications", c.PersistentNotifications)
	viper.Set("notification_sound", c.NotificationSound)
	viper.Set("show_duration", c.ShowDuration)
	viper.Set("max_meetings", c.MaxMeetings)
	viper.Set("start_with_system", c.StartWithSystem)
	viper.Set("auto_refresh_startup", c.AutoRefreshStartup)
	viper.Set("launch_at_login", c.LaunchAtLogin)
	viper.Set("debug", c.Debug)
	viper.Set("oauth2", c.OAuth2)
	
	return viper.WriteConfig()
}

func (c *Config) GetRefreshDuration() time.Duration {
	return time.Duration(c.RefreshInterval) * time.Minute
}

func (c *Config) GetNotificationDuration() time.Duration {
	return time.Duration(c.NotificationTime) * time.Minute
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "meetingbar"), nil
}

func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".cache", "meetingbar"), nil
}

func ensureConfigDir() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(configDir, 0755)
}

func EnsureCacheDir() error {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(cacheDir, 0755)
}

// NewConfig creates a new config with default values
func NewConfig() *Config {
	return &Config{
		Accounts:                []Account{},
		EnabledCalendars:        []string{},
		RefreshInterval:         DefaultRefreshInterval,
		NotificationTime:        DefaultNotificationTime,
		EnableNotifications:     DefaultEnableNotifications,
		ShowMeetingLinks:        DefaultShowMeetingLinks,
		PersistentNotifications: DefaultPersistentNotifications,
		NotificationSound:       DefaultNotificationSound,
		ShowDuration:            DefaultShowDuration,
		MaxMeetings:             DefaultMaxMeetings,
		StartWithSystem:         DefaultStartWithSystem,
		AutoRefreshStartup:      DefaultAutoRefreshStartup,
		LaunchAtLogin:           DefaultLaunchAtLogin,
		Debug:                   false,
		OAuth2:                  OAuth2Config{},
	}
}