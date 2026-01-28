package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTerminalPane renders the embedded terminal pane
func (m model) renderTerminalPane() string {
	if !m.term.active {
		return ""
	}

	var content string
	borderColor := lipgloss.Color("99") // Purple accent

	// Check if we have static output (from MCP tools, etc.)
	if m.term.staticOutput != "" {
		content = m.term.staticOutput
	} else if m.term.vt != nil {
		// Get screen from vterm
		screen := m.term.vt.Screen
		if len(screen) == 0 {
			return ""
		}

		// Convert vterm screen to styled string
		var lines []string
		for _, row := range screen {
			var line strings.Builder
			for _, ch := range row {
				if ch.Rune == 0 {
					line.WriteRune(' ')
				} else {
					// Apply styling from vterm char
					style := lipgloss.NewStyle()

					// Foreground color (use Code for 256-color palette)
					if ch.Style.Fg.ColorMode != 0 {
						style = style.Foreground(lipgloss.Color(fmt.Sprintf("%d", ch.Style.Fg.Code)))
					}

					// Background color
					if ch.Style.Bg.ColorMode != 0 {
						style = style.Background(lipgloss.Color(fmt.Sprintf("%d", ch.Style.Bg.Code)))
					}

					// Text attributes
					if ch.Style.Bold {
						style = style.Bold(true)
					}
					if ch.Style.Italic {
						style = style.Italic(true)
					}
					if ch.Style.Underline {
						style = style.Underline(true)
					}
					if ch.Style.Reverse {
						style = style.Reverse(true)
					}

					line.WriteString(style.Render(string(ch.Rune)))
				}
			}
			lines = append(lines, line.String())
		}
		content = strings.Join(lines, "\n")

		// Gray border when not focused for vterm
		if !m.term.focused {
			borderColor = lipgloss.Color("240")
		}
	} else {
		return ""
	}

	// Status indicator
	var statusText string
	if m.term.staticOutput != "" {
		title := m.term.staticTitle
		if title == "" {
			title = "Output"
		}
		statusText = title + " | ESC: close"
	} else if m.term.exited {
		if m.term.exitErr != nil {
			statusText = fmt.Sprintf("Exited with error: %v | ESC: close", m.term.exitErr)
		} else {
			statusText = "Process exited | ESC: close"
		}
	} else if m.term.focused {
		statusText = "Terminal focused | F1: return to skitz"
	} else {
		statusText = "F1: toggle focus"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	status := statusStyle.Render(statusText)

	// Build the terminal pane with border
	termStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	termPane := termStyle.Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, termPane, status)
}
