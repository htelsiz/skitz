package app

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

	titleBlock := lipgloss.JoinVertical(lipgloss.Left,
		titleArt,
		versionStyle.Render("v0.1")+" "+descStyle.Render("Command Center"),
	)

	headerTop := lipgloss.JoinHorizontal(lipgloss.Center, biaLogo, "    ", titleBlock)

	quoteBox := quoteStyle.Render(fmt.Sprintf(`╭──────────────────────────────────╮
│  %s  │
╰──────────────────────────────────╯`, paddedQuote))

	borderStyle := lipgloss.NewStyle().Foreground(dimBorder)

	topBorder := borderStyle.Render("╔" + strings.Repeat("═", headerW-2) + "╗")
	bottomBorder := borderStyle.Render("╚" + strings.Repeat("═", headerW-2) + "╝")

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

	actionsW := (m.width * 25) / 100
	mainAreaW := m.width - actionsW - 3

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

	var sidebarLines []string

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

	headerW = mainAreaW - 2
	if headerW < 60 {
		headerW = 60
	}

	topBorder = borderStyle.Render("╔" + strings.Repeat("═", headerW-2) + "╗")
	bottomBorder = borderStyle.Render("╚" + strings.Repeat("═", headerW-2) + "╝")

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

	cardW := (mainAreaW - 6) / 3
	if cardW < 25 {
		cardW = (mainAreaW - 4) / 2
	}
	if cardW < 25 {
		cardW = mainAreaW - 4
	}

	var cards []string
	for i, res := range m.resources {
		meta := toolMetadata[res.name]
		isSelected := i == m.resCursor

		shortcut := lipgloss.NewStyle().
			Foreground(subtle).
			Render(fmt.Sprintf("[%d]", i+1))

		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(meta.color)
		if meta.status == "coming_soon" {
			nameStyle = lipgloss.NewStyle().Bold(true).Foreground(subtle)
		}
		toolName := nameStyle.Render(strings.ToUpper(res.name))

		descCardStyle := lipgloss.NewStyle().Foreground(subtle)
		if meta.status == "coming_soon" {
			descCardStyle = descCardStyle.Italic(true)
		}
		description := descCardStyle.Render(res.description)

		categoryStyle := lipgloss.NewStyle().
			Foreground(meta.color).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
		categoryTag := categoryStyle.Render(meta.category)

		cardContent := lipgloss.JoinVertical(lipgloss.Left,
			toolName+"  "+shortcut,
			description,
			"",
			categoryTag,
		)

		var cardBorderStyle lipgloss.Style
		if meta.status == "coming_soon" {
			cardBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Padding(1, 1)
		} else if isSelected {
			cardBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(meta.color).
				Padding(1, 1)
		} else {
			cardBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(dimBorder).
				Padding(1, 1)
		}

		card := cardBorderStyle.Width(cardW - 2).Render(cardContent)
		cards = append(cards, card)
	}

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

	rightContent := lipgloss.JoinVertical(lipgloss.Left, header, cardGrid)

	body := lipgloss.JoinHorizontal(lipgloss.Top, actionsPanel, " ", rightContent)

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

		cmdText := cmd.raw
		if len(cmdText) > cmdW-2 {
			cmdText = cmdText[:cmdW-5] + "..."
		}

		descText := cmd.description
		if len(descText) > descW-2 {
			descText = descText[:descW-5] + "..."
		}

		cmdPadded := cmdText + strings.Repeat(" ", max(0, cmdW-len(cmdText)))

		if isSelected {
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

			row := indicator + num + cmdStyled + " " + desc

			line := lipgloss.NewStyle().
				Foreground(accentColor).
				Render("┃") + " " +
				lipgloss.NewStyle().
					Background(lipgloss.Color("237")).
					Render(row)

			lines = append(lines, line)

		} else {
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

	viewW := m.width

	var tabRow1, tabRow2, tabRow3 []string

	for i, s := range res.sections {
		title := s.title
		if len(title) > 14 {
			title = title[:12] + ".."
		}

		var label string
		if i < 9 {
			label = fmt.Sprintf("  %d  %s  ", i+1, title)
		} else {
			label = fmt.Sprintf("  %s  ", title)
		}
		labelW := len(label)

		if i == m.secCursor {
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

			bottomBorder := lipgloss.NewStyle().
				Foreground(meta.color).
				Render("┗" + strings.Repeat("━", labelW) + "┛")

			tabRow1 = append(tabRow1, topBorder)
			tabRow2 = append(tabRow2, content)
			tabRow3 = append(tabRow3, bottomBorder)
		} else {
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

		if i < len(res.sections)-1 {
			tabRow1 = append(tabRow1, "   ")
			tabRow2 = append(tabRow2, "   ")
			tabRow3 = append(tabRow3, "───")
		}
	}

	row1 := strings.Join(tabRow1, "")
	row2 := strings.Join(tabRow2, "")
	row3 := strings.Join(tabRow3, "")

	tabBar := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().PaddingLeft(1).Render(row1),
		lipgloss.NewStyle().PaddingLeft(1).Render(row2),
		lipgloss.NewStyle().PaddingLeft(1).Render(row3),
	)

	accentLine := lipgloss.NewStyle().
		Foreground(meta.color).
		Render(strings.Repeat("─", viewW))

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

	var contentArea string

	if m.viewReady {
		contentArea = m.contentView.View()
	} else {
		contentArea = "Loading..."
	}

	var view string
	if m.term.active {
		termPane := m.renderTerminalPane()
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

	leftPane := paneStyle.Width(leftW).Height(contentH).Render(m.renderResources(contentH - 2))

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

	tabs := m.renderSectionTabs(w)

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

		title := sec.title
		if len(title) > 12 {
			title = title[:10] + ".."
		}

		label := title
		if i < 9 {
			label = fmt.Sprintf("%d %s", i+1, title)
		}

		tab := style.Render(label)
		tabW := lipgloss.Width(tab) + 2

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

	r, _ := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(customStyleJSON)),
		glamour.WithWordWrap(w),
	)
	rendered, _ := r.Render(sec.content)
	lines := strings.Split(rendered, "\n")

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
		leftContent = brandStyleSB.Render("SKITZ") + bgStyle.Render("  ") +
			contextStyle.Render("Dashboard")

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
