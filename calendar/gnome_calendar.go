package calendar

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// Helper function to get property keys for debugging
func getPropertyKeys(properties map[string]dbus.Variant) []string {
	keys := make([]string, len(properties))
	i := 0
	for key := range properties {
		keys[i] = key
		i++
	}
	return keys
}

// GnomeCalendarService provides calendar access through Evolution Data Server
type GnomeCalendarService struct {
	ctx  context.Context
	conn *dbus.Conn
}

// CalendarSource represents a GNOME calendar source from EDS
type CalendarSource struct {
	ID          string
	DisplayName string
	Enabled     bool
	Color       string
	Backend     string
}

// NewGnomeCalendarService creates a new GNOME calendar service
func NewGnomeCalendarService(ctx context.Context) *GnomeCalendarService {
	return &GnomeCalendarService{
		ctx: ctx,
	}
}

// Connect establishes connection to Evolution Data Server via D-Bus
func (g *GnomeCalendarService) Connect() error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("failed to connect to D-Bus session bus: %w", err)
	}
	g.conn = conn
	return nil
}

// GetCalendars retrieves available calendar sources from Evolution Data Server
func (g *GnomeCalendarService) GetCalendars() ([]CalendarSource, error) {
	if g.conn == nil {
		if err := g.Connect(); err != nil {
			return nil, err
		}
	}

	// Use the ObjectManager interface to get all managed objects
	obj := g.conn.Object("org.gnome.evolution.dataserver.Sources5", "/org/gnome/evolution/dataserver/SourceManager")
	
	// Call GetManagedObjects to get all sources
	var managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err := obj.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&managedObjects)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed objects: %w", err)
	}

	log.Printf("Found %d managed objects from EDS", len(managedObjects))

	var calendars []CalendarSource
	var potentialCalendars int
	var sourceCount int
	var calendarExtensionCount int
	
	for objectPath, interfaces := range managedObjects {
		// Debug: log what interfaces each object has
		var interfaceNames []string
		for interfaceName := range interfaces {
			interfaceNames = append(interfaceNames, interfaceName)
		}
		log.Printf("Object %s has interfaces: %v", objectPath, interfaceNames)
		
		// Count objects with Source interface
		sourceInterface, hasSource := interfaces["org.gnome.evolution.dataserver.Source"]
		if hasSource {
			sourceCount++
		} else {
			log.Printf("Skipping %s - no Source interface", objectPath)
			continue
		}

		// Count objects with Calendar extension
		_, hasCalendar := interfaces["org.gnome.evolution.dataserver.Source.Calendar"]
		if hasCalendar {
			calendarExtensionCount++
			potentialCalendars++
		} else {
			log.Printf("Skipping %s - no Calendar extension", objectPath)
			continue // Skip non-calendar sources
		}
		
		log.Printf("✓ Found calendar source: %s with source properties: %v", objectPath, getPropertyKeys(sourceInterface))
		
		// Extract source properties
		var displayName string
		var enabled bool = true // Default to enabled if property not found
		
		// Try multiple property names for display name
		displayNameProperties := []string{"DisplayName", "display-name", "Name", "name", "Title", "title"}
		for _, propName := range displayNameProperties {
			if displayNameVariant, ok := sourceInterface[propName]; ok {
				if name, ok := displayNameVariant.Value().(string); ok {
					displayName = name
					log.Printf("  Found display name via '%s': %s", propName, name)
					break
				} else {
					log.Printf("  Property '%s' variant conversion failed: %T", propName, displayNameVariant.Value())
				}
			}
		}
		
		if displayName == "" {
			log.Printf("  No display name found in any of: %v", displayNameProperties)
		}
		
		if enabledVariant, ok := sourceInterface["Enabled"]; ok {
			if isEnabled, ok := enabledVariant.Value().(bool); ok {
				enabled = isEnabled
				log.Printf("  Enabled: %t", isEnabled)
			} else {
				log.Printf("  Enabled variant conversion failed: %T", enabledVariant.Value())
			}
		} else {
			log.Printf("  No Enabled property found, defaulting to true")
		}

		// Get backend type from Calendar extension if available
		var backend string
		if calendarInterface, hasCalendarExt := interfaces["org.gnome.evolution.dataserver.Source.Calendar"]; hasCalendarExt {
			log.Printf("  Calendar extension properties: %v", getPropertyKeys(calendarInterface))
			if backendVariant, ok := calendarInterface["BackendName"]; ok {
				if backendName, ok := backendVariant.Value().(string); ok {
					backend = backendName
					log.Printf("  Backend: %s", backendName)
				} else {
					log.Printf("  BackendName variant conversion failed: %T", backendVariant.Value())
				}
			} else {
				log.Printf("  No BackendName property found")
			}
		}

		// Use a default display name if none found
		if displayName == "" {
			// Try to use UID property as fallback
			if uidVariant, ok := sourceInterface["UID"]; ok {
				if uid, ok := uidVariant.Value().(string); ok {
					displayName = fmt.Sprintf("Calendar (%s)", uid)
					log.Printf("  Using UID as display name: %s", displayName)
				}
			}
			
			// Final fallback to object path
			if displayName == "" {
				pathParts := strings.Split(string(objectPath), "/")
				displayName = fmt.Sprintf("Calendar (%s)", pathParts[len(pathParts)-1])
				log.Printf("  Using object path as display name: %s", displayName)
			}
		}
		
		calendar := CalendarSource{
			ID:          string(objectPath),
			DisplayName: displayName,
			Enabled:     enabled,
			Backend:     backend,
			Color:       "#3366cc", // Default color, could be retrieved from EDS
		}
		
		log.Printf("✅ Adding calendar: '%s' (ID: %s, enabled: %t, backend: %s)", displayName, string(objectPath), enabled, backend)
		calendars = append(calendars, calendar)
	}

	log.Printf("=== GNOME Calendar Discovery Summary ===")
	log.Printf("Total managed objects: %d", len(managedObjects))
	log.Printf("Objects with Source interface: %d", sourceCount)
	log.Printf("Objects with Calendar extension: %d", calendarExtensionCount)
	log.Printf("Potential calendars found: %d", potentialCalendars)
	log.Printf("Final calendars created: %d", len(calendars))
	log.Printf("========================================")
	return calendars, nil
}

// GetMeetings retrieves calendar events from Evolution Data Server
func (g *GnomeCalendarService) GetMeetings(calendarIDs []string) ([]Meeting, error) {
	if g.conn == nil {
		if err := g.Connect(); err != nil {
			return nil, err
		}
	}

	var allMeetings []Meeting
	
	// Get time range for today
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	for _, calendarID := range calendarIDs {
		meetings, err := g.getMeetingsFromCalendar(calendarID, startOfDay, endOfDay)
		if err != nil {
			log.Printf("Failed to get meetings from calendar %s: %v", calendarID, err)
			continue
		}
		allMeetings = append(allMeetings, meetings...)
	}

	return allMeetings, nil
}

// getMeetingsFromCalendar retrieves events from a specific calendar
func (g *GnomeCalendarService) getMeetingsFromCalendar(calendarID string, start, end time.Time) ([]Meeting, error) {
	// Open calendar via Calendar Factory (using Calendar8 service)
	factoryObj := g.conn.Object("org.gnome.evolution.dataserver.Calendar8", "/org/gnome/evolution/dataserver/CalendarFactory")
	
	var calendarPath dbus.ObjectPath
	var busName string
	
	err := factoryObj.Call("org.gnome.evolution.dataserver.CalendarFactory.OpenCalendar", 0, calendarID).Store(&calendarPath, &busName)
	if err != nil {
		return nil, fmt.Errorf("failed to open calendar %s: %w", calendarID, err)
	}

	// Access the calendar object
	calendarObj := g.conn.Object(busName, calendarPath)
	
	// Create a query for events in the time range
	// EDS uses ISO time format for queries
	startISO := start.Format("20060102T150405Z")
	endISO := end.Format("20060102T150405Z")
	
	// S-expression query for events in time range
	query := fmt.Sprintf("(occur-in-time-range? (make-time \"%s\") (make-time \"%s\"))", startISO, endISO)
	
	var objects []string
	err = calendarObj.Call("org.gnome.evolution.dataserver.Calendar.GetObjectList", 0, query).Store(&objects)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendar events: %w", err)
	}

	var meetings []Meeting
	for _, objectData := range objects {
		meeting, err := g.parseCalendarObject(objectData)
		if err != nil {
			log.Printf("Failed to parse calendar object: %v", err)
			continue
		}
		if meeting != nil {
			meetings = append(meetings, *meeting)
		}
	}

	return meetings, nil
}

// parseCalendarObject parses iCalendar data into a Meeting struct
func (g *GnomeCalendarService) parseCalendarObject(icalData string) (*Meeting, error) {
	lines := strings.Split(icalData, "\n")
	
	var meeting Meeting
	var currentEvent bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "BEGIN:VEVENT" {
			currentEvent = true
			continue
		}
		
		if line == "END:VEVENT" {
			currentEvent = false
			break
		}
		
		if !currentEvent {
			continue
		}
		
		if strings.HasPrefix(line, "SUMMARY:") {
			meeting.Title = strings.TrimPrefix(line, "SUMMARY:")
		} else if strings.HasPrefix(line, "DTSTART:") {
			timeStr := strings.TrimPrefix(line, "DTSTART:")
			if t, err := g.parseICalTime(timeStr); err == nil {
				meeting.StartTime = t
			}
		} else if strings.HasPrefix(line, "DTEND:") {
			timeStr := strings.TrimPrefix(line, "DTEND:")
			if t, err := g.parseICalTime(timeStr); err == nil {
				meeting.EndTime = t
			}
		} else if strings.HasPrefix(line, "LOCATION:") {
			location := strings.TrimPrefix(line, "LOCATION:")
			if location != "" {
				// Check if location contains a meeting link
				if g.isMeetingLink(location) {
					meeting.MeetingLink = &MeetingLink{
						URL:  location,
						Type: g.detectMeetingTypeEnum(location),
					}
				}
			}
		}
	}
	
	// Only return meetings with required fields
	if meeting.Title == "" || meeting.StartTime.IsZero() || meeting.EndTime.IsZero() {
		return nil, fmt.Errorf("incomplete meeting data")
	}
	
	return &meeting, nil
}

// parseICalTime parses iCalendar time format
func (g *GnomeCalendarService) parseICalTime(timeStr string) (time.Time, error) {
	// Handle different iCalendar time formats
	formats := []string{
		"20060102T150405Z",     // UTC time
		"20060102T150405",      // Local time
		"20060102",             // Date only
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

// isMeetingLink checks if a string contains a video conferencing link
func (g *GnomeCalendarService) isMeetingLink(text string) bool {
	meetingDomains := []string{
		"meet.google.com",
		"zoom.us",
		"teams.microsoft.com",
		"webex.com",
		"gotomeeting.com",
	}
	
	text = strings.ToLower(text)
	for _, domain := range meetingDomains {
		if strings.Contains(text, domain) {
			return true
		}
	}
	return false
}

// detectMeetingType determines the type of meeting link (string version)
func (g *GnomeCalendarService) detectMeetingType(url string) string {
	url = strings.ToLower(url)
	switch {
	case strings.Contains(url, "meet.google.com"):
		return "Google Meet"
	case strings.Contains(url, "zoom.us"):
		return "Zoom"
	case strings.Contains(url, "teams.microsoft.com"):
		return "Microsoft Teams"
	case strings.Contains(url, "webex.com"):
		return "Webex"
	case strings.Contains(url, "gotomeeting.com"):
		return "GoToMeeting"
	default:
		return "Video Call"
	}
}

// detectMeetingTypeEnum determines the type of meeting link (enum version)
func (g *GnomeCalendarService) detectMeetingTypeEnum(url string) MeetingType {
	url = strings.ToLower(url)
	switch {
	case strings.Contains(url, "meet.google.com"):
		return MeetingTypeGoogleMeet
	case strings.Contains(url, "zoom.us"):
		return MeetingTypeZoom
	case strings.Contains(url, "teams.microsoft.com"):
		return MeetingTypeTeams
	default:
		return MeetingTypeUnknown
	}
}

// Close closes the D-Bus connection
func (g *GnomeCalendarService) Close() error {
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}