# MeetingBar - Claude Code Context

## Project Overview
MeetingBar is a Linux desktop application that shows upcoming Google Calendar meetings in the system tray with one-click join functionality. The application was originally designed for Google Calendar but has been extended to support GNOME Calendar integration as an alternative backend.

## Current Issue: GNOME Calendar Discovery Problem

### Background
The user requested GNOME Calendar integration as an alternative to Google Calendar while keeping both backends available. I successfully implemented:

1. **Unified Calendar Service**: `calendar/unified_service.go` - Routes calls between Google and GNOME backends
2. **GNOME Calendar Backend**: `calendar/gnome_calendar.go` - Uses Evolution Data Server D-Bus interfaces
3. **Web Settings Interface**: Updated to support calendar selection for both backends
4. **Configuration**: Added `CalendarBackend` field to switch between "google" and "gnome"

### Current Problem
GNOME Calendar discovery is finding 27 managed objects from Evolution Data Server and correctly identifying 4 objects with Calendar extensions, but **0 final calendars are being created**.

From the latest debug logs:
```
=== GNOME Calendar Discovery Summary ===
Total managed objects: 27
Objects with Source interface: 27
Objects with Calendar extension: 4
Potential calendars found: 4
Final calendars created: 0
========================================
```

The issue appears to be in property extraction. All 4 calendar objects have properties `[Parent Enabled UID]` but lack a `DisplayName` property, causing them to be filtered out.

## Architecture

### Key Files
- `calendar/gnome_calendar.go` - GNOME Calendar backend implementation
- `calendar/unified_service.go` - Backend abstraction layer
- `calendar/google_calendar.go` - Google Calendar backend
- `ui/tray.go` - System tray management
- `ui/web_settings.go` - Web-based settings interface
- `config/config.go` - Configuration management

### Calendar Backend Architecture
```go
type UnifiedCalendarService struct {
    ctx              context.Context
    config           *config.Config
    googleService    *GoogleCalendarService
    gnomeService     *GnomeCalendarService
}
```

The service routes calls based on `config.CalendarBackend` ("google" or "gnome").

### GNOME Calendar Implementation Details
Uses Evolution Data Server D-Bus interfaces:
- **Service**: `org.gnome.evolution.dataserver.Sources5`
- **Object Path**: `/org/gnome/evolution/dataserver/SourceManager`
- **Method**: `org.freedesktop.DBus.ObjectManager.GetManagedObjects`

Expected interfaces on calendar objects:
- `org.gnome.evolution.dataserver.Source` (has properties like UID, Parent, Enabled)
- `org.gnome.evolution.dataserver.Source.Calendar` (has BackendName property)

## Recent Changes

### Latest Debug Improvements (Still Not Working)
1. Enhanced property detection to try multiple display name properties
2. Added fallback logic to use UID property for display names
3. Improved logging to track exactly where the filtering occurs

### Code Structure
```go
type CalendarSource struct {
    ID          string
    DisplayName string
    Enabled     bool
    Color       string
    Backend     string
}
```

## Expected User Setup
- User has GNOME Calendar installed
- Evolution Data Server is running
- User has configured calendars in GNOME Calendar
- User selects "gnome" as CalendarBackend in settings

## Debug Commands
```bash
# Check if Evolution Data Server is running
busctl --user list | grep evolution

# Inspect managed objects manually
busctl --user call org.gnome.evolution.dataserver.Sources5 /org/gnome/evolution/dataserver/SourceManager org.freedesktop.DBus.ObjectManager GetManagedObjects
```

## Next Steps for Linux Machine
1. **Immediate Focus**: Fix the GNOME calendar discovery in `calendar/gnome_calendar.go`
2. **Root Issue**: Objects with Calendar extensions aren't being converted to final calendar objects
3. **Approach**: Need to debug why property extraction is failing or add more resilient fallback logic

## Key Configuration
```go
type Config struct {
    CalendarBackend string `mapstructure:"calendar_backend"` // "google" or "gnome"
    // ... other fields
}
```

## Test Instructions
```bash
# Build and run with debug output
go build -o meetingbar
./meetingbar 2>&1 | grep -A 20 "GNOME Calendar Discovery Summary"
```

The issue is definitely in the GNOME calendar discovery logic where 4 potential calendars are found but 0 are actually created. Focus debugging efforts on the property extraction and CalendarSource creation in `GetCalendars()` method.

## User's Last Feedback
"Same output" - indicating the latest property extraction improvements didn't resolve the issue. The problem persists where calendar objects are found but not converted to usable CalendarSource structs.