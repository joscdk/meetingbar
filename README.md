# MeetingBar for Linux

A lightweight system tray application for Linux that displays your Google Calendar meetings with one-click join functionality for Google Meet, Microsoft Teams, and Zoom.

## Features

- **System Tray Integration**: Shows next meeting in system tray with time countdown
- **One-Click Meeting Join**: Click any meeting to open in your default browser
- **Multi-Platform Support**: Google Meet, Microsoft Teams, and Zoom detection
- **Desktop Notifications**: Configurable meeting reminders (1, 5, 10, 15 minutes)
- **Multiple Google Accounts**: Support for work and personal calendars
- **Calendar Selection**: Choose which calendars to monitor
- **Secure Storage**: OAuth2 tokens stored in system keyring
- **Lightweight**: <20MB memory usage, minimal CPU overhead

## Screenshots

```
System Tray: "Team Standup (5m)"

Menu:
┌─────────────────────────────┐
│ ▶ Team Standup - 10:00 AM   │
│   Product Review - 2:00 PM   │
│   1:1 with Maria - 3:30 PM   │
│ ─────────────────────────── │
│ 📅 Refresh                   │
│ ⚙️  Settings                  │
│ ─────────────────────────── │
│ ❌ Quit                       │
└─────────────────────────────┘
```

## Installation

### Prerequisites

- Linux distribution (Ubuntu 20.04+, Fedora 35+, Arch, Debian 11+)
- Desktop environment (GNOME, KDE, XFCE, Cinnamon)
- Go 1.21+ (for building from source)
- CGO enabled for system tray support

### From Release Package

1. Download the latest release from GitHub releases
2. Extract the archive:
   ```bash
   tar -xzf meetingbar-1.0.0-linux.tar.gz
   cd meetingbar-1.0.0
   ```
3. Run the installer:
   ```bash
   ./install.sh
   ```

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/meetingbar.git
   cd meetingbar
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Build and install:
   ```bash
   make build
   make install
   ```

## Configuration

### Google OAuth2 Setup

Before using MeetingBar, you need to configure Google OAuth2 credentials:

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google Calendar API
4. Create OAuth2 credentials:
   - Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client IDs"
   - Application type: "Desktop application"
   - Download the JSON file

5. Set environment variables or update the code with your credentials:
   ```bash
   export GOOGLE_CLIENT_ID="your-client-id"
   export GOOGLE_CLIENT_SECRET="your-client-secret"
   ```

### First Run

1. Launch MeetingBar:
   ```bash
   meetingbar
   ```

2. Right-click the tray icon and select "Settings"
3. Click "Add Google Account" and complete the OAuth flow
4. Select which calendars you want to monitor
5. Configure notification preferences

## Usage

### System Tray

The tray icon shows your meeting status:
- **"No meetings today"** - No upcoming meetings
- **"Meeting Name (15m)"** - Next meeting in 15 minutes
- **"In: Meeting Name"** - Currently in a meeting

### Menu Actions

- **Left-click**: Open meeting list menu
- **Click meeting**: Join meeting in browser
- **Right-click**: Access settings and quit options

### Notifications

Desktop notifications appear before meetings (configurable timing):
- Shows meeting title and start time
- Click notification to join meeting (if supported by desktop environment)

## Configuration Files

- **Config**: `~/.config/meetingbar/config.json`
- **Cache**: `~/.cache/meetingbar/`
- **Credentials**: System keyring (secure storage)

### Configuration Options

```json
{
  "accounts": [
    {
      "id": "user-id",
      "email": "user@example.com"
    }
  ],
  "enabled_calendars": ["calendar-id-1", "calendar-id-2"],
  "refresh_interval": 5,
  "notification_time": 5,
  "enable_notifications": true,
  "launch_at_login": false
}
```

## Building

### Requirements

- Go 1.21+
- CGO enabled
- Linux development headers

### Build Commands

```bash
# Install dependencies
make deps

# Build for current platform
make build

# Build for specific architectures
make build-linux-amd64
make build-linux-arm64

# Run tests
make test

# Format code
make fmt

# Lint code
make lint

# Create package
make package
```

## Supported Desktop Environments

- **GNOME 3.38+**: Full support including notification actions
- **KDE Plasma 5.20+**: Full support
- **XFCE 4.16+**: Basic support (no notification actions)
- **Cinnamon 5.0+**: Full support

## Meeting Link Detection

MeetingBar automatically detects meeting links in:
- Event location field
- Event description
- Google Calendar conference data

### Supported Patterns

- **Google Meet**: `meet.google.com/xxx-xxxx-xxx`
- **Teams**: `teams.microsoft.com/l/meetup-join/...`
- **Zoom**: `zoom.us/j/123456789`, `company.zoom.us/my/room`

## Troubleshooting

### Common Issues

1. **Tray icon not appearing**:
   - Ensure your desktop environment supports system tray
   - Try restarting the desktop environment

2. **OAuth2 authentication fails**:
   - Check client ID and secret configuration
   - Verify redirect URI is set to `http://localhost:8080/callback`

3. **No meetings showing**:
   - Check calendar permissions in Google account
   - Verify enabled calendars in settings
   - Check network connectivity

4. **Notifications not working**:
   - Ensure desktop notifications are enabled
   - Check notification daemon is running (`systemctl --user status dunst` or similar)

### Debug Mode

Run with debug logging:
```bash
DEBUG=1 meetingbar
```

### Logs

Application logs are written to stderr. To save logs:
```bash
meetingbar 2> meetingbar.log
```

## Security

- OAuth2 tokens stored in system keyring (never in plain text)
- Minimal API permissions requested (calendar.readonly)
- No telemetry or usage tracking
- Local-only data processing

## Performance

- Memory usage: <20MB baseline
- CPU usage: <1% idle, <5% during refresh
- Startup time: <1 second
- Calendar refresh: <2 seconds

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make lint` and `make test`
6. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- Inspired by [MeetingBar for macOS](https://github.com/leits/MeetingBar)
- Built with [systray](https://github.com/getlantern/systray) for cross-platform tray support
- Uses [zenity](https://github.com/ncruces/zenity) for native Linux dialogs