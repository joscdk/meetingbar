#!/bin/bash

echo "=== MeetingBar Linux Diagnostic ==="
echo "Date: $(date)"
echo "User: $(whoami)"
echo "OS: $(uname -a)"
echo ""

echo "=== Evolution Data Server Check ==="
if busctl --user list | grep -q evolution; then
    echo "✅ Evolution Data Server is running"
    busctl --user list | grep evolution
else
    echo "❌ Evolution Data Server not found"
    echo "Try: sudo apt install evolution-data-server"
fi
echo ""

echo "=== D-Bus Calendar Sources Test ==="
echo "Attempting to list calendar sources via D-Bus..."
if timeout 10s busctl --user call org.gnome.evolution.dataserver.Sources5 /org/gnome/evolution/dataserver/SourceManager org.freedesktop.DBus.ObjectManager GetManagedObjects > /tmp/eds-objects.txt 2>&1; then
    echo "✅ Successfully connected to Evolution Data Server"
    OBJECT_COUNT=$(grep -o "\"/" /tmp/eds-objects.txt | wc -l)
    echo "Found $OBJECT_COUNT managed objects"
    
    echo ""
    echo "Sample object paths:"
    grep -o '"/[^"]*"' /tmp/eds-objects.txt | head -5
    
    echo ""
    echo "Looking for Calendar interfaces:"
    if grep -q "org.gnome.evolution.dataserver.Source.Calendar" /tmp/eds-objects.txt; then
        echo "✅ Found Calendar interfaces in managed objects"
        CALENDAR_COUNT=$(grep -c "org.gnome.evolution.dataserver.Source.Calendar" /tmp/eds-objects.txt)
        echo "Calendar interface appears $CALENDAR_COUNT times"
    else
        echo "❌ No Calendar interfaces found"
    fi
else
    echo "❌ Failed to connect to Evolution Data Server"
    echo "Error details:"
    cat /tmp/eds-objects.txt
fi
echo ""

echo "=== GNOME Calendar Check ==="
if command -v gnome-calendar >/dev/null 2>&1; then
    echo "✅ GNOME Calendar is installed"
else
    echo "❌ GNOME Calendar not found"
    echo "Try: sudo apt install gnome-calendar"
fi
echo ""

echo "=== MeetingBar Build Test ==="
if [ -f "go.mod" ]; then
    echo "✅ Go module found"
    if go build -o meetingbar-test 2>&1; then
        echo "✅ Build successful"
        rm -f meetingbar-test
    else
        echo "❌ Build failed"
    fi
else
    echo "❌ No go.mod found - run from project directory"
fi
echo ""

echo "=== Next Steps ==="
echo "1. Run: go build -o meetingbar"
echo "2. Run: ./meetingbar 2>&1 | tee debug.log"
echo "3. Share debug.log with Claude Code"
echo "4. Check CLAUDE.md for full context"
echo ""

# Cleanup
rm -f /tmp/eds-objects.txt