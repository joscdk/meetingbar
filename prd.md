# Product Requirements Document (PRD)
## MeetingBar for Linux

**Version**: 1.0  
**Last Updated**: July 28, 2025  
**Product Owner**: [Your Name]  
**Target Release**: Q3 2025

---

## 1. Executive Summary

MeetingBar for Linux is a lightweight system tray application that helps users manage their Google Calendar meetings with one-click access to video conferences. It provides timely notifications and quick meeting joins for Google Meet, Microsoft Teams, and Zoom directly from the Linux system tray.

### Key Value Propositions
- Never miss a meeting with smart notifications
- Join meetings instantly without searching through calendars
- Minimal resource usage and distraction-free design
- Native Linux desktop integration

---

## 2. Problem Statement

Linux users lack a native, lightweight solution for managing calendar meetings from their system tray. Current solutions require:
- Keeping a browser tab open with Google Calendar
- Manually checking for upcoming meetings
- Searching through calendar events to find meeting links
- Missing meetings due to lack of timely notifications

### User Pain Points
1. **Meeting Discovery**: Hard to quickly see what's next without opening calendar
2. **Join Friction**: Multiple clicks needed to find and join meeting links
3. **Notification Gaps**: Browser notifications are unreliable when tabs are closed
4. **Resource Usage**: Full calendar apps consume significant memory

---

## 3. Goals and Success Metrics

### Primary Goals
1. Reduce time to join meetings to <3 seconds
2. Achieve 100% meeting notification delivery
3. Maintain memory footprint under 20MB
4. Support 95% of common meeting platforms

### Success Metrics
- **Adoption**: 1,000 active users within 3 months
- **Reliability**: 99.9% uptime over 24-hour periods
- **Performance**: <1 second startup time
- **User Satisfaction**: >4.5 star rating

---

## 4. User Personas

### Primary Persona: "DevOps Dave"
- **Role**: Senior DevOps Engineer
- **Age**: 32
- **Tech Savvy**: Expert
- **Needs**: 
  - Attends 5-8 meetings daily across different projects
  - Uses Ubuntu 22.04 with GNOME
  - Values minimal, efficient tools
  - Juggles multiple Google Workspace accounts

### Secondary Persona: "Manager Maria"
- **Role**: Engineering Manager  
- **Age**: 38
- **Tech Savvy**: Intermediate
- **Needs**:
  - Back-to-back meetings throughout the day
  - Uses Fedora with KDE
  - Needs reliable notifications
  - Manages team calendars and personal calendar

---

## 5. User Stories

### Must Have (P0)

1. **As a user, I want to** see my next meeting in the system tray **so that** I always know what's coming up
   - AC: Tray shows meeting title and time until start
   - AC: Updates automatically as time passes

2. **As a user, I want to** select which Google calendars to monitor **so that** I only see relevant meetings
   - AC: Settings shows all available calendars with checkboxes
   - AC: Can enable/disable individual calendars
   - AC: Changes apply immediately without restart

3. **As a user, I want to** join meetings with one click **so that** I'm never late
   - AC: Click meeting in tray menu opens browser with meeting URL
   - AC: Works for Google Meet, Teams, and Zoom links

4. **As a user, I want to** receive notifications before meetings **so that** I can prepare
   - AC: Desktop notification appears at configured time
   - AC: Notification includes meeting title and "Join" button

5. **As a user, I want to** authenticate with Google securely **so that** my credentials are protected
   - AC: OAuth2 flow in browser
   - AC: Tokens stored in system keyring
   - AC: Automatic token refresh

### Should Have (P1)

6. **As a user, I want to** see meetings from multiple Google accounts **so that** I can manage work and personal calendars
   - AC: Can add multiple Google accounts
   - AC: Calendar list shows which account each belongs to
   - AC: Can remove accounts individually

7. **As a user, I want to** configure notification timing **so that** I get alerts when I need them
   - AC: Dropdown with 1, 5, 10, 15 minute options
   - AC: Different settings per calendar possible

8. **As a user, I want to** launch MeetingBar at startup **so that** I don't forget to run it
   - AC: Checkbox in settings for "Launch at login"
   - AC: Creates appropriate autostart entry

### Nice to Have (P2)

9. **As a user, I want to** see meeting details on hover **so that** I can preview without clicking
   - AC: Tooltip shows full meeting title, time, and description preview

10. **As a user, I want to** snooze notifications **so that** I can delay reminders
    - AC: "Snooze 5 min" button in notification

---

## 6. Functional Requirements

### 6.1 System Tray Interface

#### Display States
1. **No Meetings**: "No meetings today"
2. **Meeting Soon**: "[Title] in [X]m"
3. **Meeting Now**: "Now: [Title]"
4. **In Meeting**: "In meeting: [Title]" (with different icon)

#### Menu Structure
```
┌─────────────────────────────┐
│ ▶ Team Standup - 10:00 AM   │ <- Next meeting highlighted
│   Product Review - 2:00 PM   │
│   1:1 with Maria - 3:30 PM   │
│ ─────────────────────────── │
│ 📅 Refresh                   │
│ ⚙️  Settings                  │
│ ─────────────────────────── │
│ ❌ Quit                       │
└─────────────────────────────┘
```

### 6.2 Calendar Integration

#### Calendar Selection
- Fetch all calendars from authenticated Google accounts
- Display calendars grouped by account:
  ```
  work@company.com
  ☑ My Calendar
  ☑ Team Calendar  
  ☐ Company Holidays
  
  personal@gmail.com
  ☑ Personal
  ☐ Family Calendar
  ```
- Store selection state persistently
- Apply filters to all calendar operations

#### Event Fetching
- Time range: Current time to +24 hours
- Include: Regular events, recurring events
- Exclude: All-day events (unless no other meetings)
- Fields needed: Title, Start/End time, Description, Location, Conference data

### 6.3 Meeting Link Detection

#### Supported Patterns
1. **Google Meet**
   - `meet.google.com/xxx-xxxx-xxx`
   - `https://meet.google.com/xxx-xxxx-xxx`
   
2. **Microsoft Teams**
   - `teams.microsoft.com/l/meetup-join/*`
   - `teams.live.com/meet/*`

3. **Zoom**
   - `zoom.us/j/[0-9]+`
   - `[subdomain].zoom.us/j/[0-9]+`
   - `zoom.us/my/[username]`

#### Link Priority
1. Check conference data field (Google Calendar native)
2. Check location field
3. Check description field
4. Use first valid link found

### 6.4 Settings Window

#### Layout
```
┌─ MeetingBar Settings ────────────────┐
│                                      │
│ Accounts:                            │
│ ┌──────────────────────────────────┐ │
│ │ work@company.com      [Remove]   │ │
│ │ personal@gmail.com    [Remove]   │ │
│ └──────────────────────────────────┘ │
│ [+ Add Google Account]               │
│                                      │
│ Calendars:                           │
│ ┌──────────────────────────────────┐ │
│ │ work@company.com                  │ │
│ │ ☑ My Calendar                    │ │
│ │ ☑ Team Calendar                  │ │
│ │ ☐ Company Holidays               │ │
│ │                                   │ │
│ │ personal@gmail.com                │ │
│ │ ☑ Personal                       │ │
│ │ ☐ Family Calendar                │ │
│ └──────────────────────────────────┘ │
│                                      │
│ Notifications:                       │
│ ☑ Enable meeting notifications       │
│ Notify [5 minutes ▼] before meeting  │
│                                      │
│ General:                             │
│ ☑ Launch at login                    │
│ Refresh every [5 minutes ▼]          │
│                                      │
│        [Cancel]  [Save]              │
└──────────────────────────────────────┘
```

---

## 7. Technical Requirements

### 7.1 Architecture

```
┌─────────────────┐     ┌──────────────────┐
│   System Tray   │────▶│  Meeting Manager │
└─────────────────┘     └──────────────────┘
                               │
                    ┌──────────┴──────────┐
                    ▼                     ▼
            ┌───────────────┐    ┌────────────────┐
            │ Google Calendar│    │ Notification   │
            │   Service     │    │   Service      │
            └───────────────┘    └────────────────┘
                    │
                    ▼
            ┌───────────────┐
            │ Secure Storage│
            │  (Keyring)    │
            └───────────────┘
```

### 7.2 Data Models

```go
type Meeting struct {
    ID          string
    Title       string
    StartTime   time.Time
    EndTime     time.Time
    MeetingLink string
    MeetingType string // "meet", "teams", "zoom"
    CalendarID  string
    AccountID   string
}

type Calendar struct {
    ID        string
    Name      string
    AccountID string
    Enabled   bool
    Color     string
}

type Account struct {
    ID    string
    Email string
    Token *oauth2.Token
}
```

### 7.3 Performance Requirements

- **Startup Time**: <1 second to tray icon visible
- **Memory Usage**: <20MB baseline, <30MB with 50 meetings
- **CPU Usage**: <1% when idle, <5% during refresh
- **Calendar Refresh**: Complete within 2 seconds
- **UI Responsiveness**: Menu opens within 100ms

---

## 8. Design Requirements

### 8.1 Visual Design

- **Tray Icons**: 
  - Normal: Calendar icon (monochrome)
  - In Meeting: Calendar with dot indicator
  - Error State: Calendar with exclamation mark
- **Theme Support**: Respect system light/dark theme
- **Font**: System default font
- **Spacing**: Follow GNOME HIG or KDE HIG based on desktop

### 8.2 Interaction Design

- **Left Click**: Open meeting menu
- **Right Click**: Open context menu (settings, quit)
- **Hover**: Show tooltip with next meeting details
- **Meeting Click**: Open browser immediately
- **Notification Click**: Open browser with meeting

---

## 9. Security & Privacy

### Requirements
1. OAuth2 tokens stored in system keyring (never plain text)
2. Calendar data cached encrypted at rest
3. No telemetry or usage tracking
4. All Google API calls over HTTPS
5. Minimal permissions requested (calendar.readonly)

### OAuth2 Scopes Required
- `https://www.googleapis.com/auth/calendar.readonly`
- `https://www.googleapis.com/auth/userinfo.email`

---

## 10. Platform Requirements

### Supported Distributions
- Ubuntu 20.04+
- Fedora 35+
- Debian 11+
- Arch Linux (current)
- openSUSE Leap 15.4+

### Desktop Environments
- GNOME 3.38+
- KDE Plasma 5.20+
- XFCE 4.16+
- Cinnamon 5.0+

### Dependencies
- systemd or equivalent for autostart
- Secret Service API for keyring
- D-Bus for notifications
- X11 or Wayland display server

---

## 11. Out of Scope (v1.0)

The following features are explicitly out of scope for initial release:
- Calendar providers other than Google
- Creating or modifying calendar events  
- Meeting room booking
- Slack/Discord/WebEx support
- Mobile companion app
- Windows/macOS versions
- Custom notification sounds
- Meeting transcription features

---

## 12. Release Criteria

### Beta Release
- [ ] Core functionality working on Ubuntu/GNOME
- [ ] Google Calendar authentication flow complete
- [ ] Meeting detection for all three platforms
- [ ] Basic settings persistence

### 1.0 Release
- [ ] All P0 user stories complete
- [ ] Tested on all supported distributions
- [ ] Memory usage under 20MB confirmed
- [ ] Documentation complete
- [ ] No critical bugs for 1 week
- [ ] Installation packages created (.deb, .rpm, AUR)

---

## 13. Future Roadmap (Post 1.0)

### Version 1.1
- CalDAV support for Nextcloud/ownCloud
- Outlook.com calendar support
- Custom notification sounds

### Version 2.0
- Calendar event creation
- Meeting room suggestions
- Colleague availability checking
- Integration with todo apps

---

## Appendix A: Mockups

[Include visual mockups of tray menu, notifications, and settings window]

## Appendix B: Competitive Analysis

| Feature | MeetingBar (macOS) | GNOME Calendar | Thunderbird | Our Solution |
|---------|-------------------|----------------|-------------|--------------|
| System Tray | ✓ | ✗ | ✗ | ✓ |
| One-click Join | ✓ | ✗ | ✗ | ✓ |
| Multiple Accounts | ✓ | ✓ | ✓ | ✓ |
| Memory Usage | ~50MB | ~200MB | ~300MB | <20MB |
| Calendar Selection | ✓ | ✓ | ✓ | ✓ |