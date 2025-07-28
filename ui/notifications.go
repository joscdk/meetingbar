package ui

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	"meetingbar/calendar"
	"meetingbar/config"

	"github.com/gen2brain/beeep"
)

type NotificationManager struct {
	config          *config.Config
	meetings        []calendar.Meeting
	notifiedMeetings map[string]bool
}

func NewNotificationManager(cfg *config.Config) *NotificationManager {
	return &NotificationManager{
		config:           cfg,
		notifiedMeetings: make(map[string]bool),
	}
}

func (nm *NotificationManager) UpdateMeetings(meetings []calendar.Meeting) {
	nm.meetings = meetings
	nm.checkForUpcomingMeetings()
}

func (nm *NotificationManager) checkForUpcomingMeetings() {
	if !nm.config.EnableNotifications {
		return
	}

	now := time.Now()
	notificationTime := nm.config.GetNotificationDuration()

	for _, meeting := range nm.meetings {
		// Skip if already notified
		if nm.notifiedMeetings[meeting.ID] {
			continue
		}

		// Check if meeting is within notification window
		timeUntilMeeting := meeting.StartTime.Sub(now)
		if timeUntilMeeting <= notificationTime && timeUntilMeeting > 0 {
			nm.sendMeetingNotification(&meeting)
			nm.notifiedMeetings[meeting.ID] = true
		}
	}

	// Clean up old notifications (meetings that have passed)
	for meetingID := range nm.notifiedMeetings {
		found := false
		for _, meeting := range nm.meetings {
			if meeting.ID == meetingID && now.Before(meeting.EndTime) {
				found = true
				break
			}
		}
		if !found {
			delete(nm.notifiedMeetings, meetingID)
		}
	}
}

func (nm *NotificationManager) sendMeetingNotification(meeting *calendar.Meeting) {
	now := time.Now()
	timeUntil := meeting.StartTime.Sub(now)
	
	var timeText string
	if timeUntil < time.Minute {
		timeText = "starting now"
	} else if timeUntil < time.Hour {
		minutes := int(timeUntil.Minutes())
		timeText = fmt.Sprintf("in %d minutes", minutes)
	} else {
		timeText = fmt.Sprintf("at %s", meeting.StartTime.Format("15:04"))
	}

	title := "Upcoming Meeting"
	message := fmt.Sprintf("%s %s", meeting.Title, timeText)

	// Try to send notification with action button if meeting has a link
	if meeting.MeetingLink != nil {
		nm.sendNotificationWithAction(title, message, meeting)
	} else {
		// Send simple notification
		err := beeep.Notify(title, message, "")
		if err != nil {
			log.Printf("Failed to send notification: %v", err)
		}
	}
}

func (nm *NotificationManager) sendNotificationWithAction(title, message string, meeting *calendar.Meeting) {
	// Try to use native Linux desktop notifications with actions
	// This varies by desktop environment, so we'll try a few approaches
	
	// First try with notify-send (most common)
	if nm.tryNotifySend(title, message, meeting) {
		return
	}
	
	// Fallback to simple notification
	err := beeep.Notify(title, message, "")
	if err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (nm *NotificationManager) tryNotifySend(title, message string, meeting *calendar.Meeting) bool {
	// Check if notify-send is available
	if _, err := exec.LookPath("notify-send"); err != nil {
		return false
	}

	args := []string{
		"notify-send",
		"--app-name=MeetingBar",
		"--category=calendar",
		"--urgency=normal",
		title,
		message,
	}

	// Add action button if meeting has a link (GNOME/KDE support)
	if meeting.MeetingLink != nil {
		args = append(args, "--action=join=Join Meeting")
	}

	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()
	
	if err != nil {
		log.Printf("notify-send failed: %v", err)
		return false
	}

	// If we added an action, we need to handle the response
	// This is complex and varies by desktop environment
	// For now, we'll just log that we sent the notification
	log.Printf("Sent notification for meeting: %s", meeting.Title)
	return true
}

// StartNotificationWatcher starts a goroutine that periodically checks for upcoming meetings
func (nm *NotificationManager) StartNotificationWatcher() {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			nm.checkForUpcomingMeetings()
		}
	}()
}