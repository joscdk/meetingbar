package ui

import (
	"context"
	"fmt"
	"log"
	"strings"

	"meetingbar/calendar"
	"meetingbar/config"

	"github.com/ncruces/zenity"
)

type SettingsManager struct {
	config          *config.Config
	calendarService *calendar.GoogleCalendarService
	ctx             context.Context
}

func NewSettingsManager(cfg *config.Config, ctx context.Context) *SettingsManager {
	return &SettingsManager{
		config:          cfg,
		calendarService: calendar.NewGoogleCalendarService(ctx),
		ctx:             ctx,
	}
}

func (sm *SettingsManager) ShowSettings() error {
	// For now, we'll create a text-based settings dialog
	// In a full implementation, this would be a proper GUI
	
	for {
		choice, err := sm.showMainMenu()
		if err != nil {
			return err
		}
		
		switch choice {
		case "Accounts":
			if err := sm.manageAccounts(); err != nil {
				log.Printf("Error managing accounts: %v", err)
			}
		case "Calendars":
			if err := sm.manageCalendars(); err != nil {
				log.Printf("Error managing calendars: %v", err)
			}
		case "Notifications":
			if err := sm.manageNotifications(); err != nil {
				log.Printf("Error managing notifications: %v", err)
			}
		case "General":
			if err := sm.manageGeneral(); err != nil {
				log.Printf("Error managing general settings: %v", err)
			}
		case "Done":
			return nil
		}
	}
}

func (sm *SettingsManager) showMainMenu() (string, error) {
	options := []string{
		"Accounts",
		"Calendars", 
		"Notifications",
		"General",
		"Done",
	}
	
	choice, err := zenity.List(
		"MeetingBar Settings",
		options,
		zenity.Title("Settings"),
		zenity.Width(400),
		zenity.Height(300),
	)
	
	return choice, err
}

func (sm *SettingsManager) manageAccounts() error {
	for {
		var accountList []string
		accountList = append(accountList, "Add Google Account")
		
		for _, account := range sm.config.Accounts {
			accountList = append(accountList, fmt.Sprintf("Remove: %s", account.Email))
		}
		
		accountList = append(accountList, "Back")
		
		choice, err := zenity.List(
			"Select an option:",
			accountList,
			zenity.Title("Manage Accounts"),
			zenity.Width(400),
			zenity.Height(300),
		)
		
		if err != nil {
			return err
		}
		
		if choice == "Back" {
			return nil
		}
		
		if choice == "Add Google Account" {
			if err := sm.addGoogleAccount(); err != nil {
				zenity.Error(fmt.Sprintf("Failed to add account: %v", err))
			}
		} else if strings.HasPrefix(choice, "Remove: ") {
			email := strings.TrimPrefix(choice, "Remove: ")
			if err := sm.removeAccount(email); err != nil {
				zenity.Error(fmt.Sprintf("Failed to remove account: %v", err))
			}
		}
	}
}

func (sm *SettingsManager) addGoogleAccount() error {
	// This would need OAuth2 client credentials
	// For now, show a message about setup required
	return zenity.Info(
		"To add a Google account, you need to configure OAuth2 credentials.\n\n" +
		"Please set up your Google Cloud project and configure the OAuth2 client ID and secret.",
		zenity.Title("OAuth2 Setup Required"),
	)
}

func (sm *SettingsManager) removeAccount(email string) error {
	err := zenity.Question(
		fmt.Sprintf("Are you sure you want to remove account: %s?", email),
		zenity.Title("Confirm Removal"),
	)
	
	if err != nil {
		return err
	}
	
	// Find and remove account
	for i, account := range sm.config.Accounts {
		if account.Email == email {
			// Remove from keyring
			config.DeleteToken(account.ID)
			
			// Remove from config
			sm.config.Accounts = append(sm.config.Accounts[:i], sm.config.Accounts[i+1:]...)
			break
		}
	}
	
	return sm.config.Save()
}

func (sm *SettingsManager) manageCalendars() error {
	if len(sm.config.Accounts) == 0 {
		return zenity.Info(
			"No accounts configured. Please add a Google account first.",
			zenity.Title("No Accounts"),
		)
	}
	
	// Get all calendars from all accounts
	var allCalendars []config.Calendar
	for _, account := range sm.config.Accounts {
		calendars, err := sm.calendarService.GetCalendars(account.ID)
		if err != nil {
			log.Printf("Failed to get calendars for %s: %v", account.Email, err)
			continue
		}
		allCalendars = append(allCalendars, calendars...)
	}
	
	if len(allCalendars) == 0 {
		return zenity.Info(
			"No calendars found. Please check your account permissions.",
			zenity.Title("No Calendars"),
		)
	}
	
	// Create calendar selection list
	var calendarOptions []string
	var selectedCalendars []string
	
	for _, cal := range allCalendars {
		option := fmt.Sprintf("%s (%s)", cal.Name, cal.AccountID)
		calendarOptions = append(calendarOptions, option)
		
		// Check if calendar is enabled
		for _, enabledID := range sm.config.EnabledCalendars {
			if enabledID == cal.ID {
				selectedCalendars = append(selectedCalendars, option)
				break
			}
		}
	}
	
	selected, err := zenity.ListMultiple(
		"Select calendars to monitor:",
		calendarOptions,
		zenity.Title("Calendar Selection"),
		zenity.Width(500),
		zenity.Height(400),
	)
	
	if err != nil {
		return err
	}
	
	// Update enabled calendars
	sm.config.EnabledCalendars = nil
	for _, selectedOption := range selected {
		for _, cal := range allCalendars {
			option := fmt.Sprintf("%s (%s)", cal.Name, cal.AccountID)
			if option == selectedOption {
				sm.config.EnabledCalendars = append(sm.config.EnabledCalendars, cal.ID)
				break
			}
		}
	}
	
	return sm.config.Save()
}

func (sm *SettingsManager) manageNotifications() error {
	// Notification enable/disable
	err := zenity.Question(
		"Enable meeting notifications?",
		zenity.Title("Notifications"),
	)
	
	enabled := (err == nil)
	sm.config.EnableNotifications = enabled
	
	if enabled {
		// Notification timing
		timingOptions := []string{"1 minute", "5 minutes", "10 minutes", "15 minutes"}
		currentTiming := fmt.Sprintf("%d minutes", sm.config.NotificationTime)
		
		timing, err := zenity.List(
			fmt.Sprintf("Notify how many minutes before meetings? (Current: %s)", currentTiming),
			timingOptions,
			zenity.Title("Notification Timing"),
		)
		
		if err != nil {
			return err
		}
		
		// Parse selected timing
		if strings.Contains(timing, "1 minute") {
			sm.config.NotificationTime = 1
		} else if strings.Contains(timing, "5 minutes") {
			sm.config.NotificationTime = 5
		} else if strings.Contains(timing, "10 minutes") {
			sm.config.NotificationTime = 10
		} else if strings.Contains(timing, "15 minutes") {
			sm.config.NotificationTime = 15
		}
	}
	
	return sm.config.Save()
}

func (sm *SettingsManager) manageGeneral() error {
	// Launch at login
	err := zenity.Question(
		"Launch MeetingBar at login?",
		zenity.Title("Startup"),
	)
	
	launchAtLogin := (err == nil)
	sm.config.LaunchAtLogin = launchAtLogin
	
	// Refresh interval
	intervalOptions := []string{"1 minute", "5 minutes", "10 minutes", "30 minutes"}
	currentInterval := fmt.Sprintf("%d minutes", sm.config.RefreshInterval)
	
	interval, err := zenity.List(
		fmt.Sprintf("Calendar refresh interval? (Current: %s)", currentInterval),
		intervalOptions,
		zenity.Title("Refresh Interval"),
	)
	
	if err != nil {
		return err
	}
	
	// Parse selected interval
	if strings.Contains(interval, "1 minute") {
		sm.config.RefreshInterval = 1
	} else if strings.Contains(interval, "5 minutes") {
		sm.config.RefreshInterval = 5
	} else if strings.Contains(interval, "10 minutes") {
		sm.config.RefreshInterval = 10
	} else if strings.Contains(interval, "30 minutes") {
		sm.config.RefreshInterval = 30
	}
	
	return sm.config.Save()
}

// Simple settings dialog using text input (fallback when zenity is not available)
func (sm *SettingsManager) showSimpleSettings() error {
	fmt.Println("=== MeetingBar Settings ===")
	fmt.Printf("Accounts: %d configured\n", len(sm.config.Accounts))
	fmt.Printf("Enabled Calendars: %d\n", len(sm.config.EnabledCalendars))
	fmt.Printf("Notifications: %t\n", sm.config.EnableNotifications)
	fmt.Printf("Notification Time: %d minutes\n", sm.config.NotificationTime)
	fmt.Printf("Refresh Interval: %d minutes\n", sm.config.RefreshInterval)
	fmt.Printf("Launch at Login: %t\n", sm.config.LaunchAtLogin)
	fmt.Println("==========================")
	
	return nil
}