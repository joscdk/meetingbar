package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"meetingbar/calendar"
	"meetingbar/config"
)

type AdvancedSettingsManager struct {
	config          *config.Config
	calendarService *calendar.GoogleCalendarService
	ctx             context.Context
	scanner         *bufio.Scanner
}

func NewAdvancedSettingsManager(cfg *config.Config, ctx context.Context) *AdvancedSettingsManager {
	return &AdvancedSettingsManager{
		config:          cfg,
		calendarService: calendar.NewGoogleCalendarService(ctx),
		ctx:             ctx,
		scanner:         bufio.NewScanner(os.Stdin),
	}
}

func (sm *AdvancedSettingsManager) ShowSettings() error {
	// Check if zenity is available for GUI
	if sm.isZenityAvailable() {
		return sm.showGUISettings()
	}
	
	// Fall back to advanced terminal UI
	return sm.showTerminalSettings()
}

func (sm *AdvancedSettingsManager) showTerminalSettings() error {
	for {
		sm.clearScreen()
		sm.printHeader()
		sm.printSidebar()
		
		fmt.Print("\nSelect option (1-6, or 'q' to quit): ")
		if !sm.scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(sm.scanner.Text())
		
		switch input {
		case "1":
			sm.manageOAuth2Credentials()
		case "2":
			sm.manageAccounts()
		case "3":
			sm.manageCalendars()
		case "4":
			sm.manageNotifications()
		case "5":
			sm.manageGeneral()
		case "6":
			sm.showCurrentConfig()
		case "q", "Q":
			return nil
		default:
			fmt.Print("Invalid option. Press Enter to continue...")
			sm.scanner.Scan()
		}
	}
	return nil
}

func (sm *AdvancedSettingsManager) clearScreen() {
	// Clear screen for better UX
	cmd := exec.Command("clear")
	if cmd.Run() != nil {
		// If clear doesn't work, print some newlines
		fmt.Print("\n\n\n\n\n\n\n\n\n\n")
	}
}

func (sm *AdvancedSettingsManager) printHeader() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        MeetingBar Settings                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

func (sm *AdvancedSettingsManager) printSidebar() {
	fmt.Println("\n┌─ SETTINGS MENU ─────────────────────────────────────────────────┐")
	fmt.Println("│                                                                 │")
	
	// OAuth2 Status
	oauth2Status := "❌ Not configured"
	if sm.config.OAuth2.ClientID != "" && sm.config.OAuth2.ClientSecret != "" {
		oauth2Status = "✅ Configured"
	}
	fmt.Printf("│  1. 🔐 OAuth2 Credentials                    %s        │\n", oauth2Status)
	
	// Accounts status
	accountStatus := fmt.Sprintf("(%d accounts)", len(sm.config.Accounts))
	fmt.Printf("│  2. 👤 Google Accounts                       %-15s │\n", accountStatus)
	
	// Calendars status
	calendarStatus := fmt.Sprintf("(%d enabled)", len(sm.config.EnabledCalendars))
	fmt.Printf("│  3. 📅 Calendar Selection                    %-15s │\n", calendarStatus)
	
	// Notifications status
	notifStatus := "❌ Disabled"
	if sm.config.EnableNotifications {
		notifStatus = fmt.Sprintf("✅ %dm before", sm.config.NotificationTime)
	}
	fmt.Printf("│  4. 🔔 Notifications                         %-15s │\n", notifStatus)
	
	// General settings
	fmt.Printf("│  5. ⚙️  General Settings                     Refresh: %dm      │\n", sm.config.RefreshInterval)
	
	// View current config
	fmt.Println("│  6. 📋 View Current Configuration                               │")
	fmt.Println("│                                                                 │")
	fmt.Println("│  q. Quit Settings                                               │")
	fmt.Println("└─────────────────────────────────────────────────────────────────┘")
}

func (sm *AdvancedSettingsManager) manageOAuth2Credentials() {
	sm.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    OAuth2 Credentials Setup                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	fmt.Println("To use MeetingBar, you need to set up Google OAuth2 credentials:")
	fmt.Println()
	fmt.Println("1. Go to: https://console.cloud.google.com/")
	fmt.Println("2. Create a new project or select existing one")
	fmt.Println("3. Enable the Google Calendar API")
	fmt.Println("4. Create OAuth 2.0 Client IDs:")
	fmt.Println("   - Application type: Desktop application")
	fmt.Println("   - Authorized redirect URIs: http://localhost:8080/callback")
	fmt.Println()
	
	// Show current status
	if sm.config.OAuth2.ClientID != "" {
		fmt.Printf("Current Client ID: %s...%s\n", 
			sm.config.OAuth2.ClientID[:8], 
			sm.config.OAuth2.ClientID[len(sm.config.OAuth2.ClientID)-8:])
		fmt.Println("Current Client Secret: [CONFIGURED]")
		fmt.Println()
	}
	
	fmt.Println("Choose an option:")
	fmt.Println("1. Set new credentials")
	fmt.Println("2. Clear current credentials")
	fmt.Println("3. Back to main menu")
	fmt.Print("\nYour choice: ")
	
	if !sm.scanner.Scan() {
		return
	}
	
	choice := strings.TrimSpace(sm.scanner.Text())
	switch choice {
	case "1":
		sm.setOAuth2Credentials()
	case "2":
		sm.clearOAuth2Credentials()
	case "3":
		return
	}
	
	fmt.Print("\nPress Enter to continue...")
	sm.scanner.Scan()
}

func (sm *AdvancedSettingsManager) setOAuth2Credentials() {
	fmt.Print("\nEnter Google OAuth2 Client ID: ")
	if !sm.scanner.Scan() {
		return
	}
	clientID := strings.TrimSpace(sm.scanner.Text())
	
	fmt.Print("Enter Google OAuth2 Client Secret: ")
	if !sm.scanner.Scan() {
		return
	}
	clientSecret := strings.TrimSpace(sm.scanner.Text())
	
	if clientID == "" || clientSecret == "" {
		fmt.Println("❌ Both Client ID and Client Secret are required!")
		return
	}
	
	// Basic validation
	if len(clientID) < 20 || !strings.Contains(clientID, ".googleusercontent.com") {
		fmt.Println("⚠️  Warning: Client ID doesn't look like a valid Google OAuth2 Client ID")
		fmt.Print("Continue anyway? (y/N): ")
		if sm.scanner.Scan() {
			if strings.ToLower(strings.TrimSpace(sm.scanner.Text())) != "y" {
				return
			}
		}
	}
	
	sm.config.OAuth2.ClientID = clientID
	sm.config.OAuth2.ClientSecret = clientSecret
	
	if err := sm.config.Save(); err != nil {
		fmt.Printf("❌ Failed to save credentials: %v\n", err)
	} else {
		fmt.Println("✅ OAuth2 credentials saved successfully!")
	}
}

func (sm *AdvancedSettingsManager) clearOAuth2Credentials() {
	fmt.Print("Are you sure you want to clear OAuth2 credentials? (y/N): ")
	if !sm.scanner.Scan() {
		return
	}
	
	if strings.ToLower(strings.TrimSpace(sm.scanner.Text())) == "y" {
		sm.config.OAuth2.ClientID = ""
		sm.config.OAuth2.ClientSecret = ""
		
		if err := sm.config.Save(); err != nil {
			fmt.Printf("❌ Failed to clear credentials: %v\n", err)
		} else {
			fmt.Println("✅ OAuth2 credentials cleared successfully!")
		}
	}
}

func (sm *AdvancedSettingsManager) manageAccounts() {
	sm.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        Google Accounts                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	// Check OAuth2 credentials first
	if sm.config.OAuth2.ClientID == "" || sm.config.OAuth2.ClientSecret == "" {
		fmt.Println("❌ OAuth2 credentials not configured!")
		fmt.Println("Please set up OAuth2 credentials first (option 1 in main menu).")
		fmt.Print("\nPress Enter to continue...")
		sm.scanner.Scan()
		return
	}
	
	// Show current accounts
	if len(sm.config.Accounts) == 0 {
		fmt.Println("No Google accounts configured.")
	} else {
		fmt.Println("Current accounts:")
		for i, account := range sm.config.Accounts {
			fmt.Printf("  %d. %s (ID: %s)\n", i+1, account.Email, account.ID)
		}
	}
	
	fmt.Println("\nChoose an option:")
	fmt.Println("1. Add Google account")
	if len(sm.config.Accounts) > 0 {
		fmt.Println("2. Remove account")
	}
	fmt.Println("3. Back to main menu")
	fmt.Print("\nYour choice: ")
	
	if !sm.scanner.Scan() {
		return
	}
	
	choice := strings.TrimSpace(sm.scanner.Text())
	switch choice {
	case "1":
		sm.addGoogleAccount()
	case "2":
		if len(sm.config.Accounts) > 0 {
			sm.removeGoogleAccount()
		}
	case "3":
		return
	}
	
	fmt.Print("\nPress Enter to continue...")
	sm.scanner.Scan()
}

func (sm *AdvancedSettingsManager) addGoogleAccount() {
	fmt.Println("\n🔄 Starting OAuth2 flow...")
	fmt.Println("This will open a browser window for authentication.")
	fmt.Print("Continue? (Y/n): ")
	
	if sm.scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(sm.scanner.Text()))
		if response == "n" {
			return
		}
	}
	
	account, err := calendar.StartOAuth2Flow(sm.ctx, sm.config)
	if err != nil {
		fmt.Printf("❌ Failed to add account: %v\n", err)
		return
	}
	
	// Add to config
	sm.config.Accounts = append(sm.config.Accounts, *account)
	if err := sm.config.Save(); err != nil {
		fmt.Printf("❌ Failed to save account: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Successfully added account: %s\n", account.Email)
}

func (sm *AdvancedSettingsManager) removeGoogleAccount() {
	fmt.Println("\nSelect account to remove:")
	for i, account := range sm.config.Accounts {
		fmt.Printf("  %d. %s\n", i+1, account.Email)
	}
	fmt.Print("\nEnter number (or 0 to cancel): ")
	
	if !sm.scanner.Scan() {
		return
	}
	
	numStr := strings.TrimSpace(sm.scanner.Text())
	num, err := strconv.Atoi(numStr)
	if err != nil || num < 0 || num > len(sm.config.Accounts) {
		fmt.Println("❌ Invalid selection!")
		return
	}
	
	if num == 0 {
		return
	}
	
	account := sm.config.Accounts[num-1]
	fmt.Printf("Remove account: %s? (y/N): ", account.Email)
	
	if sm.scanner.Scan() {
		if strings.ToLower(strings.TrimSpace(sm.scanner.Text())) == "y" {
			// Remove from keyring
			config.DeleteToken(account.ID)
			
			// Remove from config
			sm.config.Accounts = append(sm.config.Accounts[:num-1], sm.config.Accounts[num:]...)
			
			if err := sm.config.Save(); err != nil {
				fmt.Printf("❌ Failed to save changes: %v\n", err)
			} else {
				fmt.Printf("✅ Account %s removed successfully!\n", account.Email)
			}
		}
	}
}

func (sm *AdvancedSettingsManager) manageCalendars() {
	sm.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      Calendar Selection                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	if len(sm.config.Accounts) == 0 {
		fmt.Println("❌ No Google accounts configured!")
		fmt.Println("Please add a Google account first (option 2 in main menu).")
		fmt.Print("\nPress Enter to continue...")
		sm.scanner.Scan()
		return
	}
	
	// Get all calendars from all accounts
	fmt.Println("🔄 Loading calendars...")
	var allCalendars []calendar.CalendarInfo
	for _, account := range sm.config.Accounts {
		calendars, err := sm.calendarService.GetCalendars(account.ID)
		if err != nil {
			fmt.Printf("⚠️  Failed to load calendars for %s: %v\n", account.Email, err)
			continue
		}
		allCalendars = append(allCalendars, calendars...)
	}
	
	if len(allCalendars) == 0 {
		fmt.Println("❌ No calendars found!")
		fmt.Print("\nPress Enter to continue...")
		sm.scanner.Scan()
		return
	}
	
	fmt.Printf("\nFound %d calendars:\n\n", len(allCalendars))
	
	// Show calendars with current status
	for i, cal := range allCalendars {
		enabled := "❌"
		for _, enabledID := range sm.config.EnabledCalendars {
			if enabledID == cal.ID {
				enabled = "✅"
				break
			}
		}
		fmt.Printf("  %d. %s %s\n", i+1, enabled, cal.Summary)
	}
	
	fmt.Println("\nChoose an option:")
	fmt.Println("1. Enable/disable specific calendar")
	fmt.Println("2. Enable all calendars")
	fmt.Println("3. Disable all calendars")
	fmt.Println("4. Back to main menu")
	fmt.Print("\nYour choice: ")
	
	if !sm.scanner.Scan() {
		return
	}
	
	choice := strings.TrimSpace(sm.scanner.Text())
	switch choice {
	case "1":
		sm.toggleCalendar(allCalendars)
	case "2":
		sm.enableAllCalendars(allCalendars)
	case "3":
		sm.disableAllCalendars()
	case "4":
		return
	}
	
	fmt.Print("\nPress Enter to continue...")
	sm.scanner.Scan()
}

func (sm *AdvancedSettingsManager) toggleCalendar(calendars []calendar.CalendarInfo) {
	fmt.Print("\nEnter calendar number to toggle: ")
	if !sm.scanner.Scan() {
		return
	}
	
	numStr := strings.TrimSpace(sm.scanner.Text())
	num, err := strconv.Atoi(numStr)
	if err != nil || num < 1 || num > len(calendars) {
		fmt.Println("❌ Invalid calendar number!")
		return
	}
	
	cal := calendars[num-1]
	
	// Check if calendar is currently enabled
	enabled := false
	for i, enabledID := range sm.config.EnabledCalendars {
		if enabledID == cal.ID {
			// Remove from enabled list
			sm.config.EnabledCalendars = append(sm.config.EnabledCalendars[:i], sm.config.EnabledCalendars[i+1:]...)
			enabled = true
			break
		}
	}
	
	if !enabled {
		// Add to enabled list
		sm.config.EnabledCalendars = append(sm.config.EnabledCalendars, cal.ID)
	}
	
	if err := sm.config.Save(); err != nil {
		fmt.Printf("❌ Failed to save changes: %v\n", err)
	} else {
		status := "enabled"
		if enabled {
			status = "disabled"
		}
		fmt.Printf("✅ Calendar '%s' %s!\n", cal.Summary, status)
	}
}

func (sm *AdvancedSettingsManager) enableAllCalendars(calendars []calendar.CalendarInfo) {
	sm.config.EnabledCalendars = nil
	for _, cal := range calendars {
		sm.config.EnabledCalendars = append(sm.config.EnabledCalendars, cal.ID)
	}
	
	if err := sm.config.Save(); err != nil {
		fmt.Printf("❌ Failed to save changes: %v\n", err)
	} else {
		fmt.Printf("✅ All %d calendars enabled!\n", len(calendars))
	}
}

func (sm *AdvancedSettingsManager) disableAllCalendars() {
	sm.config.EnabledCalendars = nil
	
	if err := sm.config.Save(); err != nil {
		fmt.Printf("❌ Failed to save changes: %v\n", err)
	} else {
		fmt.Println("✅ All calendars disabled!")
	}
}

func (sm *AdvancedSettingsManager) manageNotifications() {
	sm.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        Notifications                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	// Show current settings
	fmt.Printf("Current status: %s\n", map[bool]string{true: "✅ Enabled", false: "❌ Disabled"}[sm.config.EnableNotifications])
	if sm.config.EnableNotifications {
		fmt.Printf("Notification timing: %d minutes before meeting\n", sm.config.NotificationTime)
	}
	
	fmt.Println("\nChoose an option:")
	fmt.Println("1. Enable notifications")
	fmt.Println("2. Disable notifications")
	if sm.config.EnableNotifications {
		fmt.Println("3. Change notification timing")
	}
	fmt.Println("4. Back to main menu")
	fmt.Print("\nYour choice: ")
	
	if !sm.scanner.Scan() {
		return
	}
	
	choice := strings.TrimSpace(sm.scanner.Text())
	switch choice {
	case "1":
		sm.config.EnableNotifications = true
		if err := sm.config.Save(); err != nil {
			fmt.Printf("❌ Failed to save: %v\n", err)
		} else {
			fmt.Println("✅ Notifications enabled!")
		}
	case "2":
		sm.config.EnableNotifications = false
		if err := sm.config.Save(); err != nil {
			fmt.Printf("❌ Failed to save: %v\n", err)
		} else {
			fmt.Println("✅ Notifications disabled!")
		}
	case "3":
		if sm.config.EnableNotifications {
			sm.changeNotificationTiming()
		}
	case "4":
		return
	}
	
	fmt.Print("\nPress Enter to continue...")
	sm.scanner.Scan()
}

func (sm *AdvancedSettingsManager) changeNotificationTiming() {
	fmt.Println("\nSelect notification timing:")
	options := []int{1, 5, 10, 15, 30}
	for i, minutes := range options {
		marker := "  "
		if sm.config.NotificationTime == minutes {
			marker = "▶️"
		}
		fmt.Printf("  %s %d. %d minutes before\n", marker, i+1, minutes)
	}
	
	fmt.Print("\nYour choice (1-5): ")
	if !sm.scanner.Scan() {
		return
	}
	
	choiceStr := strings.TrimSpace(sm.scanner.Text())
	choice, err := strconv.Atoi(choiceStr)
	if err != nil || choice < 1 || choice > len(options) {
		fmt.Println("❌ Invalid choice!")
		return
	}
	
	sm.config.NotificationTime = options[choice-1]
	if err := sm.config.Save(); err != nil {
		fmt.Printf("❌ Failed to save: %v\n", err)
	} else {
		fmt.Printf("✅ Notification timing set to %d minutes before meeting!\n", sm.config.NotificationTime)
	}
}

func (sm *AdvancedSettingsManager) manageGeneral() {
	sm.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      General Settings                         ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	fmt.Printf("Launch at login: %s\n", map[bool]string{true: "✅ Enabled", false: "❌ Disabled"}[sm.config.LaunchAtLogin])
	fmt.Printf("Calendar refresh interval: %d minutes\n", sm.config.RefreshInterval)
	
	fmt.Println("\nChoose an option:")
	fmt.Println("1. Toggle launch at login")
	fmt.Println("2. Change refresh interval")
	fmt.Println("3. Back to main menu")
	fmt.Print("\nYour choice: ")
	
	if !sm.scanner.Scan() {
		return
	}
	
	choice := strings.TrimSpace(sm.scanner.Text())
	switch choice {
	case "1":
		sm.config.LaunchAtLogin = !sm.config.LaunchAtLogin
		if err := sm.config.Save(); err != nil {
			fmt.Printf("❌ Failed to save: %v\n", err)
		} else {
			status := "enabled"
			if !sm.config.LaunchAtLogin {
				status = "disabled"
			}
			fmt.Printf("✅ Launch at login %s!\n", status)
		}
	case "2":
		sm.changeRefreshInterval()
	case "3":
		return
	}
	
	fmt.Print("\nPress Enter to continue...")
	sm.scanner.Scan()
}

func (sm *AdvancedSettingsManager) changeRefreshInterval() {
	fmt.Println("\nSelect refresh interval:")
	options := []int{1, 5, 10, 15, 30}
	for i, minutes := range options {
		marker := "  "
		if sm.config.RefreshInterval == minutes {
			marker = "▶️"
		}
		fmt.Printf("  %s %d. %d minutes\n", marker, i+1, minutes)
	}
	
	fmt.Print("\nYour choice (1-5): ")
	if !sm.scanner.Scan() {
		return
	}
	
	choiceStr := strings.TrimSpace(sm.scanner.Text())
	choice, err := strconv.Atoi(choiceStr)
	if err != nil || choice < 1 || choice > len(options) {
		fmt.Printf("❌ Invalid choice!\n")
		return
	}
	
	sm.config.RefreshInterval = options[choice-1]
	if err := sm.config.Save(); err != nil {
		fmt.Printf("❌ Failed to save: %v\n", err)
	} else {
		fmt.Printf("✅ Refresh interval set to %d minutes!\n", sm.config.RefreshInterval)
	}
}

func (sm *AdvancedSettingsManager) showCurrentConfig() {
	sm.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Current Configuration                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	
	// OAuth2 Credentials
	fmt.Println("🔐 OAuth2 Credentials:")
	if sm.config.OAuth2.ClientID != "" {
		fmt.Printf("   Client ID: %s...%s\n", 
			sm.config.OAuth2.ClientID[:8], 
			sm.config.OAuth2.ClientID[len(sm.config.OAuth2.ClientID)-8:])
		fmt.Println("   Client Secret: [CONFIGURED]")
	} else {
		fmt.Println("   ❌ Not configured")
	}
	
	// Accounts
	fmt.Printf("\n👤 Google Accounts (%d):\n", len(sm.config.Accounts))
	if len(sm.config.Accounts) == 0 {
		fmt.Println("   ❌ No accounts configured")
	} else {
		for i, account := range sm.config.Accounts {
			fmt.Printf("   %d. %s (ID: %s)\n", i+1, account.Email, account.ID)
		}
	}
	
	// Calendars
	fmt.Printf("\n📅 Enabled Calendars (%d):\n", len(sm.config.EnabledCalendars))
	if len(sm.config.EnabledCalendars) == 0 {
		fmt.Println("   ❌ No calendars enabled")
	} else {
		for i, calID := range sm.config.EnabledCalendars {
			fmt.Printf("   %d. %s\n", i+1, calID)
		}
	}
	
	// Notifications
	fmt.Printf("\n🔔 Notifications: %s\n", map[bool]string{true: "✅ Enabled", false: "❌ Disabled"}[sm.config.EnableNotifications])
	if sm.config.EnableNotifications {
		fmt.Printf("   Timing: %d minutes before meeting\n", sm.config.NotificationTime)
	}
	
	// General
	fmt.Printf("\n⚙️  General Settings:\n")
	fmt.Printf("   Refresh interval: %d minutes\n", sm.config.RefreshInterval)
	fmt.Printf("   Launch at login: %s\n", map[bool]string{true: "✅ Yes", false: "❌ No"}[sm.config.LaunchAtLogin])
	
	// File locations
	fmt.Printf("\n📁 File Locations:\n")
	fmt.Println("   Config: ~/.config/meetingbar/config.json")
	fmt.Println("   Cache: ~/.cache/meetingbar/")
	fmt.Println("   Credentials: System keyring")
	
	fmt.Print("\nPress Enter to continue...")
	sm.scanner.Scan()
}

func (sm *AdvancedSettingsManager) isZenityAvailable() bool {
	_, err := exec.LookPath("zenity")
	return err == nil
}

func (sm *AdvancedSettingsManager) showGUISettings() error {
	// This could be implemented with zenity forms for a GUI experience
	// For now, fall back to terminal UI even if zenity is available
	// since the terminal UI is much more comprehensive
	return sm.showTerminalSettings()
}