package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Accounts          []Account `mapstructure:"accounts"`
	EnabledCalendars  []string  `mapstructure:"enabled_calendars"`
	RefreshInterval   int       `mapstructure:"refresh_interval"`   // minutes
	NotificationTime  int       `mapstructure:"notification_time"`  // minutes before meeting
	EnableNotifications bool    `mapstructure:"enable_notifications"`
	LaunchAtLogin     bool      `mapstructure:"launch_at_login"`
	Debug             bool      `mapstructure:"debug"`
}

type Account struct {
	ID    string `mapstructure:"id"`
	Email string `mapstructure:"email"`
}

type Calendar struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AccountID string `json:"account_id"`
	Enabled   bool   `json:"enabled"`
	Color     string `json:"color"`
}

const (
	DefaultRefreshInterval   = 5  // minutes
	DefaultNotificationTime  = 5  // minutes
	DefaultEnableNotifications = true
	DefaultLaunchAtLogin     = false
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
	viper.SetDefault("launch_at_login", DefaultLaunchAtLogin)
	viper.SetDefault("debug", false)
	viper.SetDefault("accounts", []Account{})
	viper.SetDefault("enabled_calendars", []string{})
	
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
	viper.Set("launch_at_login", c.LaunchAtLogin)
	viper.Set("debug", c.Debug)
	
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