package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func (m model) renderDashboard() string {
	contentH := m.height - 2

	// Logo style
	logoStyle := lipgloss.NewStyle().Foreground(primary)

	// Crane with BIA bar underneath
	craneStyle := lipgloss.NewStyle().Foreground(primary)
	biaYellow := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	biaBlack := lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("220"))

	crane := craneStyle.Render(`⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠿⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⡿⠟⠋⣁⡄⠀⢠⣄⣉⡙⠛⠿⢿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⠿⠛⣁⣤⣶⣿⠇⣤⠈⣿⣿⣿⣿⣶⣦⣄⣉⠙⠛⠿
⣿⣿⣯⣤⣴⣿⣿⣿⣿⣿⣤⣿⣤⣽⣿⣿⣿⣿⣿⣿⣿⣿⣷⣦
⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿
⣿⣿⣿⡟⠛⠛⠛⣿⣿⣿⣿⡟⠛⢻⡟⠛⢻⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣷⣶⣶⣶⣿⣿⣿⣿⣇⣀⣸⣇⣀⣼⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡏⠉⢹⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡇⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⡇⠀⢸⡿⣿⣿⣿⣿⠀⠀⠀⢸⣿
⣿⣿⣿⣿⣿⣿⣿⡿⠋⣁⣴⡇⠀⢸⣷⣌⠙⢿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣷⣾⣿⣿⣷⣤⣼⣿⣿⣿⣶⣿⣿⣿⣿⣿⣿`)

	biaBar := biaYellow.Render("▟") + biaBlack.Bold(true).Render(" B I A ") + biaYellow.Render("▙")

	biaLogo := lipgloss.JoinVertical(lipgloss.Center, crane, biaBar)

	// Clean block title
	titleArt := logoStyle.Render(`█▀ █▄▀ █ ▀█▀ ▀█
▄█ █ █ █  █  █▄`)

	// Styles
	versionStyle := lipgloss.NewStyle().Foreground(subtle)
	descStyle := lipgloss.NewStyle().Foreground(secondary).Italic(true)

	// Animated quote with typewriter effect
	quoteText := `"It is with us and in control"`
	visibleChars := int(m.quotePos)
	if visibleChars > len(quoteText) {
		visibleChars = len(quoteText)
	}
	revealedQuote := quoteText[:visibleChars]

	// Build the padded quote to maintain box width
	var paddedQuote string
	if visibleChars < len(quoteText) {
		spacesNeeded := len(quoteText) - visibleChars - 1
		if spacesNeeded < 0 {
			spacesNeeded = 0
		}
		paddedQuote = revealedQuote + "▌" + strings.Repeat(" ", spacesNeeded)
	} else {
		paddedQuote = revealedQuote
	}

	quoteStyle := lipgloss.NewStyle().Foreground(primary).Italic(true)

	// Header width
	headerW := m.width - 4
	if headerW < 60 {
		headerW = 60
	}

	// Build the title block with version and description inline
	titleBlock := lipgloss.JoinVertical(lipgloss.Left,
		titleArt,
		versionStyle.Render("v0.1")+" "+descStyle.Render("Command Center"),
	)

	// Combine BIA logo and title side by side (BIA logo includes crane imagery)
	headerTop := lipgloss.JoinHorizontal(lipgloss.Center, biaLogo, "    ", titleBlock)

	// Quote box
	quoteBox := quoteStyle.Render(fmt.Sprintf(`╭──────────────────────────────────╮
│  %s  │
╰──────────────────────────────────╯`, paddedQuote))

	// Decorative border styles
	borderStyle := lipgloss.NewStyle().Foreground(dimBorder)

	// Simple borders without tagline
	topBorder := borderStyle.Render("╔" + strings.Repeat("═", headerW-2) + "╗")
	bottomBorder := borderStyle.Render("╚" + strings.Repeat("═", headerW-2) + "╝")

	// Combine header content (centered)
	headerInner := lipgloss.JoinVertical(lipgloss.Center,
		"",
		headerTop,
		"",
		quoteBox,
		"",
	)
	headerInner = lipgloss.NewStyle().Width(headerW).Align(lipgloss.Center).Render(headerInner)

	header := lipgloss.JoinVertical(lipgloss.Left,
		topBorder,
		headerInner,
		bottomBorder,
	)

	// Layout: 30% actions panel (left), 70% main content (right)
	actionsW := (m.width * 25) / 100
	mainAreaW := m.width - actionsW - 3 // -3 for spacing

	// Actions panel (full height on left)
	// Consistent indentation: 2 spaces for items, 4 for sub-items, 6 for sub-sub-items
	actionsTitleStyle := lipgloss.NewStyle().
		Foreground(secondary).
		Bold(true)

	actionItemStyle := lipgloss.NewStyle().
		Foreground(white)

	actionDimStyle := lipgloss.NewStyle().
		Foreground(subtle)

	maxLineLen := actionsW - 6
	if maxLineLen < 10 {
		maxLineLen = 10
	}

	// Sidebar content
	var sidebarLines []string

	// Command palette hint (prominent)
	paletteHintStyle := lipgloss.NewStyle().
		Background(primary).
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Padding(0, 1)

	paletteDescStyle := lipgloss.NewStyle().
		Foreground(subtle).
		Italic(true)

	sidebarLines = append(sidebarLines,
		paletteHintStyle.Render("⌘K Command Palette"),
		paletteDescStyle.Render("  ctrl+k to open"),
	)

	// Favorites section
	if len(m.config.Favorites) > 0 {
		sidebarLines = append(sidebarLines, "", actionsTitleStyle.Render("⭐ Favorites"))
		for i, fav := range m.config.Favorites {
			if i >= 3 {
				sidebarLines = append(sidebarLines, actionDimStyle.Render(fmt.Sprintf("  +%d more", len(m.config.Favorites)-3)))
				break
			}
			favDisplay := fav
			if len(favDisplay) > 18 {
				favDisplay = favDisplay[:15] + "..."
			}
			sidebarLines = append(sidebarLines, actionItemStyle.Render("  "+favDisplay))
		}
	}

	// Agent interactions section
	sidebarLines = append(sidebarLines, "", actionsTitleStyle.Render("🤖 Agent History"))

	agentDisplayCount := 3
	if agentDisplayCount > len(m.agentHistory) {
		agentDisplayCount = len(m.agentHistory)
	}

	if agentDisplayCount == 0 {
		sidebarLines = append(sidebarLines, actionDimStyle.Render("  No agent chats"))
	} else {
		for i := 0; i < agentDisplayCount; i++ {
			entry := m.agentHistory[i]
			timeAgo := formatTimeAgo(entry.Timestamp)
			actionDisplay := entry.Action
			if len(actionDisplay) > 12 {
				actionDisplay = actionDisplay[:10] + ".."
			}

			statusIcon := "✓"
			if !entry.Success {
				statusIcon = "✗"
			}

			line := fmt.Sprintf("  %s %s", statusIcon, actionDisplay)
			sidebarLines = append(sidebarLines,
				actionItemStyle.Render(line)+actionDimStyle.Render("  "+timeAgo))
		}
	}

	// MCP connections section
	if m.config.MCP.Enabled {
		sidebarLines = append(sidebarLines, "", actionsTitleStyle.Render("🧩 MCP Connections"))
		if len(m.mcpStatus) == 0 {
			sidebarLines = append(sidebarLines, actionDimStyle.Render("  No MCP data"))
		} else {
			appendList := func(label string, items []string, errText string) {
				if errText != "" {
					sidebarLines = append(sidebarLines, actionDimStyle.Render("    "+truncate(errText, maxLineLen-6)))
					return
				}

				sidebarLines = append(sidebarLines, actionDimStyle.Render(fmt.Sprintf("    %s (%d)", strings.ToLower(label), len(items))))
				if len(items) == 0 {
					sidebarLines = append(sidebarLines, actionDimStyle.Render("      none"))
					return
				}

				maxItems := 3
				itemsToShow := items
				if len(items) > maxItems {
					itemsToShow = items[:maxItems]
				}
				for _, item := range itemsToShow {
					sidebarLines = append(sidebarLines, actionDimStyle.Render("      "+truncate(item, maxLineLen-10)))
				}
				if len(items) > maxItems {
					sidebarLines = append(sidebarLines, actionDimStyle.Render(fmt.Sprintf("      +%d more", len(items)-maxItems)))
				}
			}

			for _, status := range m.mcpStatus {
				displayName := status.Name
				if displayName == "" {
					displayName = status.URL
				}

				nameLine := truncate(displayName, maxLineLen-6)
				statusIcon := "✗"
				statusColor := lipgloss.Color("196")
				statusLabel := "disconnected"
				if status.Connected {
					statusIcon = "✓"
					statusColor = lipgloss.Color("114")
					statusLabel = "connected"
				}

				statusStyle := lipgloss.NewStyle().Foreground(statusColor)
				sidebarLines = append(sidebarLines, statusStyle.Render("  "+statusIcon+" "+nameLine+" "+statusLabel))
				if status.URL != "" {
					sidebarLines = append(sidebarLines, actionDimStyle.Render("    url: "+truncate(status.URL, maxLineLen-8)))
				}

				if status.Error != "" {
					errLine := truncate(status.Error, maxLineLen-6)
					sidebarLines = append(sidebarLines, actionDimStyle.Render("    "+errLine))
					continue
				}

				appendList("Tools", status.Tools, status.ToolsError)
				appendList("Prompts", status.Prompts, status.PromptsError)
				appendList("Resources", status.Resources, status.ResourcesError)
				appendList("Templates", status.ResourceTemplates, status.ResourceTemplatesError)
			}
		}
	}

	// Recent history section
	sidebarLines = append(sidebarLines, "", actionsTitleStyle.Render("⏱ Recent"))

	displayCount := m.config.History.DisplayCount
	if displayCount > len(m.history) {
		displayCount = len(m.history)
	}

	if displayCount == 0 {
		sidebarLines = append(sidebarLines, actionDimStyle.Render("  No history yet"))
	} else {
		for i := 0; i < displayCount; i++ {
			entry := m.history[i]
			cmdDisplay := entry.Command
			if len(cmdDisplay) > 18 {
				cmdDisplay = cmdDisplay[:15] + "..."
			}
			if entry.Tool != "" {
				sidebarLines = append(sidebarLines, actionDimStyle.Render(fmt.Sprintf("  [%s] %s", entry.Tool[:1], cmdDisplay)))
			} else {
				sidebarLines = append(sidebarLines, actionDimStyle.Render("  "+cmdDisplay))
			}
		}
	}

	actionsContent := lipgloss.JoinVertical(lipgloss.Left, sidebarLines...)

	actionsPanel := paneStyle.
		Width(actionsW).
		Height(contentH).
		Padding(1, 2).
		Render(actionsContent)

	// Adjust header width to fit main area
	headerW = mainAreaW - 2
	if headerW < 60 {
		headerW = 60
	}

	// Recalculate borders for new width
	topBorder = borderStyle.Render("╔" + strings.Repeat("═", headerW-2) + "╗")
	bottomBorder = borderStyle.Render("╚" + strings.Repeat("═", headerW-2) + "╝")

	// Rebuild header content with correct width and centering
	headerInner = lipgloss.JoinVertical(lipgloss.Center,
		"",
		headerTop,
		"",
		quoteBox,
		"",
	)
	headerInner = lipgloss.NewStyle().Width(headerW).Align(lipgloss.Center).Render(headerInner)

	header = lipgloss.JoinVertical(lipgloss.Left,
		topBorder,
		headerInner,
		bottomBorder,
	)

	// Calculate card width (fit 2-3 cards per row in the main area)
	cardW := (mainAreaW - 6) / 3
	if cardW < 25 {
		cardW = (mainAreaW - 4) / 2
	}
	if cardW < 25 {
		cardW = mainAreaW - 4
	}

	// Render enhanced resource cards
	var cards []string
	for i, res := range m.resources {
		meta := toolMetadata[res.name]
		isSelected := i == m.resCursor

		// Keyboard shortcut
		shortcut := lipgloss.NewStyle().
			Foreground(subtle).
			Render(fmt.Sprintf("[%d]", i+1))

		// Tool name (bold, colored like category)
		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(meta.color)
		if meta.status == "coming_soon" {
			nameStyle = lipgloss.NewStyle().Bold(true).Foreground(subtle)
		}
		// Uppercase for more prominence
		toolName := nameStyle.Render(strings.ToUpper(res.name))

		// Description
		descCardStyle := lipgloss.NewStyle().Foreground(subtle)
		if meta.status == "coming_soon" {
			descCardStyle = descCardStyle.Italic(true)
		}
		description := descCardStyle.Render(res.description)

		// Category tag
		categoryStyle := lipgloss.NewStyle().
			Foreground(meta.color).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
		categoryTag := categoryStyle.Render(meta.category)

		// Build card content
		cardContent := lipgloss.JoinVertical(lipgloss.Left,
			toolName+"  "+shortcut,
			description,
			"",
			categoryTag,
		)

		// Card border style based on state
		var cardBorderStyle lipgloss.Style
		if meta.status == "coming_soon" {
			// Dimmed dashed-look border for coming soon
			cardBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Padding(1, 1)
		} else if isSelected {
			// Glowing border for selected
			cardBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(meta.color).
				Padding(1, 1)
		} else {
			// Subtle border with accent for default
			cardBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(dimBorder).
				Padding(1, 1)
		}

		card := cardBorderStyle.Width(cardW - 2).Render(cardContent)
		cards = append(cards, card)
	}

	// Arrange cards in rows
	cardsPerRow := mainAreaW / cardW
	if cardsPerRow < 1 {
		cardsPerRow = 1
	}

	var rows []string
	for i := 0; i < len(cards); i += cardsPerRow {
		end := i + cardsPerRow
		if end > len(cards) {
			end = len(cards)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards[i:end]...)
		rows = append(rows, row)
	}

	cardGrid := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Right side: header + cards
	rightContent := lipgloss.JoinVertical(lipgloss.Left, header, cardGrid)

	// Combine: actions panel (left) + main content (right)
	body := lipgloss.JoinHorizontal(lipgloss.Top, actionsPanel, " ", rightContent)

	// Pad to fill height
	bodyH := lipgloss.Height(body)
	if bodyH < contentH {
		body = body + strings.Repeat("\n", contentH-bodyH)
	}

	return body
}

// renderCommandList renders a beautiful command list with selection highlighting
func (m model) renderCommandList(width int, accentColor lipgloss.Color) string {
	if len(m.commands) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true).
			Padding(2, 2)
		return emptyStyle.Render("No runnable commands in this section")
	}

	var lines []string

	// Calculate command and description widths from available space
	// Leave room for: indicator(3) + num(4) + spacing(4)
	availableW := width - 11
	cmdW := (availableW * 50) / 100
	descW := availableW - cmdW

	if cmdW < 25 {
		cmdW = 25
	}
	if descW < 15 {
		descW = 15
	}

	for i, cmd := range m.commands {
		isSelected := i == m.cmdCursor

		// Prepare command text (truncate if needed)
		cmdText := cmd.raw
		if len(cmdText) > cmdW-2 {
			cmdText = cmdText[:cmdW-5] + "..."
		}

		// Prepare description text (truncate if needed)
		descText := cmd.description
		if len(descText) > descW-2 {
			descText = descText[:descW-5] + "..."
		}

		// Pad command to fixed width (plain string, no ANSI)
		cmdPadded := cmdText + strings.Repeat(" ", max(0, cmdW-len(cmdText)))

		if isSelected {
			// ═══════════════════════════════════════════════════════════
			// SELECTED ROW - Left border accent + background highlight
			// ═══════════════════════════════════════════════════════════

			indicator := lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Render("▶ ")

			num := lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Render(fmt.Sprintf("%-3d", i+1))

			cmdStyled := lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("240")).
				Bold(true).
				Render(" " + cmdPadded + " ")

			desc := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Render(descText)

			// Build row without width constraint
			row := indicator + num + cmdStyled + " " + desc

			// Add left border for selection indicator
			line := lipgloss.NewStyle().
				Foreground(accentColor).
				Render("┃") + " " +
				lipgloss.NewStyle().
					Background(lipgloss.Color("237")).
					Render(row)

			lines = append(lines, line)

		} else {
			// ═══════════════════════════════════════════════════════════
			// NORMAL ROW - Clean, minimal style
			// ═══════════════════════════════════════════════════════════

			num := lipgloss.NewStyle().
				Foreground(subtle).
				Render(fmt.Sprintf("%-3d", i+1))

			cmdStyled := lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")).
				Background(lipgloss.Color("236")).
				Render(" " + cmdPadded + " ")

			desc := lipgloss.NewStyle().
				Foreground(subtle).
				Render(descText)

			// Build row with spacing to align with selected rows
			row := "  " + num + cmdStyled + " " + desc

			lines = append(lines, row)
		}
	}

	return strings.Join(lines, "\n")
}

// renderResourceView renders the full-screen resource view
func (m model) renderResourceView() string {
	res := m.currentResource()
	if res == nil {
		return ""
	}

	meta := toolMetadata[res.name]

	// Full-screen width
	viewW := m.width

	// ═══════════════════════════════════════════════════════════════
	// TABS - Premium tab bar with proper tab styling
	// ═══════════════════════════════════════════════════════════════

	// Build tab row with visual tab shapes
	var tabRow1, tabRow2, tabRow3 []string

	for i, s := range res.sections {
		// Truncate long titles
		title := s.title
		if len(title) > 14 {
			title = title[:12] + ".."
		}

		// Format label with number - consistent padding
		var label string
		if i < 9 {
			label = fmt.Sprintf("  %d  %s  ", i+1, title)
		} else {
			label = fmt.Sprintf("  %s  ", title)
		}
		labelW := len(label)

		if i == m.secCursor {
			// ══════════════════════════════════════════════
			// ACTIVE TAB - Prominent with accent color, filled style
			// ══════════════════════════════════════════════
			topBorder := lipgloss.NewStyle().
				Foreground(meta.color).
				Render("┏" + strings.Repeat("━", labelW) + "┓")

			content := lipgloss.NewStyle().
				Foreground(meta.color).
				Render("┃") +
				lipgloss.NewStyle().
					Background(meta.color).
					Foreground(lipgloss.Color("255")).
					Bold(true).
					Render(label) +
				lipgloss.NewStyle().
					Foreground(meta.color).
					Render("┃")

			// Bottom open to connect with content
			bottomBorder := lipgloss.NewStyle().
				Foreground(meta.color).
				Render("┗" + strings.Repeat("━", labelW) + "┛")

			tabRow1 = append(tabRow1, topBorder)
			tabRow2 = append(tabRow2, content)
			tabRow3 = append(tabRow3, bottomBorder)
		} else {
			// ══════════════════════════════════════════════
			// INACTIVE TAB - Outline only, no fill
			// ══════════════════════════════════════════════
			topBorder := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("┌" + strings.Repeat("─", labelW) + "┐")

			content := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("│") +
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("248")).
					Render(label) +
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("240")).
					Render("│")

			bottomBorder := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("└" + strings.Repeat("─", labelW) + "┘")

			tabRow1 = append(tabRow1, topBorder)
			tabRow2 = append(tabRow2, content)
			tabRow3 = append(tabRow3, bottomBorder)
		}

		// Add spacing between tabs
		if i < len(res.sections)-1 {
			tabRow1 = append(tabRow1, "   ")
			tabRow2 = append(tabRow2, "   ")
			tabRow3 = append(tabRow3, "───")
		}
	}

	// Combine the three rows
	row1 := strings.Join(tabRow1, "")
	row2 := strings.Join(tabRow2, "")
	row3 := strings.Join(tabRow3, "")

	tabBar := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().PaddingLeft(1).Render(row1),
		lipgloss.NewStyle().PaddingLeft(1).Render(row2),
		lipgloss.NewStyle().PaddingLeft(1).Render(row3),
	)

	// Accent line under tabs (uses tool color)
	accentLine := lipgloss.NewStyle().
		Foreground(meta.color).
		Render(strings.Repeat("─", viewW))

	// ═══════════════════════════════════════════════════════════════
	// INFO BAR - Command count and instructions
	// ═══════════════════════════════════════════════════════════════
	cmdCount := len(m.commands)
	var infoBar string

	if cmdCount > 0 {
		infoBg := lipgloss.NewStyle().
			Background(lipgloss.Color("234"))

		countStyle := lipgloss.NewStyle().
			Foreground(meta.color).
			Bold(true)

		textStyle := lipgloss.NewStyle().
			Foreground(subtle)

		keyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238")).
			Padding(0, 1)

		infoContent := countStyle.Render(fmt.Sprintf(" %d", cmdCount)) +
			textStyle.Render(" commands  ") +
			keyStyle.Render("↑↓") + textStyle.Render(" select  ") +
			keyStyle.Render("enter") + textStyle.Render(" run  ") +
			keyStyle.Render("ctrl+y") + textStyle.Render(" copy")

		infoBar = infoBg.Width(viewW).Padding(0, 1).Render(infoContent)
	} else {
		infoBar = lipgloss.NewStyle().
			Background(lipgloss.Color("234")).
			Foreground(subtle).
			Width(viewW).
			Padding(0, 2).
			Render("No runnable commands in this section")
	}

	// ═══════════════════════════════════════════════════════════════
	// CONTENT - Always use viewport (glamour handles commands better)
	// ═══════════════════════════════════════════════════════════════
	var contentArea string

	if m.viewReady {
		// Use viewport for all content (including commands)
		contentArea = m.contentView.View()
	} else {
		contentArea = "Loading..."
	}

	// ═══════════════════════════════════════════════════════════════
	// COMBINE - Vertical layout
	// ═══════════════════════════════════════════════════════════════

	// If terminal is active, show it below the content
	var view string
	if m.term.active {
		termPane := m.renderTerminalPane()
		// Reduce content area height to make room for terminal
		view = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			accentLine,
			infoBar,
			termPane,
		)
	} else {
		view = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			accentLine,
			infoBar,
			contentArea,
		)
	}

	// Pad to fill height (minus 1 for status bar)
	viewH := lipgloss.Height(view)
	targetH := m.height - 1
	if viewH < targetH {
		padding := strings.Repeat("\n", targetH-viewH)
		view = view + padding
	}

	return view
}

func (m model) renderDetailView() string {
	leftW := 18
	rightW := m.width - leftW - 3
	contentH := m.height - 2

	// Left pane - resources
	leftPane := paneStyle.Width(leftW).Height(contentH).Render(m.renderResources(contentH - 2))

	// Right pane - section tabs + content
	rightContent := m.renderRightPane(rightW-2, contentH-2)
	rightPane := focusedPaneStyle.Width(rightW).Height(contentH).Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

func (m model) renderResources(h int) string {
	var lines []string

	title := paneTitleStyle.Render("Resources")
	lines = append(lines, title, "")

	for i, res := range m.resources {
		if i >= h-2 {
			break
		}

		style := normalItem
		prefix := "  "
		if i == m.resCursor {
			style = selectedItem
			prefix = "> "
		}

		lines = append(lines, style.Render(prefix+res.name))
	}

	return strings.Join(lines, "\n")
}

func (m model) renderRightPane(w, h int) string {
	res := m.currentResource()
	if res == nil {
		return dimItem.Render("No resource selected")
	}

	// Render section tabs
	tabs := m.renderSectionTabs(w)

	// Render content
	content := m.renderContent(w, h-3)

	return lipgloss.JoinVertical(lipgloss.Left, tabs, content)
}

func (m model) renderSectionTabs(maxW int) string {
	res := m.currentResource()
	if res == nil {
		return ""
	}

	var tabs []string
	totalW := 0

	for i, sec := range res.sections {
		style := inactiveTabStyle
		if i == m.secCursor {
			style = activeTabStyle
		}

		// Truncate long titles
		title := sec.title
		if len(title) > 12 {
			title = title[:10] + ".."
		}

		// Add number hint for first 9
		label := title
		if i < 9 {
			label = fmt.Sprintf("%d %s", i+1, title)
		}

		tab := style.Render(label)
		tabW := lipgloss.Width(tab) + 2 // spacing

		if totalW+tabW > maxW-2 {
			tabs = append(tabs, dimItem.Render("..."))
			break
		}

		tabs = append(tabs, tab)
		totalW += tabW
	}

	return strings.Join(tabs, dimItem.Render(" · "))
}

func (m model) renderContent(w, h int) string {
	sec := m.currentSection()
	if sec == nil {
		return dimItem.Render("No content")
	}

	// Create renderer with custom premium style
	r, _ := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(customStyleJSON)),
		glamour.WithWordWrap(w),
	)
	rendered, _ := r.Render(sec.content)
	lines := strings.Split(rendered, "\n")

	// Bounds check
	scroll := m.scroll
	if scroll >= len(lines) {
		scroll = max(0, len(lines)-1)
	}

	end := min(scroll+h, len(lines))
	if end <= scroll {
		return ""
	}

	return strings.Join(lines[scroll:end], "\n")
}

func (m model) renderStatusBar() string {
	// Styles for the new footer
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))
	keyStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(primary).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252"))
	sepStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("240"))
	brandStyleSB := lipgloss.NewStyle().
		Background(lipgloss.Color("99")).
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Padding(0, 1)
	contextStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(secondary).
		Italic(true)

	sep := sepStyle.Render("  │  ")

	var leftContent, rightContent string

	if m.currentView == viewDashboard {
		// Left: brand
		leftContent = brandStyleSB.Render("SKITZ") + bgStyle.Render("  ") +
			contextStyle.Render("Dashboard")

		// Right: keybindings
		rightContent = keyStyle.Render("ctrl+k") + descStyle.Render(" palette") + sep +
			keyStyle.Render("↑↓") + descStyle.Render(" nav") + sep +
			keyStyle.Render("1-9") + descStyle.Render(" jump") + sep +
			keyStyle.Render("enter") + descStyle.Render(" open") + sep +
			keyStyle.Render("q") + descStyle.Render(" quit")
	} else {
		res := m.currentResource()
		sec := m.currentSection()
		breadcrumb := ""
		if res != nil {
			meta := toolMetadata[res.name]
			breadcrumb = lipgloss.NewStyle().
				Background(meta.color).
				Foreground(lipgloss.Color("255")).
				Bold(true).
				Padding(0, 1).
				Render(strings.ToUpper(res.name))
			if sec != nil {
				breadcrumb += bgStyle.Render("  ") + contextStyle.Render(sec.title)
			}
		}

		leftContent = breadcrumb

		rightContent = keyStyle.Render("↑↓/jk") + descStyle.Render(" select") + sep +
			keyStyle.Render("enter") + descStyle.Render(" run") + sep +
			keyStyle.Render("←→/hl") + descStyle.Render(" section") + sep +
			keyStyle.Render("esc") + descStyle.Render(" back")
	}

	// Calculate padding to fill width
	leftW := lipgloss.Width(leftContent)
	rightW := lipgloss.Width(rightContent)
	padW := m.width - leftW - rightW - 2
	if padW < 1 {
		padW = 1
	}
	padding := bgStyle.Render(strings.Repeat(" ", padW))

	return leftContent + padding + rightContent + bgStyle.Render(" ")
}

// formatTimeAgo formats a timestamp as a human-readable relative time
func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}

