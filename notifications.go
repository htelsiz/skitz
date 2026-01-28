package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Notification represents a toast message
type Notification struct {
	Message string
	Icon    string
	Style   string // "success", "info", "warning", "error"
}

// clearNotificationMsg clears the current notification
type clearNotificationMsg struct{}

// showNotification sets a notification and returns a command to clear it after delay
func (m *model) showNotification(icon, message, style string) tea.Cmd {
	m.notification = &Notification{
		Message: message,
		Icon:    icon,
		Style:   style,
	}
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

// renderNotification renders a toast notification
func (m model) renderNotification() string {
	if m.notification == nil {
		return ""
	}

	// Choose colors based on style
	var bgColor, fgColor lipgloss.Color
	switch m.notification.Style {
	case "success":
		bgColor = lipgloss.Color("42")  // Green
		fgColor = lipgloss.Color("255") // White
	case "error":
		bgColor = lipgloss.Color("196") // Red
		fgColor = lipgloss.Color("255") // White
	case "warning":
		bgColor = lipgloss.Color("214") // Orange
		fgColor = lipgloss.Color("235") // Dark
	default: // "info"
		bgColor = lipgloss.Color("99")  // Purple
		fgColor = lipgloss.Color("255") // White
	}

	toastStyle := lipgloss.NewStyle().
		Background(bgColor).
		Foreground(fgColor).
		Padding(0, 2).
		Bold(true)

	return toastStyle.Render(m.notification.Icon + "  " + m.notification.Message)
}
