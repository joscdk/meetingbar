package calendar

import (
	"context"
	"fmt"
	"log"

	"meetingbar/config"
)

// CalendarService defines the interface for calendar backends
type CalendarService interface {
	GetMeetings(accountID string, enabledCalendars []string) ([]Meeting, error)
	GetCalendars(accountID string) ([]config.Calendar, error)
}

// UnifiedCalendarService manages multiple calendar backends
type UnifiedCalendarService struct {
	ctx              context.Context
	config           *config.Config
	googleService    *GoogleCalendarService
	gnomeService     *GnomeCalendarService
}

// NewUnifiedCalendarService creates a new unified calendar service
func NewUnifiedCalendarService(ctx context.Context, cfg *config.Config) *UnifiedCalendarService {
	return &UnifiedCalendarService{
		ctx:           ctx,
		config:        cfg,
		googleService: NewGoogleCalendarService(ctx),
		gnomeService:  NewGnomeCalendarService(ctx),
	}
}

// GetMeetings retrieves meetings from the configured backend
func (u *UnifiedCalendarService) GetMeetings(accountID string, enabledCalendars []string) ([]Meeting, error) {
	switch u.config.CalendarBackend {
	case "google":
		return u.googleService.GetMeetings(accountID, enabledCalendars)
	case "gnome":
		// For GNOME, we use calendar IDs directly instead of account-based lookup
		if len(enabledCalendars) == 0 {
			// Get all available calendars if none specified
			calendars, err := u.GetGnomeCalendars()
			if err != nil {
				return nil, fmt.Errorf("failed to get GNOME calendars: %w", err)
			}
			var calendarIDs []string
			for _, cal := range calendars {
				if cal.Enabled {
					calendarIDs = append(calendarIDs, cal.ID)
				}
			}
			return u.gnomeService.GetMeetings(calendarIDs)
		}
		return u.gnomeService.GetMeetings(enabledCalendars)
	default:
		return nil, fmt.Errorf("unsupported calendar backend: %s", u.config.CalendarBackend)
	}
}

// GetCalendars retrieves available calendars from the configured backend
func (u *UnifiedCalendarService) GetCalendars(accountID string) ([]config.Calendar, error) {
	switch u.config.CalendarBackend {
	case "google":
		return u.googleService.GetCalendars(accountID)
	case "gnome":
		return u.GetGnomeCalendars()
	default:
		return nil, fmt.Errorf("unsupported calendar backend: %s", u.config.CalendarBackend)
	}
}

// GetGnomeCalendars retrieves calendars from GNOME and converts to common format
func (u *UnifiedCalendarService) GetGnomeCalendars() ([]config.Calendar, error) {
	gnomeCalendars, err := u.gnomeService.GetCalendars()
	if err != nil {
		return nil, err
	}

	var calendars []config.Calendar
	for _, gnomeCal := range gnomeCalendars {
		calendars = append(calendars, config.Calendar{
			ID:        gnomeCal.ID,
			Name:      gnomeCal.DisplayName,
			AccountID: "gnome", // Use a fixed account ID for GNOME calendars
			Enabled:   gnomeCal.Enabled,
			Color:     gnomeCal.Color,
		})
	}

	return calendars, nil
}

// IsGoogleBackend returns true if using Google Calendar backend
func (u *UnifiedCalendarService) IsGoogleBackend() bool {
	return u.config.CalendarBackend == "google"
}

// IsGnomeBackend returns true if using GNOME Calendar backend
func (u *UnifiedCalendarService) IsGnomeBackend() bool {
	return u.config.CalendarBackend == "gnome"
}

// RequiresAuthentication returns true if the backend requires OAuth authentication
func (u *UnifiedCalendarService) RequiresAuthentication() bool {
	return u.config.CalendarBackend == "google"
}

// GetBackendName returns the human-readable name of the current backend
func (u *UnifiedCalendarService) GetBackendName() string {
	switch u.config.CalendarBackend {
	case "google":
		return "Google Calendar"
	case "gnome":
		return "GNOME Calendar"
	default:
		return "Unknown"
	}
}

// TestConnection tests the connection to the configured backend
func (u *UnifiedCalendarService) TestConnection() error {
	switch u.config.CalendarBackend {
	case "google":
		// For Google, we need at least one account configured
		if len(u.config.Accounts) == 0 {
			return fmt.Errorf("no Google accounts configured")
		}
		// Could add more specific Google API connectivity test here
		return nil
	case "gnome":
		// Test D-Bus connection to Evolution Data Server
		if err := u.gnomeService.Connect(); err != nil {
			return fmt.Errorf("failed to connect to GNOME Calendar: %w", err)
		}
		
		// Test if we can list calendars
		_, err := u.gnomeService.GetCalendars()
		if err != nil {
			return fmt.Errorf("failed to access GNOME calendars: %w", err)
		}
		
		return nil
	default:
		return fmt.Errorf("unsupported calendar backend: %s", u.config.CalendarBackend)
	}
}

// GetAuthURL returns OAuth2 authorization URL (Google backend only)
func (u *UnifiedCalendarService) GetAuthURL() (string, error) {
	if u.config.CalendarBackend != "google" {
		return "", fmt.Errorf("GetAuthURL is only available for Google Calendar backend")
	}
	return u.googleService.GetAuthURL()
}

// RemoveAccount removes an account (Google backend only)
func (u *UnifiedCalendarService) RemoveAccount(accountID string) error {
	if u.config.CalendarBackend != "google" {
		return fmt.Errorf("RemoveAccount is only available for Google Calendar backend")
	}
	return u.googleService.RemoveAccount(accountID)
}

// Close closes connections to all backends
func (u *UnifiedCalendarService) Close() error {
	var lastErr error
	
	if u.gnomeService != nil {
		if err := u.gnomeService.Close(); err != nil {
			log.Printf("Failed to close GNOME calendar service: %v", err)
			lastErr = err
		}
	}
	
	// Google service doesn't need explicit closing
	
	return lastErr
}