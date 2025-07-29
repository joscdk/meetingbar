#!/bin/bash

# MeetingBar Transfer to Linux Script
echo "=== MeetingBar Linux Transfer ==="
echo "This script helps transfer the project to your Linux machine"
echo ""

# Show current project structure
echo "Current project files:"
find . -type f -name "*.go" -o -name "*.md" -o -name "*.json" | head -20

echo ""
echo "Key files to transfer:"
echo "- All .go files in calendar/, ui/, config/ directories"
echo "- go.mod and go.sum"
echo "- CLAUDE.md (context file)"
echo ""

echo "Recommended transfer methods:"
echo "1. Git repository (if you have one)"
echo "2. scp/rsync to Linux machine"
echo "3. Archive and transfer"
echo ""

echo "Example scp command (replace with your details):"
echo "scp -r . user@linux-machine:~/meetingbar/"
echo ""

echo "Example rsync command:"
echo "rsync -av --exclude='.git' . user@linux-machine:~/meetingbar/"
echo ""

echo "Or create archive:"
echo "tar -czf meetingbar.tar.gz --exclude='.git' --exclude='meetingbar' ."
echo "scp meetingbar.tar.gz user@linux-machine:~/"
echo ""

echo "On Linux machine:"
echo "cd ~/meetingbar"
echo "go mod download"
echo "go build -o meetingbar"
echo "./meetingbar 2>&1 | tee debug.log"
echo ""

echo "The CLAUDE.md file contains all context for Claude Code on Linux!"