package ui

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"
	"time"

	"meetingbar/calendar"
	"meetingbar/config"

	"github.com/getlantern/systray"
)

type TrayManager struct {
	config          *config.Config
	calendarService *calendar.UnifiedCalendarService
	meetings        []calendar.Meeting
	ticker          *time.Ticker
	ctx             context.Context
	cancel          context.CancelFunc
	notificationMgr *NotificationManager
	settingsMgr     *WebSettingsManager
	
	// Menu items
	titleItem         *systray.MenuItem
	meetingItems      []*systray.MenuItem
	refreshItem       *systray.MenuItem
	settingsItem      *systray.MenuItem
	quitItem          *systray.MenuItem
	createItem        *systray.MenuItem
	rateItem          *systray.MenuItem
	quickActionsHeader *systray.MenuItem
	
	// Pre-allocated meeting items to maintain order
	maxMeetingSlots   int
	meetingSlots      []*systray.MenuItem
}

var trayManager *TrayManager

func OnReady(cfg *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	
	trayManager = &TrayManager{
		config:          cfg,
		calendarService: calendar.NewUnifiedCalendarService(ctx, cfg),
		ctx:             ctx,
		cancel:          cancel,
		notificationMgr: NewNotificationManager(cfg),
		settingsMgr:     NewWebSettingsManager(cfg, ctx),
	}
	
	trayManager.setupTray()
	trayManager.startPeriodicRefresh()
	trayManager.notificationMgr.StartNotificationWatcher()
	trayManager.refreshMeetings()
}

func OnExit() {
	if trayManager != nil {
		trayManager.cleanup()
	}
}

func (tm *TrayManager) setupTray() {
	// Set tray icon
	systray.SetIcon(getDefaultIcon())
	systray.SetTitle("MeetingBar")
	systray.SetTooltip("MeetingBar - No meetings")
	
	// Create menu structure
	tm.setupMenuStructure()
	
	// Handle menu clicks
	go tm.handleMenuClicks()
}

func (tm *TrayManager) setupMenuStructure() {
	// Today header with date
	now := time.Now()
	dateHeader := fmt.Sprintf("Today (%s):", now.Format("Mon, 2 Jan"))
	tm.titleItem = systray.AddMenuItem(dateHeader, "Today's meetings")
	tm.titleItem.Disable()
	
	systray.AddSeparator()
	
	// Pre-create meeting slots to maintain proper order
	tm.maxMeetingSlots = 10 // Allow up to 10 meetings to be displayed
	tm.meetingSlots = make([]*systray.MenuItem, tm.maxMeetingSlots)
	for i := 0; i < tm.maxMeetingSlots; i++ {
		item := systray.AddMenuItem("", "")
		item.Hide() // Hide by default
		tm.meetingSlots[i] = item
	}
	
	// Create static menu items in correct order
	systray.AddSeparator()
	
	tm.quickActionsHeader = systray.AddMenuItem("Quick Actions", "")
	tm.quickActionsHeader.Disable()
	
	tm.createItem = systray.AddMenuItem("➕ Create meeting", "Create a new meeting")
	tm.refreshItem = systray.AddMenuItem("🔄 Refresh", "Refresh calendar data")
	tm.settingsItem = systray.AddMenuItem("⚙️ Settings", "Open settings")
	tm.rateItem = systray.AddMenuItem("⭐ Rate MeetingBar", "Help us improve by rating the app")
	
	systray.AddSeparator()
	
	tm.quitItem = systray.AddMenuItem("Quit MeetingBar", "Quit MeetingBar")
}


func (tm *TrayManager) createMeeting() {
	// Open Google Calendar create meeting URL
	createMeetingURL := "https://calendar.google.com/calendar/u/0/r/eventedit"
	err := exec.Command("xdg-open", createMeetingURL).Start()
	if err != nil {
		log.Printf("Failed to open create meeting URL: %v", err)
	}
}

func (tm *TrayManager) handleMenuClicks() {
	for {
		// Check if menu items exist before trying to read from their channels
		if tm.createItem != nil && tm.refreshItem != nil && tm.settingsItem != nil && tm.rateItem != nil && tm.quitItem != nil {
			select {
			case <-tm.createItem.ClickedCh:
				tm.createMeeting()
				
			case <-tm.refreshItem.ClickedCh:
				go tm.refreshMeetings()
				
			case <-tm.settingsItem.ClickedCh:
				go tm.openSettings()
				
			case <-tm.rateItem.ClickedCh:
				// Open GitHub repo for feedback
				exec.Command("xdg-open", "https://github.com/your-repo/meetingbar").Start()
				
			case <-tm.quitItem.ClickedCh:
				systray.Quit()
				return
				
			case <-tm.ctx.Done():
				return
			}
		} else {
			// Wait a bit before checking again if menu items aren't ready
			select {
			case <-tm.ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}
	}
}

func (tm *TrayManager) startPeriodicRefresh() {
	tm.ticker = time.NewTicker(tm.config.GetRefreshDuration())
	
	go func() {
		for {
			select {
			case <-tm.ticker.C:
				tm.refreshMeetings()
			case <-tm.ctx.Done():
				return
			}
		}
	}()
}

func (tm *TrayManager) refreshMeetings() {
	// Check backend requirements
	if tm.calendarService.RequiresAuthentication() && len(tm.config.Accounts) == 0 {
		tm.updateTrayForNoAccounts()
		return
	}
	
	var allMeetings []calendar.Meeting
	
	if tm.calendarService.IsGnomeBackend() {
		// For GNOME backend, we don't use accounts - get meetings directly
		var enabledCalendars []string
		if len(tm.config.EnabledCalendars) == 0 {
			// Get all available calendars if none specifically enabled
			calendars, err := tm.calendarService.GetCalendars("")
			if err != nil {
				log.Printf("Failed to get GNOME calendars: %v", err)
				tm.updateTrayForNoAccounts()
				return
			}
			for _, cal := range calendars {
				if cal.Enabled {
					enabledCalendars = append(enabledCalendars, cal.ID)
				}
			}
		} else {
			enabledCalendars = tm.config.EnabledCalendars
		}
		
		meetings, err := tm.calendarService.GetMeetings("", enabledCalendars)
		if err != nil {
			log.Printf("Failed to get meetings from GNOME Calendar: %v", err)
			tm.updateTrayForNoAccounts()
			return
		}
		allMeetings = meetings
	} else {
		// For Google backend, iterate through accounts
		for _, account := range tm.config.Accounts {
			// Get enabled calendars for this account
			var enabledCalendars []string
			
			// If no calendars are specifically enabled, try to get all calendars
			if len(tm.config.EnabledCalendars) == 0 {
				calendars, err := tm.calendarService.GetCalendars(account.ID)
				if err != nil {
					log.Printf("Failed to get calendars for account %s: %v", account.Email, err)
					continue
				}
				for _, cal := range calendars {
					enabledCalendars = append(enabledCalendars, cal.ID)
				}
			} else {
				enabledCalendars = tm.config.EnabledCalendars
			}
			
			meetings, err := tm.calendarService.GetMeetings(account.ID, enabledCalendars)
			if err != nil {
				log.Printf("Failed to get meetings for account %s: %v", account.Email, err)
				continue
			}
			
			allMeetings = append(allMeetings, meetings...)
		}
	}
	
	// Sort meetings by start time
	sort.Slice(allMeetings, func(i, j int) bool {
		return allMeetings[i].StartTime.Before(allMeetings[j].StartTime)
	})
	
	tm.meetings = allMeetings
	tm.notificationMgr.UpdateMeetings(allMeetings)
	tm.updateTrayDisplay()
}

func (tm *TrayManager) updateTrayDisplay() {
	now := time.Now()
	
	// Hide all meeting slots first
	for _, slot := range tm.meetingSlots {
		slot.Hide()
	}
	
	if len(tm.meetings) == 0 {
		tm.updateTrayForNoMeetings()
		tm.displayNoMeetingsInSlots()
		return
	}
	
	// Find current and upcoming meetings
	var currentMeeting *calendar.Meeting
	var upcomingMeetings []calendar.Meeting
	
	for i := range tm.meetings {
		meeting := &tm.meetings[i]
		if now.After(meeting.StartTime) && now.Before(meeting.EndTime) {
			currentMeeting = meeting
		} else if now.Before(meeting.StartTime) {
			upcomingMeetings = append(upcomingMeetings, *meeting)
		}
	}
	
	// Update tray title and tooltip
	if currentMeeting != nil {
		tm.updateTrayForCurrentMeeting(currentMeeting)
	} else if len(upcomingMeetings) > 0 {
		tm.updateTrayForUpcomingMeeting(&upcomingMeetings[0])
	} else {
		tm.updateTrayForNoMeetings()
	}
	
	// Display meetings in pre-allocated slots
	tm.displayMeetingsInSlots(currentMeeting, upcomingMeetings, now)
}

func (tm *TrayManager) displayNoMeetingsInSlots() {
	// Use first slot to show no meetings message
	if len(tm.meetingSlots) > 0 {
		tm.meetingSlots[0].SetTitle("🌅    No meetings today")
		tm.meetingSlots[0].SetTooltip("Enjoy your free time!")
		tm.meetingSlots[0].Disable()
		tm.meetingSlots[0].Show()
	}
	
	// Use second slot for helpful info
	if len(tm.meetingSlots) > 1 {
		tm.meetingSlots[1].SetTitle("ℹ️    Refresh to check for new meetings")
		tm.meetingSlots[1].SetTooltip("Click refresh or wait for automatic update")
		tm.meetingSlots[1].Disable()
		tm.meetingSlots[1].Show()
	}
}

func (tm *TrayManager) displayMeetingsInSlots(currentMeeting *calendar.Meeting, upcomingMeetings []calendar.Meeting, now time.Time) {
	slotIndex := 0
	
	// Display current meeting first
	if currentMeeting != nil && slotIndex < len(tm.meetingSlots) {
		timeLeft := currentMeeting.EndTime.Sub(now)
		startTime := currentMeeting.StartTime.Format("15:04")
		endTime := currentMeeting.EndTime.Format("15:04")
		
		title := fmt.Sprintf("🔴 %s    %s    %s",
			startTime,
			endTime,
			tm.truncateTitle(currentMeeting.Title))
		
		tooltip := fmt.Sprintf("🔴 LIVE NOW: %s\n⏰ Started: %s\n⏱ Ends: %s\n⌛ %s remaining", 
			currentMeeting.Title,
			startTime,
			endTime,
			formatDuration(timeLeft))
		
		// Add meeting location if available
		if currentMeeting.MeetingLink != nil {
			tooltip += fmt.Sprintf("\n🔗 %s meeting", currentMeeting.MeetingLink.Type)
		}
		
		tm.meetingSlots[slotIndex].SetTitle(title)
		tm.meetingSlots[slotIndex].SetTooltip(tooltip)
		tm.meetingSlots[slotIndex].Enable()
		tm.meetingSlots[slotIndex].Show()
		
		// Set up click handler for this slot
		go tm.handleMeetingSlotClick(tm.meetingSlots[slotIndex], currentMeeting)
		
		slotIndex++
	}
	
	// Display upcoming meetings
	maxMeetings := tm.config.MaxMeetings
	if maxMeetings <= 0 {
		maxMeetings = 5
	}
	
	displayMeetings := upcomingMeetings
	if len(displayMeetings) > maxMeetings {
		displayMeetings = displayMeetings[:maxMeetings]
	}
	
	for _, meeting := range displayMeetings {
		if slotIndex >= len(tm.meetingSlots) {
			break // No more slots available
		}
		
		timeUntil := meeting.StartTime.Sub(now)
		startTime := meeting.StartTime.Format("15:04")
		endTime := meeting.EndTime.Format("15:04")
		duration := meeting.EndTime.Sub(meeting.StartTime)
		
		// Meeting link indicator
		linkIcon := "🟢" // Green dot for meetings with video links
		if meeting.MeetingLink == nil {
			linkIcon = "⚪️" // White dot for meetings without links
		}
		
		var prefix string
		if timeUntil < time.Minute {
			prefix = "🔴" // Red indicator for starting now
		} else if timeUntil < 5*time.Minute {
			prefix = "🟡" // Yellow indicator for very soon
		} else {
			prefix = linkIcon // Use link indicator for normal meetings
		}
		
		title := fmt.Sprintf("%s %s    %s    %s",
			prefix,
			startTime,
			endTime,
			tm.truncateTitle(meeting.Title))
		
		tooltip := fmt.Sprintf("%s\n⏰ %s - %s (Duration: %s)\n🕒 Starts in %s", 
			meeting.Title,
			startTime,
			endTime,
			formatDuration(duration),
			formatDuration(timeUntil))
		
		// Add meeting location if available
		if meeting.MeetingLink != nil {
			tooltip += fmt.Sprintf("\n🔗 %s", meeting.MeetingLink.Type)
		}
		
		tm.meetingSlots[slotIndex].SetTitle(title)
		tm.meetingSlots[slotIndex].SetTooltip(tooltip)
		tm.meetingSlots[slotIndex].Enable()
		tm.meetingSlots[slotIndex].Show()
		
		// Set up click handler for this slot
		meetingCopy := meeting // Create a copy for the closure
		go tm.handleMeetingSlotClick(tm.meetingSlots[slotIndex], &meetingCopy)
		
		slotIndex++
	}
	
	// Show "more meetings" if truncated
	if len(upcomingMeetings) > maxMeetings && slotIndex < len(tm.meetingSlots) {
		tm.meetingSlots[slotIndex].SetTitle(fmt.Sprintf("…    and %d more meetings", len(upcomingMeetings)-maxMeetings))
		tm.meetingSlots[slotIndex].SetTooltip("Configure max meetings in settings")
		tm.meetingSlots[slotIndex].Disable()
		tm.meetingSlots[slotIndex].Show()
	}
}

func (tm *TrayManager) handleMeetingSlotClick(slot *systray.MenuItem, meeting *calendar.Meeting) {
	for {
		select {
		case <-slot.ClickedCh:
			tm.joinMeeting(meeting)
		case <-tm.ctx.Done():
			return
		}
	}
}




func (tm *TrayManager) joinMeeting(meeting *calendar.Meeting) {
	if meeting.MeetingLink == nil {
		log.Printf("No meeting link found for: %s", meeting.Title)
		return
	}
	
	// Open meeting URL in default browser
	err := exec.Command("xdg-open", meeting.MeetingLink.URL).Start()
	if err != nil {
		log.Printf("Failed to open meeting URL: %v", err)
	}
}

func (tm *TrayManager) updateTrayForNoAccounts() {
	systray.SetTitle("MeetingBar")
	systray.SetTooltip("MeetingBar - No accounts configured")
	
	// Hide all meeting slots first
	for _, slot := range tm.meetingSlots {
		slot.Hide()
	}
	
	// Use first slot to show no accounts message
	if len(tm.meetingSlots) > 0 {
		tm.meetingSlots[0].SetTitle("⚠️ No accounts configured")
		tm.meetingSlots[0].SetTooltip("Add a Google account in settings")
		tm.meetingSlots[0].Disable()
		tm.meetingSlots[0].Show()
	}
	
	// Use second slot for setup link
	if len(tm.meetingSlots) > 1 {
		tm.meetingSlots[1].SetTitle("⚙️ Open Settings to Add Account")
		tm.meetingSlots[1].SetTooltip("Configure your Google account")
		tm.meetingSlots[1].Enable()
		tm.meetingSlots[1].Show()
		
		// Set up click handler for settings
		go func() {
			for {
				select {
				case <-tm.meetingSlots[1].ClickedCh:
					go tm.openSettings()
				case <-tm.ctx.Done():
					return
				}
			}
		}()
	}
}

func (tm *TrayManager) updateTrayForNoMeetings() {
	systray.SetTitle("MeetingBar")
	systray.SetTooltip("MeetingBar - No meetings today")
}

func (tm *TrayManager) updateTrayForCurrentMeeting(meeting *calendar.Meeting) {
	now := time.Now()
	timeLeft := meeting.EndTime.Sub(now)
	
	// Use customizable format
	title := tm.formatMeetingDisplay(tm.config.CurrentMeetingFormat, meeting, timeLeft, true)
	systray.SetTitle(title)
	systray.SetTooltip(fmt.Sprintf("Currently in meeting: %s\nEnds at %s (%s remaining)", 
		meeting.Title, 
		meeting.EndTime.Format("15:04"), 
		formatDuration(timeLeft)))
	tm.titleItem.SetTitle(fmt.Sprintf("▶ %s", tm.truncateTitle(meeting.Title)))
}

func (tm *TrayManager) updateTrayForUpcomingMeeting(meeting *calendar.Meeting) {
	now := time.Now()
	timeUntil := meeting.StartTime.Sub(now)
	
	var title string
	if timeUntil < time.Minute {
		title = fmt.Sprintf("%s starting now", tm.truncateTitle(meeting.Title))
	} else {
		// Use customizable format
		title = tm.formatMeetingDisplay(tm.config.UpcomingMeetingFormat, meeting, timeUntil, false)
	}
	
	systray.SetTitle(title)
	systray.SetTooltip(fmt.Sprintf("Next meeting: %s\nStarts at %s (in %s)", 
		meeting.Title, 
		meeting.StartTime.Format("15:04"), 
		formatDuration(timeUntil)))
	tm.titleItem.SetTitle(fmt.Sprintf("Next: %s", tm.truncateTitle(meeting.Title)))
}

func (tm *TrayManager) truncateTitle(title string) string {
	maxLength := tm.config.MaxTitleLength
	if maxLength <= 0 {
		maxLength = 25 // fallback to default
	}
	if len(title) > maxLength {
		if maxLength <= 3 {
			return title[:maxLength]
		}
		return title[:maxLength-3] + "..."
	}
	return title
}

// formatDuration formats a duration into a human-readable string like "1h 20m" or "5m"
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0m"
	}
	
	totalMinutes := int(d.Minutes())
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	if minutes <= 0 {
		return "<1m"
	}
	return fmt.Sprintf("%dm", minutes)
}

// formatMeetingDisplay formats meeting display text using template strings
func (tm *TrayManager) formatMeetingDisplay(template string, meeting *calendar.Meeting, timeValue time.Duration, isTimeLeft bool) string {
	title := tm.truncateTitle(meeting.Title)
	timeStr := formatDuration(timeValue)
	
	// Replace template variables
	result := template
	result = strings.ReplaceAll(result, "{title}", title)
	if isTimeLeft {
		result = strings.ReplaceAll(result, "{time_left}", timeStr)
	} else {
		result = strings.ReplaceAll(result, "{time_until}", timeStr)
	}
	result = strings.ReplaceAll(result, "{start_time}", meeting.StartTime.Format("15:04"))
	result = strings.ReplaceAll(result, "{end_time}", meeting.EndTime.Format("15:04"))
	
	return result
}

func (tm *TrayManager) openSettings() {
	go func() {
		if err := tm.settingsMgr.ShowSettings(); err != nil {
			log.Printf("Settings error: %v", err)
		}
		// Refresh meetings after settings might have changed
		tm.refreshMeetings()
	}()
}

func (tm *TrayManager) cleanup() {
	if tm.ticker != nil {
		tm.ticker.Stop()
	}
	if tm.cancel != nil {
		tm.cancel()
	}
}

// Calendar icon - a simple 16x16 PNG calendar icon
func getDefaultIcon() []byte {
	// Simple calendar icon with blue header and grid pattern
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10, 0x08, 0x03, 0x00, 0x00, 0x00, 0x28, 0x2D, 0x0F,
		0x53, 0x00, 0x00, 0x00, 0x2A, 0x50, 0x4C, 0x54, 0x45, 0xFF, 0xFF, 0xFF, 0x3B, 0x82, 0xF6, 0x60,
		0x9C, 0xF8, 0x93, 0xC5, 0xFD, 0xDB, 0xEA, 0xFE, 0xE5, 0xF3, 0xFF, 0xC0, 0xC0, 0xC0, 0x80, 0x80,
		0x80, 0x40, 0x40, 0x40, 0x00, 0x00, 0x00, 0xF0, 0xF0, 0xF0, 0xD0, 0xD0, 0xD0, 0xA0, 0xA0, 0xA0,
		0x70, 0x70, 0x70, 0x20, 0x20, 0x20, 0x99, 0xF5, 0x2C, 0xA2, 0x00, 0x00, 0x00, 0x74, 0x49, 0x44,
		0x41, 0x54, 0x18, 0x19, 0x63, 0x60, 0x00, 0x82, 0x46, 0x26, 0x06, 0x06, 0x86, 0x26, 0x66, 0x16,
		0x76, 0x0E, 0x4E, 0x2E, 0x6E, 0x1E, 0x5E, 0x3E, 0x7E, 0x01, 0x41, 0x21, 0x61, 0x11, 0x51, 0x31,
		0x71, 0x09, 0x49, 0x29, 0x69, 0x19, 0x59, 0x39, 0x79, 0x05, 0x45, 0x25, 0x65, 0x15, 0x55, 0x35,
		0x75, 0x0D, 0x4D, 0x2D, 0x6D, 0x1D, 0x5D, 0x3D, 0x7D, 0x03, 0x43, 0x23, 0x63, 0x13, 0x53, 0x33,
		0x73, 0x0B, 0x4B, 0x2B, 0x6B, 0x1B, 0x5B, 0x3B, 0x7B, 0x07, 0x47, 0x27, 0x67, 0x17, 0x57, 0x37,
		0x77, 0x0F, 0x4F, 0x2F, 0x6F, 0x1F, 0x5F, 0x3F, 0x7F, 0x80, 0x80, 0x40, 0xC0, 0x20, 0xA0, 0x60,
		0xE0, 0x10, 0x90, 0x50, 0xD0, 0x30, 0xB0, 0x70, 0xF0, 0x08, 0x88, 0x48, 0xC8, 0x28, 0xA8, 0x68,
		0xE8, 0x18, 0x98, 0x58, 0xD8, 0x38, 0xB8, 0x78, 0xF8, 0x04, 0x84, 0x44, 0xC4, 0x24, 0xA4, 0x64,
		0xE4, 0x14, 0x94, 0x54, 0xD4, 0x34, 0xB4, 0x74, 0xF4, 0x0C, 0x8C, 0x00, 0x00, 0x19, 0x0A, 0x0E,
		0x3F, 0x92, 0x38, 0x04, 0xE9, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60,
		0x82,
	}
}