package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"meetingbar/config"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Meeting struct {
	ID          string
	Title       string
	StartTime   time.Time
	EndTime     time.Time
	MeetingLink *MeetingLink
	CalendarID  string
	AccountID   string
	IsAllDay    bool
}

type GoogleCalendarService struct {
	ctx context.Context
}

func NewGoogleCalendarService(ctx context.Context) *GoogleCalendarService {
	return &GoogleCalendarService{ctx: ctx}
}

type CalendarInfo struct {
	ID              string `json:"id"`
	Summary         string `json:"summary"`
	Description     string `json:"description"`
	BackgroundColor string `json:"backgroundColor"`
}

func (g *GoogleCalendarService) GetCalendars(accountID string) ([]config.Calendar, error) {
	client, err := GetClientForAccount(g.ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for account: %w", err)
	}

	service, err := calendar.NewService(g.ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	calendarList, err := service.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve calendar list: %w", err)
	}

	var calendars []config.Calendar
	for _, item := range calendarList.Items {
		calendars = append(calendars, config.Calendar{
			ID:        item.Id,
			Name:      item.Summary,
			AccountID: accountID,
			Enabled:   true, // Google calendars are enabled by default
			Color:     item.BackgroundColor,
		})
	}

	return calendars, nil
}

func (g *GoogleCalendarService) GetMeetings(accountID string, enabledCalendars []string) ([]Meeting, error) {
	client, err := GetClientForAccount(g.ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for account: %w", err)
	}

	service, err := calendar.NewService(g.ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	var allMeetings []Meeting
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	for _, calendarID := range enabledCalendars {
		events, err := service.Events.List(calendarID).
			ShowDeleted(false).
			SingleEvents(true).
			TimeMin(now.Format(time.RFC3339)).
			TimeMax(tomorrow.Format(time.RFC3339)).
			OrderBy("startTime").
			Do()

		if err != nil {
			// Log error but continue with other calendars
			fmt.Printf("Warning: failed to get events for calendar %s: %v\n", calendarID, err)
			continue
		}

		for _, event := range events.Items {
			meeting := g.convertEventToMeeting(event, calendarID, accountID)
			if meeting != nil {
				allMeetings = append(allMeetings, *meeting)
			}
		}
	}

	return allMeetings, nil
}

func (g *GoogleCalendarService) convertEventToMeeting(event *calendar.Event, calendarID, accountID string) *Meeting {
	// Skip events without start time or cancelled events
	if event.Start == nil || event.Status == "cancelled" {
		return nil
	}

	// Parse times
	var startTime, endTime time.Time
	var isAllDay bool

	if event.Start.DateTime != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, event.Start.DateTime)
		if err != nil {
			fmt.Printf("Warning: failed to parse start time for event %s: %v\n", event.Id, err)
			return nil
		}
		
		if event.End != nil && event.End.DateTime != "" {
			endTime, err = time.Parse(time.RFC3339, event.End.DateTime)
			if err != nil {
				fmt.Printf("Warning: failed to parse end time for event %s: %v\n", event.Id, err)
				endTime = startTime.Add(time.Hour) // Default to 1 hour
			}
		} else {
			endTime = startTime.Add(time.Hour) // Default to 1 hour
		}
	} else if event.Start.Date != "" {
		// All-day event
		var err error
		startTime, err = time.Parse("2006-01-02", event.Start.Date)
		if err != nil {
			fmt.Printf("Warning: failed to parse start date for event %s: %v\n", event.Id, err)
			return nil
		}
		endTime = startTime.Add(24 * time.Hour)
		isAllDay = true
	} else {
		return nil
	}

	// Skip all-day events unless specifically handling them
	if isAllDay {
		return nil
	}

	// Extract meeting link
	var meetingLink *MeetingLink
	
	// Check for Google Meet conference data first
	if event.ConferenceData != nil && len(event.ConferenceData.EntryPoints) > 0 {
		for _, entryPoint := range event.ConferenceData.EntryPoints {
			if entryPoint.EntryPointType == "video" && strings.Contains(entryPoint.Uri, "meet.google.com") {
				meetingLink = &MeetingLink{
					URL:  entryPoint.Uri,
					Type: MeetingTypeGoogleMeet,
				}
				break
			}
		}
	}

	// If no conference data, parse description and location
	if meetingLink == nil {
		description := ""
		if event.Description != "" {
			description = event.Description
		}
		location := ""
		if event.Location != "" {
			location = event.Location
		}
		meetingLink = GetPrimaryMeetingLink(description, location)
	}

	title := event.Summary
	if title == "" {
		title = "(No title)"
	}

	return &Meeting{
		ID:          event.Id,
		Title:       title,
		StartTime:   startTime,
		EndTime:     endTime,
		MeetingLink: meetingLink,
		CalendarID:  calendarID,
		AccountID:   accountID,
		IsAllDay:    isAllDay,
	}
}

func (g *GoogleCalendarService) GetAccountEmail(accountID string) (string, error) {
	// This would require additional API call to get user info
	// For now, return empty string and handle in UI
	return "", nil
}

// GetAuthURL generates an OAuth2 authorization URL
func (g *GoogleCalendarService) GetAuthURL() (string, error) {
	// Load config to get OAuth2 credentials
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}
	
	if cfg.OAuth2.ClientID == "" || cfg.OAuth2.ClientSecret == "" {
		return "", fmt.Errorf("OAuth2 credentials not configured")
	}
	
	// Update global OAuth2 config
	SetOAuth2Config(cfg.OAuth2.ClientID, cfg.OAuth2.ClientSecret)
	
	// Generate state for security
	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	
	// Generate authorization URL
	authURL := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return authURL, nil
}

// RemoveAccount removes stored tokens for an account
func (g *GoogleCalendarService) RemoveAccount(accountID string) error {
	return config.RemoveToken(accountID)
}