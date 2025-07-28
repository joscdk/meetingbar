#!/bin/bash

# MeetingBar Installation Script
# This script installs MeetingBar on Linux systems

set -e

BINARY_NAME="meetingbar"
INSTALL_DIR="/usr/local/bin"
DESKTOP_FILE="meetingbar.desktop"
APPLICATIONS_DIR="$HOME/.local/share/applications"

echo "Installing MeetingBar..."

# Check if running as root
if [[ $EUID -eq 0 ]]; then
    echo "Please don't run this script as root. It will ask for sudo when needed."
    exit 1
fi

# Check if binary exists
if [[ ! -f "$BINARY_NAME" ]]; then
    echo "Error: $BINARY_NAME binary not found in current directory"
    exit 1
fi

# Install binary
echo "Installing binary to $INSTALL_DIR..."
sudo cp "$BINARY_NAME" "$INSTALL_DIR/"
sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Install desktop file
echo "Installing desktop file..."
mkdir -p "$APPLICATIONS_DIR"
cp "$DESKTOP_FILE" "$APPLICATIONS_DIR/"

# Update desktop database
if command -v update-desktop-database &> /dev/null; then
    update-desktop-database "$APPLICATIONS_DIR"
fi

# Create config directory
echo "Creating configuration directory..."
mkdir -p "$HOME/.config/meetingbar"
mkdir -p "$HOME/.cache/meetingbar"

echo ""
echo "MeetingBar has been installed successfully!"
echo ""
echo "To get started:"
echo "1. Run 'meetingbar' from terminal or find it in your applications menu"
echo "2. Right-click the tray icon and select 'Settings'"
echo "3. Add your Google account to start seeing your meetings"
echo ""
echo "Note: You'll need to configure OAuth2 credentials for Google Calendar access."
echo "See the README.md file for detailed setup instructions."
echo ""

# Optionally set up autostart
read -p "Would you like MeetingBar to start automatically at login? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    AUTOSTART_DIR="$HOME/.config/autostart"
    mkdir -p "$AUTOSTART_DIR"
    cp "$DESKTOP_FILE" "$AUTOSTART_DIR/"
    echo "Autostart enabled. MeetingBar will launch when you log in."
fi

echo "Installation complete!"