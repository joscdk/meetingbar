package ui

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"time"

	"meetingbar/calendar"
	"meetingbar/config"

	"github.com/getlantern/systray"
)

type TrayManager struct {
	config          *config.Config
	calendarService *calendar.GoogleCalendarService
	meetings        []calendar.Meeting
	ticker          *time.Ticker
	ctx             context.Context
	cancel          context.CancelFunc
	notificationMgr *NotificationManager
	settingsMgr     *AdvancedSettingsManager
	
	// Menu items
	titleItem        *systray.MenuItem
	meetingItems     []*systray.MenuItem
	refreshItem      *systray.MenuItem
	settingsItem     *systray.MenuItem
	quitItem         *systray.MenuItem
}

var trayManager *TrayManager

func OnReady(cfg *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	
	trayManager = &TrayManager{
		config:          cfg,
		calendarService: calendar.NewGoogleCalendarService(ctx),
		ctx:             ctx,
		cancel:          cancel,
		notificationMgr: NewNotificationManager(cfg),
		settingsMgr:     NewAdvancedSettingsManager(cfg, ctx),
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
	// Set tray icon - for now, use a simple icon
	systray.SetIcon(getDefaultIcon())
	systray.SetTitle("MeetingBar")
	systray.SetTooltip("MeetingBar - No meetings")
	
	// Create menu structure
	tm.titleItem = systray.AddMenuItem("No meetings today", "No upcoming meetings")
	tm.titleItem.Disable()
	
	systray.AddSeparator()
	
	tm.refreshItem = systray.AddMenuItem("Refresh", "Refresh calendar data")
	tm.settingsItem = systray.AddMenuItem("Settings", "Open settings")
	
	systray.AddSeparator()
	
	tm.quitItem = systray.AddMenuItem("Quit", "Quit MeetingBar")
	
	// Handle menu clicks
	go tm.handleMenuClicks()
}

func (tm *TrayManager) handleMenuClicks() {
	for {
		select {
		case <-tm.refreshItem.ClickedCh:
			go tm.refreshMeetings()
			
		case <-tm.settingsItem.ClickedCh:
			go tm.openSettings()
			
		case <-tm.quitItem.ClickedCh:
			systray.Quit()
			return
			
		case <-tm.ctx.Done():
			return
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
	if len(tm.config.Accounts) == 0 {
		tm.updateTrayForNoAccounts()
		return
	}
	
	var allMeetings []calendar.Meeting
	
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
	
	// Remove old meeting menu items
	for _, item := range tm.meetingItems {
		item.Hide()
	}
	tm.meetingItems = nil
	
	if len(tm.meetings) == 0 {
		tm.updateTrayForNoMeetings()
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
	
	// Add meeting menu items (limit to 5)
	displayMeetings := upcomingMeetings
	if len(displayMeetings) > 5 {
		displayMeetings = displayMeetings[:5]
	}
	
	if currentMeeting != nil {
		// Add current meeting at the top
		item := systray.AddMenuItemCheckbox(
			fmt.Sprintf("▶ %s (Now)", tm.truncateTitle(currentMeeting.Title)),
			fmt.Sprintf("Currently in meeting: %s", currentMeeting.Title),
			false,
		)
		tm.meetingItems = append(tm.meetingItems, item)
		
		// Handle clicks for current meeting
		go tm.handleMeetingClick(item, currentMeeting)
	}
	
	for i := range displayMeetings {
		meeting := &displayMeetings[i]
		timeUntil := meeting.StartTime.Sub(now)
		
		var title string
		if timeUntil < time.Minute {
			title = fmt.Sprintf("📅 %s (Starting now)", tm.truncateTitle(meeting.Title))
		} else if timeUntil < time.Hour {
			minutes := int(timeUntil.Minutes())
			title = fmt.Sprintf("📅 %s (%dm)", tm.truncateTitle(meeting.Title), minutes)
		} else {
			title = fmt.Sprintf("📅 %s (%s)", tm.truncateTitle(meeting.Title), meeting.StartTime.Format("15:04"))
		}
		
		tooltip := fmt.Sprintf("%s\n%s - %s", 
			meeting.Title,
			meeting.StartTime.Format("15:04"),
			meeting.EndTime.Format("15:04"))
		
		item := systray.AddMenuItem(title, tooltip)
		tm.meetingItems = append(tm.meetingItems, item)
		
		// Handle clicks
		go tm.handleMeetingClick(item, meeting)
	}
}

func (tm *TrayManager) handleMeetingClick(item *systray.MenuItem, meeting *calendar.Meeting) {
	for {
		select {
		case <-item.ClickedCh:
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
	tm.titleItem.SetTitle("No accounts configured")
}

func (tm *TrayManager) updateTrayForNoMeetings() {
	systray.SetTitle("MeetingBar")
	systray.SetTooltip("MeetingBar - No meetings today")
	tm.titleItem.SetTitle("No meetings today")
}

func (tm *TrayManager) updateTrayForCurrentMeeting(meeting *calendar.Meeting) {
	title := fmt.Sprintf("In: %s", tm.truncateTitle(meeting.Title))
	systray.SetTitle(title)
	systray.SetTooltip(fmt.Sprintf("Currently in meeting: %s", meeting.Title))
	tm.titleItem.SetTitle(fmt.Sprintf("▶ Now: %s", meeting.Title))
}

func (tm *TrayManager) updateTrayForUpcomingMeeting(meeting *calendar.Meeting) {
	now := time.Now()
	timeUntil := meeting.StartTime.Sub(now)
	
	var title string
	if timeUntil < time.Minute {
		title = fmt.Sprintf("%s (Now)", tm.truncateTitle(meeting.Title))
	} else if timeUntil < time.Hour {
		minutes := int(timeUntil.Minutes())
		title = fmt.Sprintf("%s (%dm)", tm.truncateTitle(meeting.Title), minutes)
	} else {
		title = fmt.Sprintf("%s (%s)", tm.truncateTitle(meeting.Title), meeting.StartTime.Format("15:04"))
	}
	
	systray.SetTitle(title)
	systray.SetTooltip(fmt.Sprintf("Next meeting: %s at %s", 
		meeting.Title, 
		meeting.StartTime.Format("15:04")))
	tm.titleItem.SetTitle(fmt.Sprintf("Next: %s", title))
}

func (tm *TrayManager) truncateTitle(title string) string {
	if len(title) > 25 {
		return title[:22] + "..."
	}
	return title
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

// Simple default icon - a minimal 16x16 PNG icon
func getDefaultIcon() []byte {
	// Valid minimal PNG (16x16 black square)
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10, 0x01, 0x00, 0x00, 0x00, 0x00, 0x37, 0x6e, 0xf9,
		0x24, 0x00, 0x00, 0x00, 0x04, 0x67, 0x41, 0x4d, 0x41, 0x00, 0x00, 0xb1, 0x8f, 0x0b, 0xfc, 0x61,
		0x05, 0x00, 0x00, 0x00, 0x20, 0x63, 0x48, 0x52, 0x4d, 0x00, 0x00, 0x7a, 0x26, 0x00, 0x00, 0x80,
		0x84, 0x00, 0x00, 0xfa, 0x00, 0x00, 0x00, 0x80, 0xe8, 0x00, 0x00, 0x75, 0x30, 0x00, 0x00, 0xea,
		0x60, 0x00, 0x00, 0x3a, 0x98, 0x00, 0x00, 0x17, 0x70, 0x9c, 0xba, 0x51, 0x3c, 0x00, 0x00, 0x00,
		0x06, 0x50, 0x4c, 0x54, 0x45, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xa5, 0xd9, 0x9f, 0xdd, 0x00,
		0x00, 0x00, 0x09, 0x70, 0x48, 0x59, 0x73, 0x00, 0x00, 0x0b, 0x13, 0x00, 0x00, 0x0b, 0x13, 0x01,
		0x00, 0x9a, 0x9c, 0x18, 0x00, 0x00, 0x00, 0x1a, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0x60,
		0x00, 0x02, 0x96, 0x00, 0x05, 0x18, 0x30, 0x60, 0xc0, 0x80, 0x01, 0x03, 0x06, 0x0c, 0x18, 0x30,
		0x60, 0xc0, 0x80, 0x01, 0x03, 0x06, 0x00, 0x00, 0x01, 0x68, 0x00, 0x0a, 0xac, 0x4f, 0xfc, 0x17,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}