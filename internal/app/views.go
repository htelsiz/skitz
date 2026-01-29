package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderDashboardTabs renders the tab bar for Resources/Actions/Agents
func (m model) renderDashboardTabs(width int) string {
	tabs := []string{"RESOURCES", "ACTIONS", "AGENTS"}

	var tabParts []string

	for i, title := range tabs {
		if i == m.dashboardTab {
			// Active tab - highlighted text with underline
			tabStyle := lipgloss.NewStyle().
				Foreground(primary).
				Bold(true).
				Underline(true).
				Padding(0, 2)
			tabParts = append(tabParts, tabStyle.Render(title))
		} else {
			// Inactive tab - dim text
			tabStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 2)
			tabParts = append(tabParts, tabStyle.Render(title))
		}
	}

	tabRow := strings.Join(tabParts, "  ")

	return lipgloss.NewStyle().PaddingLeft(1).PaddingBottom(1).Render(tabRow)
}

// renderActionsTab renders the list of available actions
func (m model) renderActionsTab(width, height int) string {
	// If add resource wizard is active, show wizard form
	if m.addResourceWizard != nil && m.addResourceWizard.InputForm != nil {
		wizardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary).
			Padding(1, 2).
			Width(width - 10).
			Align(lipgloss.Center)

		stepLabels := []string{"Step 1: Name", "Step 2: Template", "Step 3: Confirm"}
		stepLabel := ""
		if m.addResourceWizard.Step < len(stepLabels) {
			stepLabel = stepLabels[m.addResourceWizard.Step]
		}

		header := lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			Render("Add Resource Wizard - " + stepLabel)

		formView := m.addResourceWizard.InputForm.View()

		wizardContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			header,
			"",
			formView,
			"",
			lipgloss.NewStyle().Foreground(subtle).Render("Press ESC to cancel"),
			"",
		)

		return lipgloss.Place(width, height,
			lipgloss.Center, lipgloss.Center,
			wizardStyle.Render(wizardContent))
	}

	// If preferences wizard is active, show wizard form
	if m.preferencesWizard != nil && m.preferencesWizard.InputForm != nil {
		wizardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondary).
			Padding(1, 2).
			Width(width - 10).
			Align(lipgloss.Center)

		var title string
		switch m.preferencesWizard.Step {
		case 0:
			title = "Preferences"
		case 1:
			switch m.preferencesWizard.Section {
			case "history":
				title = "History Settings"
			case "mcp":
				title = "MCP Servers"
			default:
				title = "Preferences"
			}
		case 2:
			title = "MCP Server Configuration"
		}

		header := lipgloss.NewStyle().
			Foreground(secondary).
			Bold(true).
			Render("⚙ " + title)

		formView := m.preferencesWizard.InputForm.View()

		wizardContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			header,
			"",
			formView,
			"",
			lipgloss.NewStyle().Foreground(subtle).Render("Press ESC to cancel"),
			"",
		)

		return lipgloss.Place(width, height,
			lipgloss.Center, lipgloss.Center,
			wizardStyle.Render(wizardContent))
	}

	// If providers wizard is active, show wizard form or test status
	if m.providersWizard != nil && (m.providersWizard.InputForm != nil || m.providersWizard.Step == 3) {
		wizardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")). // Blue for providers
			Padding(1, 2).
			Width(width - 10).
			Align(lipgloss.Center)

		var title string
		switch m.providersWizard.Step {
		case 0:
			title = "Configure Providers"
		case 1:
			title = "Select Provider Type"
		case 2:
			if strings.HasPrefix(m.providersWizard.Action, "edit:") {
				title = "Edit Provider"
			} else {
				title = "Add Provider"
			}
		case 3:
			title = "Test Connection"
		case 4:
			title = "Set Default Provider"
		}

		header := lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true).
			Render("◈ " + title)

		var contentBody string
		if m.providersWizard.Step == 3 {
			// Test connection step - show status, not form
			if m.providersWizard.Testing {
				spinner := lipgloss.NewStyle().
					Foreground(lipgloss.Color("39")).
					Render("⠋")
				contentBody = lipgloss.JoinVertical(lipgloss.Center,
					"",
					spinner+" Testing connection to "+m.providersWizard.ProviderType+"...",
					"",
					lipgloss.NewStyle().Foreground(subtle).Render("Please wait"),
					"",
				)
			} else if m.providersWizard.TestError != "" {
				errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
				contentBody = lipgloss.JoinVertical(lipgloss.Center,
					"",
					errorStyle.Render("✗ Connection Failed"),
					"",
					lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(m.providersWizard.TestError),
					"",
					lipgloss.NewStyle().Foreground(subtle).Render("Press ESC to go back and fix settings"),
					"",
				)
			} else if m.providersWizard.TestResult != "" {
				successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true)
				contentBody = lipgloss.JoinVertical(lipgloss.Center,
					"",
					successStyle.Render("✓ "+m.providersWizard.TestResult),
					"",
					lipgloss.NewStyle().Foreground(subtle).Render("Provider saved successfully!"),
					"",
				)
			}
		} else {
			contentBody = m.providersWizard.InputForm.View()
		}

		wizardContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			header,
			"",
			contentBody,
			"",
			lipgloss.NewStyle().Foreground(subtle).Render("Press ESC to cancel"),
			"",
		)

		return lipgloss.Place(width, height,
			lipgloss.Center, lipgloss.Center,
			wizardStyle.Render(wizardContent))
	}

	// If run agent wizard is active, show wizard form
	if m.runAgentWizard != nil && m.runAgentWizard.InputForm != nil {
		wizardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("220")).
			Padding(1, 2).
			Width(width - 10).
			Align(lipgloss.Center)

		stepLabels := []string{"Select Provider", "Select Runtime", "Configure Agent", "Confirm"}
		stepLabel := ""
		if m.runAgentWizard.Step < len(stepLabels) {
			stepLabel = stepLabels[m.runAgentWizard.Step]
		}

		header := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true).
			Render("⚡ Run Agent - " + stepLabel)

		formView := m.runAgentWizard.InputForm.View()

		wizardContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			header,
			"",
			formView,
			"",
			lipgloss.NewStyle().Foreground(subtle).Render("Press ESC to cancel"),
			"",
		)

		return lipgloss.Place(width, height,
			lipgloss.Center, lipgloss.Center,
			wizardStyle.Render(wizardContent))
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(secondary).
		Bold(true)

	// Convert action items to CardItems
	var items []CardItem
	for i, action := range m.actionItems {
		items = append(items, CardItem{
			Title:    action.Icon + "  " + action.Name,
			Subtitle: action.Description,
			TagColor: primary,
			Shortcut: i + 1,
		})
	}

	cardGrid := CardGrid(items, width, m.actionCursor)

	// Info text
	infoStyle := lipgloss.NewStyle().
		Foreground(subtle).
		Italic(true)

	info := infoStyle.Render("Select an action and press Enter to start")

	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		titleStyle.Render("Available Actions"),
		"",
		cardGrid,
		"",
		info,
	)

	return lipgloss.NewStyle().Padding(0, 2).Render(content)
}

// renderAgentsTab renders the agents tab with active agents and history
func (m model) renderAgentsTab(width, height int) string {
	// Detail view
	if m.agentViewMode == 1 && m.selectedAgentIdx < len(m.agentHistory) {
		return m.renderAgentDetail(width, height)
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(subtle).
		Italic(true)

	var sections []string

	// Active Agents section
	if len(m.activeAgents) > 0 {
		activeTitle := titleStyle.Render("Active Agents")
		var activeItems []CardItem
		for i, agent := range m.activeAgents {
			elapsed := time.Since(agent.StartTime).Round(time.Second)
			activeItems = append(activeItems, CardItem{
				Title:       agent.Name,
				Subtitle:    fmt.Sprintf("%s | %s | %s", agent.Provider, agent.Runtime, elapsed),
				Tag:         "RUNNING",
				TagColor:    lipgloss.Color("220"),
				BorderColor: lipgloss.Color("220"),
				Shortcut:    i + 1,
			})
		}
		activeGrid := CardGrid(activeItems, width, -1) // No selection in active agents
		sections = append(sections, "", activeTitle, "", activeGrid)
	}

	// Agent History section
	historyTitle := titleStyle.Render("Agent History")

	if len(m.agentHistory) == 0 {
		emptyMsg := subtitleStyle.Render("No agent runs yet. Use Actions > Run Agent to start.")
		sections = append(sections, "", historyTitle, "", emptyMsg)
	} else {
		var historyItems []CardItem
		for i, entry := range m.agentHistory {
			statusIcon := "+"
			tagColor := lipgloss.Color("114")
			if !entry.Success {
				statusIcon = "x"
				tagColor = lipgloss.Color("196")
			}

			subtitle := formatTimeAgo(entry.Timestamp)
			if entry.Duration > 0 {
				subtitle += fmt.Sprintf(" | %dms", entry.Duration)
			}
			if entry.Runtime != "" {
				subtitle += " | " + entry.Runtime
			}

			action := entry.Action
			if len(action) > 40 {
				action = action[:37] + "..."
			}

			historyItems = append(historyItems, CardItem{
				Title:       entry.Agent,
				Subtitle:    subtitle,
				Tag:         statusIcon,
				TagColor:    tagColor,
				BorderColor: tagColor,
				Shortcut:    i + 1,
			})
		}

		// Calculate cursor position for history (offset by active agents count)
		historyCursor := m.agentCursor - len(m.activeAgents)
		if historyCursor < 0 {
			historyCursor = -1
		}
		historyGrid := CardGrid(historyItems, width, historyCursor)
		sections = append(sections, "", historyTitle, "", historyGrid)
	}

	// Info text
	infoStyle := lipgloss.NewStyle().
		Foreground(subtle).
		Italic(true)

	info := infoStyle.Render("Select an entry and press Enter to view details")
	sections = append(sections, "", info)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return lipgloss.NewStyle().Padding(0, 2).Render(content)
}

// renderAgentDetail renders the detail view for a selected agent interaction
func (m model) renderAgentDetail(width, height int) string {
	if m.selectedAgentIdx >= len(m.agentHistory) {
		return ""
	}

	entry := m.agentHistory[m.selectedAgentIdx]

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(subtle)

	// Status icon
	statusIcon := "+"
	statusColor := lipgloss.Color("114")
	statusText := "Success"
	if !entry.Success {
		statusIcon = "x"
		statusColor = lipgloss.Color("196")
		statusText = "Failed"
	}
	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)

	// Build all content lines first
	var allLines []string

	// Header with name and status
	header := titleStyle.Render(entry.Agent) + "  " + statusStyle.Render(statusIcon+" "+statusText)
	allLines = append(allLines, header, "")

	// Metadata grid
	allLines = append(allLines, labelStyle.Render("Provider: ")+valueStyle.Render(entry.Provider))
	allLines = append(allLines, labelStyle.Render("Runtime:  ")+valueStyle.Render(entry.Runtime))
	allLines = append(allLines, labelStyle.Render("Time:     ")+valueStyle.Render(entry.Timestamp.Format("2006-01-02 15:04:05")))
	if entry.Duration > 0 {
		allLines = append(allLines, labelStyle.Render("Duration: ")+valueStyle.Render(fmt.Sprintf("%dms", entry.Duration)))
	}
	allLines = append(allLines, "")

	// Input/Task section
	allLines = append(allLines, labelStyle.Render("Task/Prompt:"))
	inputLines := strings.Split(entry.Input, "\n")
	for _, line := range inputLines {
		allLines = append(allLines, "  "+valueStyle.Render(line))
	}
	allLines = append(allLines, "")

	// Output section - include all lines
	allLines = append(allLines, labelStyle.Render("Output:"))
	if entry.Output == "" {
		allLines = append(allLines, dimStyle.Render("  (no output)"))
	} else {
		outputLines := strings.Split(entry.Output, "\n")
		for _, line := range outputLines {
			if len(line) > width-14 {
				line = line[:width-17] + "..."
			}
			allLines = append(allLines, "  "+valueStyle.Render(line))
		}
	}

	// Calculate visible area (leave room for hints and border)
	visibleHeight := height - 6
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	// Apply scroll offset
	totalLines := len(allLines)
	maxScroll := totalLines - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	scrollOffset := m.agentDetailScroll
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	// Get visible lines
	endIdx := scrollOffset + visibleHeight
	if endIdx > totalLines {
		endIdx = totalLines
	}

	var visibleLines []string
	if scrollOffset < totalLines {
		visibleLines = allLines[scrollOffset:endIdx]
	}

	// Add scroll indicator if needed
	scrollInfo := ""
	if totalLines > visibleHeight {
		scrollInfo = dimStyle.Render(fmt.Sprintf(" [%d-%d of %d] ", scrollOffset+1, endIdx, totalLines))
	}

	// Key hints
	hintStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")).
		Bold(true)

	hints := hintStyle.Render(
		keyStyle.Render("j/k") + dimStyle.Render(" scroll  ") +
			keyStyle.Render("esc") + dimStyle.Render(" back  ") +
			keyStyle.Render("ctrl+y") + dimStyle.Render(" copy") + scrollInfo,
	)

	visibleLines = append(visibleLines, "", hints)

	content := lipgloss.JoinVertical(lipgloss.Left, visibleLines...)

	// Wrap in a bordered box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("213")).
		Padding(0, 2).
		Width(width - 6)

	return lipgloss.NewStyle().Padding(0, 2).Render(boxStyle.Render(content))
}

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

	// Providers section
	sidebarLines = append(sidebarLines, "", actionsTitleStyle.Render("◈ Providers"))
	if len(m.config.AI.Providers) == 0 {
		sidebarLines = append(sidebarLines, actionDimStyle.Render("  No providers"))
		sidebarLines = append(sidebarLines, actionDimStyle.Render("  Actions → Configure"))
	} else {
		providerColor := lipgloss.Color("39")
		for _, p := range m.config.AI.Providers {
			icon := "○"
			statusStyle := actionDimStyle
			if p.Enabled {
				icon = "●"
				statusStyle = lipgloss.NewStyle().Foreground(providerColor)
			}
			if p.Name == m.config.AI.DefaultProvider {
				icon = "◆"
				statusStyle = lipgloss.NewStyle().Foreground(providerColor).Bold(true)
			}

			name := p.Name
			if len(name) > 16 {
				name = name[:13] + "..."
			}
			sidebarLines = append(sidebarLines, statusStyle.Render(fmt.Sprintf("  %s %s", icon, name)))
		}

		// Show default agent
		if m.config.AI.DefaultProvider != "" {
			sidebarLines = append(sidebarLines, "")
			agentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
			sidebarLines = append(sidebarLines, agentStyle.Render("  ⚡ skitz-agent"))
			sidebarLines = append(sidebarLines, actionDimStyle.Render(fmt.Sprintf("    using %s", m.config.AI.DefaultProvider)))
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

	// Convert resources to CardItems
	var resourceItems []CardItem
	for i, res := range m.resources {
		meta := toolMetadata[res.name]
		borderColor := dimBorder
		if meta.status == "coming_soon" {
			borderColor = lipgloss.Color("238")
		}
		resourceItems = append(resourceItems, CardItem{
			Title:       strings.ToUpper(res.name),
			Subtitle:    res.description,
			Tag:         meta.category,
			TagColor:    meta.color,
			BorderColor: borderColor,
			Shortcut:    i + 1,
		})
	}

	cardGrid := CardGrid(resourceItems, mainAreaW, m.resCursor)

	// Render tab bar
	tabBar := m.renderDashboardTabs(mainAreaW)

	// Conditional content based on selected tab
	var tabContent string
	remainingH := contentH - lipgloss.Height(header) - lipgloss.Height(tabBar) - 2
	switch m.dashboardTab {
	case 0:
		// Resources tab - show resource cards
		tabContent = cardGrid
	case 1:
		// Actions tab - show actions list
		tabContent = m.renderActionsTab(mainAreaW, remainingH)
	case 2:
		// Agents tab - show agent history and active agents
		tabContent = m.renderAgentsTab(mainAreaW, remainingH)
	}

	rightContent := lipgloss.JoinVertical(lipgloss.Left, header, tabBar, tabContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, actionsPanel, " ", rightContent)

	bodyH := lipgloss.Height(body)
	if bodyH < contentH {
		body = body + strings.Repeat("\n", contentH-bodyH)
	}

	// If delete resource wizard is active, render it as an overlay (same style as other wizards)
	if m.deleteResourceWizard != nil && m.deleteResourceWizard.InputForm != nil {
		wizardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 2).
			Align(lipgloss.Center)

		header := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render("Delete Resource")

		formView := m.deleteResourceWizard.InputForm.View()

		wizardContent := lipgloss.JoinVertical(lipgloss.Center,
			"",
			header,
			"",
			formView,
			"",
			lipgloss.NewStyle().Foreground(subtle).Render("Press ESC to cancel"),
			"",
		)

		body = lipgloss.Place(m.width-4, contentH,
			lipgloss.Center, lipgloss.Center,
			wizardStyle.Render(wizardContent))
	}

	return body
}

// renderCommandList renders an interactive command list with selection highlighting.
func (m model) renderCommandList(width int, accentColor lipgloss.Color) string {
	if len(m.commands) == 0 {
		return lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true).
			Padding(2, 4).
			Render("No runnable commands in this section")
	}

	// Header block
	headerLabel := lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("COMMANDS")
	headerCount := lipgloss.NewStyle().Foreground(subtle).Render(fmt.Sprintf("  %d available", len(m.commands)))
	divider := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("─", width-6))
	header := lipgloss.NewStyle().PaddingLeft(2).MarginBottom(1).Render(
		lipgloss.JoinVertical(lipgloss.Left, headerLabel+headerCount, divider),
	)

	// Column widths
	prefixW := 8 // " ▶  1  " or "     1  "
	sepW := 3    // " │ "
	availableW := width - prefixW - sepW - 4
	cmdW := (availableW * 55) / 100
	descW := availableW - cmdW
	if cmdW < 28 {
		cmdW = 28
	}
	if descW < 12 {
		descW = 12
	}

	// Command rows
	var rows []string
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

		highlighted := highlightShellCommand(cmdText)
		cmdPad := max(0, cmdW-lipgloss.Width(highlighted))

		var inputBadge string
		if cmd.inputVar != "" {
			inputBadge = lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")).
				Render(" {{" + cmd.inputVar + "}}")
		}

		if isSelected {
			arrow := lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(" ▶ ")
			num := lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(fmt.Sprintf("%-3d", i+1))
			sep := lipgloss.NewStyle().Foreground(accentColor).Render(" │ ")
			cmdStyled := lipgloss.NewStyle().Background(lipgloss.Color("239")).Bold(true).
				Render(" " + highlighted + strings.Repeat(" ", cmdPad) + " ")
			desc := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true).Render(descText)

			row := arrow + num + sep + cmdStyled + inputBadge + "  " + desc
			rowW := lipgloss.Width(row)
			if rowW < width-3 {
				row += strings.Repeat(" ", width-3-rowW)
			}

			bar := lipgloss.NewStyle().Foreground(accentColor).Background(lipgloss.Color("236")).Render("┃")
			rows = append(rows, bar+lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(row))
		} else {
			num := lipgloss.NewStyle().Foreground(subtle).Render(fmt.Sprintf("     %-3d", i+1))
			sep := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(" │ ")
			cmdStyled := lipgloss.NewStyle().Background(lipgloss.Color("235")).
				Render(" " + highlighted + strings.Repeat(" ", cmdPad) + " ")
			desc := lipgloss.NewStyle().Foreground(subtle).Render(descText)

			rows = append(rows, " "+num+sep+cmdStyled+inputBadge+"  "+desc)
		}
	}

	commandBlock := lipgloss.NewStyle().MarginTop(1).Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, header, commandBlock)
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

	// Render Ask AI panel if active
	var askPanelView string
	if m.askPanel != nil && m.askPanel.Active {
		askPanelView = m.renderAskPanel(viewW)
	}

	var view string
	if m.askPanel != nil && m.askPanel.Active {
		view = lipgloss.JoinVertical(lipgloss.Left,
			tabBar,
			accentLine,
			infoBar,
			askPanelView,
		)
	} else if m.term.active {
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

// renderAskPanel renders the AI ask panel
func (m model) renderAskPanel(width int) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(width - 4)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("255")).
		Padding(0, 1).
		Width(width - 12)

	hintStyle := lipgloss.NewStyle().
		Foreground(subtle).
		Italic(true)

	keyHintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	var lines []string

	// Title
	lines = append(lines, titleStyle.Render("◈ Ask AI about "+m.currentResource().name))
	lines = append(lines, "")

	// Input field
	inputContent := m.askPanel.Input
	if m.askPanel.Loading {
		inputContent = m.askPanel.Input + " ..."
	}
	cursor := "▌"
	if m.askPanel.Loading {
		cursor = ""
	}
	lines = append(lines, inputStyle.Render("> "+inputContent+cursor))
	lines = append(lines, "")

	// Response or loading
	if m.askPanel.Loading {
		lines = append(lines, hintStyle.Render("Thinking..."))
	} else if m.askPanel.Error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		lines = append(lines, errorStyle.Render("Error: "+m.askPanel.Error))
	} else if m.askPanel.Response != "" {
		responseStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Width(width - 12)
		lines = append(lines, responseStyle.Render(m.askPanel.Response))

		// Show generated command if available
		if m.askPanel.GeneratedCmd != "" {
			lines = append(lines, "")
			cmdStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("114")).
				Bold(true).
				Padding(0, 1)
			lines = append(lines, cmdStyle.Render("$ "+m.askPanel.GeneratedCmd))
			lines = append(lines, "")
			lines = append(lines,
				keyHintStyle.Render("ctrl+r")+hintStyle.Render(" run  ")+
					keyHintStyle.Render("ctrl+a")+hintStyle.Render(" add to resource"))
		}
	}

	lines = append(lines, "")

	// Hints
	lines = append(lines,
		keyHintStyle.Render("enter")+hintStyle.Render(" ask  ")+
			keyHintStyle.Render("ctrl+g")+hintStyle.Render(" generate cmd  ")+
			keyHintStyle.Render("esc")+hintStyle.Render(" close"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return panelStyle.Render(content)
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
		tabNames := []string{"Resources", "Actions", "Agents"}
		tabName := tabNames[m.dashboardTab]
		leftContent = brandStyleSB.Render("SKITZ") + bgStyle.Render("  ") +
			contextStyle.Render("Dashboard › "+tabName)

		rightContent = keyStyle.Render("tab") + descStyle.Render(" switch") + sep +
			keyStyle.Render("ctrl+k") + descStyle.Render(" palette") + sep +
			keyStyle.Render("↑↓") + descStyle.Render(" nav") + sep +
			keyStyle.Render("e") + descStyle.Render(" edit") + sep +
			keyStyle.Render("d") + descStyle.Render(" delete") + sep +
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

		rightContent = keyStyle.Render("a") + descStyle.Render(" ask AI") + sep +
			keyStyle.Render("↑↓") + descStyle.Render(" select") + sep +
			keyStyle.Render("enter") + descStyle.Render(" run") + sep +
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
