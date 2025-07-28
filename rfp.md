# Request for Proposal: MeetingBar for Linux

## Project Overview

Build a lightweight Linux system tray application that displays upcoming Google Calendar meetings with quick-join functionality for video conferences. This is inspired by the macOS MeetingBar application but tailored for Linux desktop environments.

## Core Requirements

### 1. Technology Stack
- **Language**: Go (Golang)
- **Target Platform**: Linux (Ubuntu 20.04+, Fedora 35+, Arch-based distros)
- **Desktop Environments**: Must support GNOME, KDE Plasma, XFCE

### 2. Calendar Integration
- **Scope**: Google Calendar support only
- **Authentication**: OAuth2 flow with secure token storage
- **Features**:
  - Fetch events from all calendars the user has access to
  - Support for recurring events
  - Handle all-day events appropriately
  - Refresh calendar data every 5 minutes (configurable)
  - Show events for current day + next 24 hours

### 3. Meeting Detection
Automatically detect and parse meeting links from calendar event descriptions and location fields:
- **Google Meet**: Patterns like `meet.google.com/xxx-xxxx-xxx`
- **Microsoft Teams**: Patterns like `teams.microsoft.com/l/meetup-join/...`
- **Zoom**: Patterns like `zoom.us/j/MEETINGID` or `*.zoom.us/my/ROOMNAME`

### 4. System Tray Functionality
- **Display Format**: Show next meeting title and time (e.g., "Team Standup in 15m")
- **Click Actions**:
  - Left-click: Open dropdown menu with upcoming meetings
  - Right-click: Context menu with settings and quit options
- **Menu Items** should include:
  - Next 5 meetings with times
  - Click any meeting to join instantly
  - Separator line
  - "Refresh" option
  - "Settings" option
  - "Quit" option

### 5. Meeting Actions
- **Join Meeting**: Open detected meeting URL in default browser
- **Pre-meeting Notifications**: 
  - 5 minutes before (default, configurable)
  - Native Linux desktop notifications
  - Include "Join Now" action button in notification
- **Meeting Status**: Update tray text during meetings (e.g., "In Meeting: Team Standup")

### 6. Settings Window
Simple GUI preferences window with:
- **Google Account**: Add/remove Google account (OAuth2 flow)
- **Calendar Selection**: Checkboxes to show/hide specific calendars
- **Notification Settings**: 
  - Enable/disable notifications
  - Notification time before meeting (1, 5, 10, 15 minutes)
- **Refresh Interval**: Dropdown (1, 5, 10, 30 minutes)
- **Startup**: Checkbox for "Launch at login"

### 7. Data Storage
- Store configuration in `~/.config/meetingbar/config.json`
- Store OAuth tokens securely using system keyring (e.g., Secret Service API)
- Cache calendar data in `~/.cache/meetingbar/`

## Technical Implementation Details

### Recommended Libraries
```go
module meetingbar

require (
    github.com/getlantern/systray         // System tray
    golang.org/x/oauth2                   // OAuth2 flow
    google.golang.org/api/calendar/v3     // Google Calendar API
    github.com/gen2brain/beeep           // Desktop notifications
    github.com/zalando/go-keyring        // Secure credential storage
    github.com/ncruces/zenity            // Native dialogs for settings
    github.com/spf13/viper               // Configuration management
)
```

### Project Structure
```
meetingbar/
├── main.go                 // Entry point, systray setup
├── calendar/
│   ├── google.go          // Google Calendar client
│   ├── auth.go            // OAuth2 flow
│   └── parser.go          // Meeting link detection
├── ui/
│   ├── tray.go            // System tray menu management
│   ├── settings.go        // Settings window
│   └── notifications.go   // Desktop notifications
├── config/
│   ├── config.go          // Configuration management
│   └── keyring.go         // Secure token storage
├── assets/
│   ├── icon.png           // Tray icon
│   └── icon_meeting.png   // Icon during meetings
└── build/
    └── linux/             // Build scripts and .desktop file
```

### Key Features to Implement

1. **Smart Meeting Detection**
   - Parse both Description and Location fields
   - Extract meeting ID/password if present
   - Handle multiple meeting links in one event

2. **Intelligent Display**
   - "No meetings today" when calendar is empty
   - "Meeting starting now" for current time
   - "Next: [Title] in Xm" for upcoming
   - Full-day events shown separately

3. **Error Handling**
   - Graceful offline mode
   - Token refresh without user intervention
   - Clear error messages in notifications

4. **Performance**
   - Minimal memory footprint (<20MB)
   - Efficient calendar polling
   - Quick startup time (<1 second)

## Deliverables

1. **Source Code**: Complete Go source code with comments
2. **Binary**: Compiled binary for x86_64 Linux
3. **Documentation**:
   - README.md with build instructions
   - User guide for setup and configuration
4. **Build Scripts**:
   - Makefile for building
   - .desktop file for application launcher
   - Basic install.sh script

## Success Criteria

- Application runs smoothly on major Linux distributions
- Google Calendar events appear in system tray within 30 seconds of startup
- Clicking meeting links opens them correctly in the default browser
- Settings persist between application restarts
- Memory usage stays under 20MB during normal operation
- No crashes during 24-hour continuous operation

## Additional Notes

- Focus on reliability over features
- Use native OS dialogs where possible
- Follow Linux desktop integration guidelines
- Make the UI minimal and distraction-free
- Ensure the app works well with both light and dark system themes

## Example User Flow

1. User launches MeetingBar
2. System tray icon appears showing "No meetings" or next meeting
3. User right-clicks and selects "Settings"
4. User clicks "Add Google Account" and completes OAuth flow
5. Calendar events immediately populate in the tray menu
6. 5 minutes before a meeting, user gets a desktop notification
7. User clicks "Join" in notification or tray menu
8. Browser opens with the meeting URL

Please build this application with clean, idiomatic Go code, proper error handling, and a focus on being a reliable daily-use tool for Linux users.