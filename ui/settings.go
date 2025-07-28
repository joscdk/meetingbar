package ui

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"meetingbar/calendar"
	"meetingbar/config"

	"github.com/ncruces/zenity"
)

type SettingsManager struct {
	config          *config.Config
	calendarService *calendar.UnifiedCalendarService
	ctx             context.Context
}

func NewSettingsManager(cfg *config.Config, ctx context.Context) *SettingsManager {
	return &SettingsManager{
		config:          cfg,
		calendarService: calendar.NewUnifiedCalendarService(ctx, cfg),
		ctx:             ctx,
	}
}

func (sm *SettingsManager) ShowSettings() error {
	// Check if zenity is available
	if !sm.isZenityAvailable() {
		log.Println("Zenity not found, using fallback settings display")
		return sm.showSimpleSettings()
	}
	
	for {
		choice, err := sm.showMainMenu()
		if err != nil {
			// If zenity fails, fall back to simple settings
			log.Printf("Zenity error: %v, falling back to simple settings", err)
			return sm.showSimpleSettings()
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
	if sm.calendarService.IsGnomeBackend() {
		// For GNOME backend, get calendars directly
		calendars, err := sm.calendarService.GetCalendars("")
		if err != nil {
			log.Printf("Failed to get GNOME calendars: %v", err)
		} else {
			allCalendars = calendars
		}
	} else {
		// For Google backend, iterate through accounts
		for _, account := range sm.config.Accounts {
			calendars, err := sm.calendarService.GetCalendars(account.ID)
			if err != nil {
				log.Printf("Failed to get calendars for %s: %v", account.Email, err)
				continue
			}
			allCalendars = append(allCalendars, calendars...)
		}
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
		option := fmt.Sprintf("%s", cal.Name)
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
			option := fmt.Sprintf("%s", cal.Name)
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

func (sm *SettingsManager) isZenityAvailable() bool {
	_, err := exec.LookPath("zenity")
	return err == nil
}

// Simple settings dialog using text input (fallback when zenity is not available)
func (sm *SettingsManager) showSimpleSettings() error {
	fmt.Println("\n=== MeetingBar Settings ===")
	fmt.Printf("Accounts: %d configured\n", len(sm.config.Accounts))
	for i, account := range sm.config.Accounts {
		fmt.Printf("  %d. %s (ID: %s)\n", i+1, account.Email, account.ID)
	}
	
	fmt.Printf("\nEnabled Calendars: %d\n", len(sm.config.EnabledCalendars))
	for i, calID := range sm.config.EnabledCalendars {
		fmt.Printf("  %d. %s\n", i+1, calID)
	}
	
	fmt.Printf("\nNotifications: %t\n", sm.config.EnableNotifications)
	fmt.Printf("Notification Time: %d minutes before meeting\n", sm.config.NotificationTime)
	fmt.Printf("Refresh Interval: %d minutes\n", sm.config.RefreshInterval)
	fmt.Printf("Launch at Login: %t\n", sm.config.LaunchAtLogin)
	
	fmt.Println("\n=== Configuration Help ===")
	fmt.Println("To add a Google account:")
	fmt.Println("1. Set up Google OAuth2 credentials (see README.md)")
	fmt.Println("2. Set environment variables:")
	fmt.Println("   export GOOGLE_CLIENT_ID=\"your-client-id\"")
	fmt.Println("   export GOOGLE_CLIENT_SECRET=\"your-client-secret\"")
	fmt.Println("3. Install zenity for GUI settings: sudo apt install zenity")
	fmt.Println("\nConfig file location: ~/.config/meetingbar/config.json")
	fmt.Println("==========================\n")
	
	return nil
}