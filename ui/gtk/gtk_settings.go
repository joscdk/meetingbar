package gtk

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"meetingbar/calendar"
	"meetingbar/config"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type GTKSettingsManager struct {
	config          *config.Config
	calendarService *calendar.UnifiedCalendarService
	ctx             context.Context
	onRefreshCallback func()
	app             *gtk.Application
}

func NewGTKSettingsManager(cfg *config.Config, ctx context.Context, onRefresh func()) *GTKSettingsManager {
	return &GTKSettingsManager{
		config:            cfg,
		calendarService:   calendar.NewUnifiedCalendarService(ctx, cfg),
		ctx:               ctx,
		onRefreshCallback: onRefresh,
	}
}

func (gsm *GTKSettingsManager) ShowSettings() error {
	// Create GTK application
	gsm.app = gtk.NewApplication("com.meetingbar.settings", gio.ApplicationFlagsNone)
	
	gsm.app.ConnectActivate(func() {
		gsm.createMainWindow()
	})
	
	// Run the application in a separate goroutine
	go func() {
		gsm.app.Run(nil)
	}()
	
	return nil
}

func (gsm *GTKSettingsManager) ShowSettingsBlocking() error {
	// Create GTK application
	gsm.app = gtk.NewApplication("com.meetingbar.settings", gio.ApplicationFlagsNone)
	
	gsm.app.ConnectActivate(func() {
		gsm.createMainWindow()
	})
	
	// Run and block until application exits
	status := gsm.app.Run(nil)
	if status != 0 {
		return fmt.Errorf("GTK application exited with status %d", status)
	}
	
	return nil
}

func (gsm *GTKSettingsManager) createMainWindow() {
	// Create main window
	window := gtk.NewApplicationWindow(gsm.app)
	window.SetTitle("MeetingBar Settings")
	window.SetDefaultSize(750, 650)
	window.SetResizable(true)
	
	// Create notebook (tabs)
	notebook := gtk.NewNotebook()
	notebook.SetTabPos(gtk.PosTop)
	
	// Add tabs
	gsm.addOAuth2Tab(notebook)
	gsm.addBackendTab(notebook)
	gsm.addAccountsTab(notebook)
	gsm.addCalendarsTab(notebook)
	gsm.addNotificationsTab(notebook)
	gsm.addGeneralTab(notebook)
	
	// Create button box
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	buttonBox.SetHAlign(gtk.AlignEnd)
	buttonBox.SetMarginTop(10)
	buttonBox.SetMarginEnd(10)
	buttonBox.SetMarginBottom(10)
	
	// Cancel button
	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.ConnectClicked(func() {
		window.Close()
	})
	
	// Save button
	saveBtn := gtk.NewButtonWithLabel("Save & Close")
	saveBtn.AddCSSClass("suggested-action")
	saveBtn.ConnectClicked(func() {
		if err := gsm.config.Save(); err != nil {
			log.Printf("Failed to save config: %v", err)
			gsm.showErrorDialog(window, "Failed to save configuration", err.Error())
		} else {
			if gsm.onRefreshCallback != nil {
				gsm.onRefreshCallback()
			}
			window.Close()
		}
	})
	
	buttonBox.Append(cancelBtn)
	buttonBox.Append(saveBtn)
	
	// Main layout
	mainBox := gtk.NewBox(gtk.OrientationVertical, 0)
	mainBox.Append(notebook)
	mainBox.Append(buttonBox)
	
	window.SetChild(mainBox)
	window.Present()
}

func (gsm *GTKSettingsManager) addOAuth2Tab(notebook *gtk.Notebook) {
	// Create scrolled window
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	
	// Main container
	box := gtk.NewBox(gtk.OrientationVertical, 20)
	box.SetMarginTop(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(20)
	
	// Title and instructions
	titleLabel := gtk.NewLabel("Google OAuth2 Configuration")
	titleLabel.AddCSSClass("title-1")
	titleLabel.SetHAlign(gtk.AlignStart)
	
	instructionsLabel := gtk.NewLabel(`Set up Google OAuth2 credentials to access Google Calendar:

1. Go to Google Cloud Console (https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google Calendar API
4. Create OAuth 2.0 credentials (Desktop application)
5. Copy the Client ID and Client Secret below`)
	instructionsLabel.SetWrap(true)
	instructionsLabel.SetHAlign(gtk.AlignStart)
	
	// Status indicator
	statusText := "❌ Not configured"
	if gsm.config.OAuth2.ClientID != "" && gsm.config.OAuth2.ClientSecret != "" {
		statusText = "✅ Configured"
	}
	statusLabel := gtk.NewLabel(statusText)
	statusLabel.AddCSSClass("title-4")
	
	// Client ID entry
	clientIDLabel := gtk.NewLabel("Client ID:")
	clientIDLabel.SetHAlign(gtk.AlignStart)
	clientIDEntry := gtk.NewEntry()
	clientIDEntry.SetText(gsm.config.OAuth2.ClientID)
	clientIDEntry.SetPlaceholderText("Your Google OAuth2 Client ID")
	clientIDEntry.ConnectChanged(func() {
		gsm.config.OAuth2.ClientID = clientIDEntry.Text()
	})
	
	// Client Secret entry
	clientSecretLabel := gtk.NewLabel("Client Secret:")
	clientSecretLabel.SetHAlign(gtk.AlignStart)
	clientSecretEntry := gtk.NewPasswordEntry()
	clientSecretEntry.SetText(gsm.config.OAuth2.ClientSecret)
	// Note: PasswordEntry doesn't have SetPlaceholderText in GTK4
	clientSecretEntry.ConnectChanged(func() {
		gsm.config.OAuth2.ClientSecret = clientSecretEntry.Text()
	})
	
	// Add all elements to box
	box.Append(titleLabel)
	box.Append(instructionsLabel)
	box.Append(statusLabel)
	box.Append(gtk.NewSeparator(gtk.OrientationHorizontal))
	box.Append(clientIDLabel)
	box.Append(clientIDEntry)
	box.Append(clientSecretLabel)
	box.Append(clientSecretEntry)
	
	scrolled.SetChild(box)
	
	// Add tab to notebook
	tabLabel := gtk.NewLabel("🔐 OAuth2")
	notebook.AppendPage(scrolled, tabLabel)
}

func (gsm *GTKSettingsManager) addBackendTab(notebook *gtk.Notebook) {
	// Create scrolled window
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	
	// Main container
	box := gtk.NewBox(gtk.OrientationVertical, 20)
	box.SetMarginTop(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(20)
	
	// Title
	titleLabel := gtk.NewLabel("Calendar Backend Selection")
	titleLabel.AddCSSClass("title-1")
	titleLabel.SetHAlign(gtk.AlignStart)
	
	// Description
	descLabel := gtk.NewLabel(`Choose which calendar backend to use:

• Google: Use Google Calendar with OAuth2 authentication
• GNOME: Use GNOME Calendar (Evolution Data Server) - no authentication needed`)
	descLabel.SetWrap(true)
	descLabel.SetHAlign(gtk.AlignStart)
	
	// Radio buttons
	googleRadio := gtk.NewCheckButtonWithLabel("Google Calendar")
	gnomeRadio := gtk.NewCheckButtonWithLabel("GNOME Calendar (Evolution)")
	gnomeRadio.SetGroup(googleRadio)
	
	// Set initial state
	if gsm.config.CalendarBackend == "gnome" {
		gnomeRadio.SetActive(true)
	} else {
		googleRadio.SetActive(true)
	}
	
	// Connect signals
	googleRadio.ConnectToggled(func() {
		if googleRadio.Active() {
			gsm.config.CalendarBackend = "google"
		}
	})
	
	gnomeRadio.ConnectToggled(func() {
		if gnomeRadio.Active() {
			gsm.config.CalendarBackend = "gnome"
		}
	})
	
	// Add elements
	box.Append(titleLabel)
	box.Append(descLabel)
	box.Append(gtk.NewSeparator(gtk.OrientationHorizontal))
	box.Append(googleRadio)
	box.Append(gnomeRadio)
	
	scrolled.SetChild(box)
	
	// Add tab to notebook
	tabLabel := gtk.NewLabel("🔧 Backend")
	notebook.AppendPage(scrolled, tabLabel)
}

func (gsm *GTKSettingsManager) addAccountsTab(notebook *gtk.Notebook) {
	// Create placeholder for now
	box := gtk.NewBox(gtk.OrientationVertical, 20)
	box.SetMarginTop(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(20)
	
	titleLabel := gtk.NewLabel("Google Accounts")
	titleLabel.AddCSSClass("title-1")
	
	placeholderLabel := gtk.NewLabel("Account management will be implemented here.")
	
	box.Append(titleLabel)
	box.Append(placeholderLabel)
	
	tabLabel := gtk.NewLabel("👤 Accounts")
	notebook.AppendPage(box, tabLabel)
}

func (gsm *GTKSettingsManager) addCalendarsTab(notebook *gtk.Notebook) {
	// Create placeholder for now
	box := gtk.NewBox(gtk.OrientationVertical, 20)
	box.SetMarginTop(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(20)
	
	titleLabel := gtk.NewLabel("Calendar Selection")
	titleLabel.AddCSSClass("title-1")
	
	placeholderLabel := gtk.NewLabel("Calendar selection will be implemented here.")
	
	box.Append(titleLabel)
	box.Append(placeholderLabel)
	
	tabLabel := gtk.NewLabel("📅 Calendars")
	notebook.AppendPage(box, tabLabel)
}

func (gsm *GTKSettingsManager) addNotificationsTab(notebook *gtk.Notebook) {
	// Create scrolled window
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	
	// Main container
	box := gtk.NewBox(gtk.OrientationVertical, 20)
	box.SetMarginTop(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(20)
	
	// Title
	titleLabel := gtk.NewLabel("Notification Settings")
	titleLabel.AddCSSClass("title-1")
	titleLabel.SetHAlign(gtk.AlignStart)
	
	// Enable notifications
	enableNotificationsCheck := gtk.NewCheckButtonWithLabel("Enable notifications")
	enableNotificationsCheck.SetActive(gsm.config.EnableNotifications)
	enableNotificationsCheck.ConnectToggled(func() {
		gsm.config.EnableNotifications = enableNotificationsCheck.Active()
	})
	
	// Notification time
	notifTimeLabel := gtk.NewLabel("Minutes before meeting:")
	notifTimeEntry := gtk.NewEntry()
	notifTimeEntry.SetText(strconv.Itoa(gsm.config.NotificationTime))
	notifTimeEntry.ConnectChanged(func() {
		if val, err := strconv.Atoi(notifTimeEntry.Text()); err == nil {
			gsm.config.NotificationTime = val
		}
	})
	
	notifTimeBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	notifTimeBox.Append(notifTimeLabel)
	notifTimeBox.Append(notifTimeEntry)
	
	// Notification sound
	soundCheck := gtk.NewCheckButtonWithLabel("Play notification sound")
	soundCheck.SetActive(gsm.config.NotificationSound)
	soundCheck.ConnectToggled(func() {
		gsm.config.NotificationSound = soundCheck.Active()
	})
	
	// Persistent notifications
	persistentCheck := gtk.NewCheckButtonWithLabel("Persistent notifications")
	persistentCheck.SetActive(gsm.config.PersistentNotifications)
	persistentCheck.ConnectToggled(func() {
		gsm.config.PersistentNotifications = persistentCheck.Active()
	})
	
	// Add elements
	box.Append(titleLabel)
	box.Append(enableNotificationsCheck)
	box.Append(notifTimeBox)
	box.Append(soundCheck)
	box.Append(persistentCheck)
	
	scrolled.SetChild(box)
	
	// Add tab to notebook
	tabLabel := gtk.NewLabel("🔔 Notifications")
	notebook.AppendPage(scrolled, tabLabel)
}

func (gsm *GTKSettingsManager) addGeneralTab(notebook *gtk.Notebook) {
	// Create scrolled window
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	
	// Main container
	box := gtk.NewBox(gtk.OrientationVertical, 20)
	box.SetMarginTop(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginBottom(20)
	
	// Title
	titleLabel := gtk.NewLabel("General Settings")
	titleLabel.AddCSSClass("title-1")
	titleLabel.SetHAlign(gtk.AlignStart)
	
	// Refresh interval
	refreshLabel := gtk.NewLabel("Refresh interval (minutes):")
	refreshEntry := gtk.NewEntry()
	refreshEntry.SetText(strconv.Itoa(gsm.config.RefreshInterval))
	refreshEntry.ConnectChanged(func() {
		if val, err := strconv.Atoi(refreshEntry.Text()); err == nil {
			gsm.config.RefreshInterval = val
		}
	})
	
	refreshBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	refreshBox.Append(refreshLabel)
	refreshBox.Append(refreshEntry)
	
	// Max meetings
	maxMeetingsLabel := gtk.NewLabel("Max meetings to show:")
	maxMeetingsEntry := gtk.NewEntry()
	maxMeetingsEntry.SetText(strconv.Itoa(gsm.config.MaxMeetings))
	maxMeetingsEntry.ConnectChanged(func() {
		if val, err := strconv.Atoi(maxMeetingsEntry.Text()); err == nil {
			gsm.config.MaxMeetings = val
		}
	})
	
	maxMeetingsBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	maxMeetingsBox.Append(maxMeetingsLabel)
	maxMeetingsBox.Append(maxMeetingsEntry)
	
	// Show duration
	showDurationCheck := gtk.NewCheckButtonWithLabel("Show meeting duration")
	showDurationCheck.SetActive(gsm.config.ShowDuration)
	showDurationCheck.ConnectToggled(func() {
		gsm.config.ShowDuration = showDurationCheck.Active()
	})
	
	// Show meeting links
	showLinksCheck := gtk.NewCheckButtonWithLabel("Show meeting links")
	showLinksCheck.SetActive(gsm.config.ShowMeetingLinks)
	showLinksCheck.ConnectToggled(func() {
		gsm.config.ShowMeetingLinks = showLinksCheck.Active()
	})
	
	// Add elements
	box.Append(titleLabel)
	box.Append(refreshBox)
	box.Append(maxMeetingsBox)
	box.Append(showDurationCheck)
	box.Append(showLinksCheck)
	
	scrolled.SetChild(box)
	
	// Add tab to notebook
	tabLabel := gtk.NewLabel("⚙️ General")
	notebook.AppendPage(scrolled, tabLabel)
}

func (gsm *GTKSettingsManager) showErrorDialog(parent *gtk.ApplicationWindow, title, message string) {
	// Use MessageDialog for GTK4 compatibility
	dialog := gtk.NewMessageDialog(
		&parent.Window,
		gtk.DialogModal,
		gtk.MessageError,
		gtk.ButtonsClose,
	)
	dialog.SetMarkup(title + "\n\n" + message)
	dialog.ConnectResponse(func(responseID int) {
		dialog.Destroy()
	})
	dialog.Show()
}