# MeetingBar Development Notes

This document contains important development notes and patterns discovered during the implementation of MeetingBar.

## Architecture Overview

The application follows a clean separation of concerns:

- `main.go`: Entry point, minimal setup
- `config/`: Configuration management and secure storage
- `calendar/`: Google Calendar integration and OAuth2 flow
- `ui/`: System tray interface, notifications, and settings

## Key Patterns and Decisions

### Configuration Management

- Uses Viper for configuration file handling (`~/.config/meetingbar/config.json`)
- Secure credential storage via system keyring using `go-keyring`
- OAuth2 tokens are never stored in plain text
- Configuration includes refresh intervals, notification settings, and account management

### OAuth2 Flow

- Uses `golang.org/x/oauth2` with Google's endpoints
- Temporary HTTP server on localhost:8080 for callback handling
- State parameter for CSRF protection
- Automatic token refresh with secure storage updates
- Minimal scopes requested: `calendar.readonly` and `userinfo.email`

### Meeting Detection

- Regex-based parsing for Google Meet, Teams, and Zoom links
- Priority order: Google Meet > Teams > Zoom
- Checks both event description and location fields
- Also leverages Google Calendar's native conference data when available

### System Tray Integration

- Uses `getlantern/systray` for cross-platform tray support
- Dynamic menu generation based on meeting schedule
- Meeting items limited to 5 for UI clarity
- Time-based display formatting (minutes vs. hours)
- Automatic refresh every 5 minutes (configurable)

### Notification System

- Desktop notifications via `gen2brain/beeep`
- Fallback to `notify-send` for Linux-specific features
- Configurable timing (1, 5, 10, 15 minutes before meetings)
- Prevents duplicate notifications via tracking map
- Notification cleanup for past meetings

### Settings UI

- Uses `ncruces/zenity` for native Linux dialogs
- Account management with OAuth2 flow integration
- Calendar selection from all available calendars
- Notification and refresh interval configuration
- Graceful fallback for environments without zenity

## Dependencies and Their Purposes

- `getlantern/systray`: Cross-platform system tray support
- `golang.org/x/oauth2`: OAuth2 authentication flow
- `google.golang.org/api/calendar/v3`: Google Calendar API client
- `gen2brain/beeep`: Cross-platform desktop notifications
- `zalando/go-keyring`: Secure credential storage
- `ncruces/zenity`: Native Linux dialog boxes
- `spf13/viper`: Configuration file management

## Performance Considerations

- Calendar data cached to minimize API calls
- Efficient meeting sorting and filtering
- Limited menu items to prevent UI clutter
- Configurable refresh intervals (default 5 minutes)
- Minimal memory footprint design (<20MB target)

## Security Considerations

- OAuth2 tokens stored in system keyring only
- Minimal API scopes requested
- CSRF protection in OAuth flow
- No telemetry or external data transmission
- Local-only data processing and caching

## Build Requirements

- CGO enabled for system tray and keyring access
- Linux development headers for native integrations
- Go 1.21+ for modern language features
- Desktop environment with system tray support

## Known Limitations

- Google Calendar only (other providers out of scope for v1.0)
- Linux-only implementation
- Requires desktop environment with tray support
- OAuth2 credentials must be manually configured
- Limited notification actions (varies by desktop environment)

## Testing Considerations

- OAuth2 flow requires browser and user interaction
- System tray testing requires X11/Wayland environment
- Keyring operations may require user session
- Desktop notifications depend on notification daemon

## Future Improvements

- Add proper tray icons (currently placeholder)
- Implement notification action buttons where supported
- Add calendar event creation capabilities
- Support for additional calendar providers (CalDAV)
- Windows and macOS ports using same core logic