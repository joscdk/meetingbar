package calendar

import (
	"regexp"
	"strings"
)

type MeetingType string

const (
	MeetingTypeGoogleMeet MeetingType = "meet"
	MeetingTypeTeams      MeetingType = "teams"
	MeetingTypeZoom       MeetingType = "zoom"
	MeetingTypeUnknown    MeetingType = "unknown"
)

var (
	// Google Meet patterns
	googleMeetRegex = regexp.MustCompile(`https?://meet\.google\.com/[a-z-]+`)
	
	// Microsoft Teams patterns
	teamsRegex = regexp.MustCompile(`https?://teams\.microsoft\.com/l/meetup-join/[^?\s]+`)
	teamsLiveRegex = regexp.MustCompile(`https?://teams\.live\.com/meet/[^?\s]+`)
	
	// Zoom patterns
	zoomRegex = regexp.MustCompile(`https?://[^/]*zoom\.us/j/\d+`)
	zoomMyRegex = regexp.MustCompile(`https?://[^/]*zoom\.us/my/[^?\s]+`)
)

type MeetingLink struct {
	URL  string
	Type MeetingType
}

func ParseMeetingLinks(description, location string) []MeetingLink {
	var links []MeetingLink
	
	// Combine description and location for parsing
	text := strings.Join([]string{description, location}, " ")
	
	// Find Google Meet links
	if matches := googleMeetRegex.FindAllString(text, -1); matches != nil {
		for _, match := range matches {
			links = append(links, MeetingLink{
				URL:  match,
				Type: MeetingTypeGoogleMeet,
			})
		}
	}
	
	// Find Microsoft Teams links
	if matches := teamsRegex.FindAllString(text, -1); matches != nil {
		for _, match := range matches {
			links = append(links, MeetingLink{
				URL:  match,
				Type: MeetingTypeTeams,
			})
		}
	}
	
	if matches := teamsLiveRegex.FindAllString(text, -1); matches != nil {
		for _, match := range matches {
			links = append(links, MeetingLink{
				URL:  match,
				Type: MeetingTypeTeams,
			})
		}
	}
	
	// Find Zoom links
	if matches := zoomRegex.FindAllString(text, -1); matches != nil {
		for _, match := range matches {
			links = append(links, MeetingLink{
				URL:  match,
				Type: MeetingTypeZoom,
			})
		}
	}
	
	if matches := zoomMyRegex.FindAllString(text, -1); matches != nil {
		for _, match := range matches {
			links = append(links, MeetingLink{
				URL:  match,
				Type: MeetingTypeZoom,
			})
		}
	}
	
	return links
}

func GetPrimaryMeetingLink(description, location string) *MeetingLink {
	links := ParseMeetingLinks(description, location)
	if len(links) == 0 {
		return nil
	}
	
	// Priority order: Google Meet, Teams, Zoom
	for _, link := range links {
		if link.Type == MeetingTypeGoogleMeet {
			return &link
		}
	}
	
	for _, link := range links {
		if link.Type == MeetingTypeTeams {
			return &link
		}
	}
	
	for _, link := range links {
		if link.Type == MeetingTypeZoom {
			return &link
		}
	}
	
	// Return first link if no priority matches
	return &links[0]
}