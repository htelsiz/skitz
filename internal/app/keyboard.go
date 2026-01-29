package app

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// handleKeyMsg is the main keyboard event dispatcher
func (m *model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	// Terminal focus toggle
	if keyStr == "f1" && m.term.active {
		m.term.focused = !m.term.focused
		return m, nil
	}

	// Forward keys to terminal if focused
	if m.term.active && m.term.focused && !m.term.exited {
		return m, m.sendKeyToTerminal(msg)
	}

	// Close terminal if not focused
	if keyStr == "esc" && m.term.active && !m.term.focused {
		m.closeTerminal()
		return m, nil
	}

	// Command palette handling
	if m.palette.State != PaletteStateIdle {
		return m.handlePaletteKeys(msg)
	}

	// Open palette
	if keyStr == "ctrl+k" {
		m.openPalette()
		return m, nil
	}

	// Ask AI panel handling
	if m.askPanel != nil && m.askPanel.Active {
		return m.handleAskPanelKeys(msg)
	}

	// Detail view handling
	if m.currentView == viewDetail && m.viewReady {
		return m.handleDetailViewKeys(msg)
	}

	// Wizard form handling
	if m.hasActiveWizard() {
		cmd := m.handleWizardKeys(msg)
		return m, cmd
	}

	// Dashboard navigation
	return m.handleDashboardKeys(msg)
}

// handlePaletteKeys handles keyboard input for the command palette
func (m *model) handlePaletteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	// Handle parameter collection form
	if m.palette.State == PaletteStateCollectingParams && m.palette.InputForm != nil {
		if keyStr == "esc" {
			m.palette.State = PaletteStateSearching
			m.palette.InputForm = nil
			m.palette.PendingTool = nil
			m.palette.WizardState = nil
			return m, nil
		}

		form, cmd := m.palette.InputForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.palette.InputForm = f

			if f.State == huh.StateCompleted {
				if m.palette.WizardState != nil {
					return m, m.handleWizardSubmit()
				}
				return m, m.handleParameterSubmit()
			}
		}
		return m, cmd
	}

	// Handle palette states
	switch keyStr {
	case "esc", "ctrl+k":
		switch m.palette.State {
		case PaletteStateExecuting:
			return m, nil
		case PaletteStateAIInput:
			m.palette.State = PaletteStateSearching
			m.palette.PendingTool = nil
			m.palette.Query = ""
			return m, nil
		case PaletteStateShowingResult:
			m.closePalette()
			return m, nil
		default:
			m.closePalette()
			m.term.active = false
			m.term.staticOutput = ""
			m.term.staticTitle = ""
			return m, nil
		}

	case "enter":
		switch m.palette.State {
		case PaletteStateExecuting:
			return m, nil

		case PaletteStateAIInput:
			task := strings.TrimSpace(m.palette.Query)
			if task == "" {
				return m, m.showNotification("⚠️", "Please describe what you want the AI to do", "warning")
			}

			pt := m.palette.PendingTool
			if pt != nil {
				pt.AITask = task
				m.palette.State = PaletteStateExecuting
				m.palette.LoadingText = "🤖 AI is determining parameters and executing..."
				return m, m.executeMCPToolWithAIAgent(pt)
			}
			return m, nil

		case PaletteStateSearching:
			if len(m.palette.Filtered) > 0 && m.palette.Cursor < len(m.palette.Filtered) {
				item := m.palette.Filtered[m.palette.Cursor]
				m.term.staticOutput = ""
				m.term.staticTitle = ""

				if item.MCPTool != nil {
					return m, m.startMCPToolInput(item)
				}

				if item.Handler != nil {
					cmd := item.Handler(m)
					return m, cmd
				}
			}
			return m, nil

		case PaletteStateShowingResult:
			m.closePalette()
			return m, nil

		default:
			return m, nil
		}

	case "ctrl+a":
		if m.palette.State != PaletteStateSearching {
			return m, nil
		}

		if len(m.palette.Filtered) > 0 && m.palette.Cursor < len(m.palette.Filtered) {
			item := m.palette.Filtered[m.palette.Cursor]
			if item.MCPTool != nil {
				return m, m.startMCPToolWithAI(item)
			}
		}
		return m, nil

	case "up", "ctrl+p":
		if m.palette.State != PaletteStateSearching {
			return m, nil
		}
		if m.palette.Cursor > 0 {
			m.palette.Cursor--
		} else {
			m.palette.Cursor = max(0, len(m.palette.Filtered)-1)
		}
		return m, nil

	case "down", "ctrl+n":
		if m.palette.State != PaletteStateSearching {
			return m, nil
		}
		if m.palette.Cursor < len(m.palette.Filtered)-1 {
			m.palette.Cursor++
		} else {
			m.palette.Cursor = 0
		}
		return m, nil

	case "backspace":
		if m.palette.State != PaletteStateSearching && m.palette.State != PaletteStateAIInput {
			return m, nil
		}
		if len(m.palette.Query) > 0 {
			m.palette.Query = m.palette.Query[:len(m.palette.Query)-1]
			if m.palette.State == PaletteStateSearching {
				m.palette.Filtered = filterPaletteItems(m.palette.Items, m.palette.Query)
				m.palette.Cursor = 0
			}
		}
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	default:
		if m.palette.State != PaletteStateSearching && m.palette.State != PaletteStateAIInput {
			return m, nil
		}

		if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] < 127 {
			m.palette.Query += keyStr
			if m.palette.State == PaletteStateSearching {
				m.palette.Filtered = filterPaletteItems(m.palette.Items, m.palette.Query)
				m.palette.Cursor = 0
			}
		} else if keyStr == "space" {
			m.palette.Query += " "
			if m.palette.State == PaletteStateSearching {
				m.palette.Filtered = filterPaletteItems(m.palette.Items, m.palette.Query)
				m.palette.Cursor = 0
			}
		}
		return m, nil
	}
}

// handleAskPanelKeys handles keyboard input for the Ask AI panel
func (m *model) handleAskPanelKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	switch keyStr {
	case "esc":
		m.askPanel = nil
		return m, nil
	case "enter":
		if m.askPanel.Input != "" && !m.askPanel.Loading {
			return m, m.submitAskPanel()
		}
		return m, nil
	case "backspace":
		if len(m.askPanel.Input) > 0 {
			m.askPanel.Input = m.askPanel.Input[:len(m.askPanel.Input)-1]
		}
		return m, nil
	case "ctrl+g":
		// Generate command mode
		if m.askPanel.Input != "" && !m.askPanel.Loading {
			return m, m.submitGenerateCommand()
		}
		return m, nil
	case "ctrl+r":
		// Run generated command
		if m.askPanel.GeneratedCmd != "" {
			cmd := m.askPanel.GeneratedCmd
			m.askPanel = nil
			return m, m.runCommand(CommandSpec{
				Command: cmd,
				Mode:    CommandEmbedded,
			})
		}
		return m, nil
	case "ctrl+a":
		// Add generated command to resource
		if m.askPanel.GeneratedCmd != "" {
			return m, m.addCommandToResource(m.askPanel.GeneratedCmd)
		}
		return m, nil
	default:
		// Type into input
		if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] < 127 {
			m.askPanel.Input += keyStr
		} else if keyStr == "space" {
			m.askPanel.Input += " "
		}
		return m, nil
	}
}

// handleDetailViewKeys handles keyboard input in the detail view
func (m *model) handleDetailViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	keyStr := msg.String()

	switch keyStr {
	case "q":
		m.currentView = viewDashboard
		m.viewReady = false
		m.secCursor = 0
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.currentView = viewDashboard
		m.viewReady = false
		m.secCursor = 0
		return m, nil

	case "tab", "shift+tab":
		res := m.currentResource()
		if res != nil {
			if msg.String() == "tab" {
				if m.secCursor < len(res.sections)-1 {
					m.secCursor++
				} else {
					m.secCursor = 0
				}
			} else {
				if m.secCursor > 0 {
					m.secCursor--
				} else {
					m.secCursor = len(res.sections) - 1
				}
			}
			m.cmdCursor = 0
			m.updateViewportContent()
		}
		return m, nil

	case "left", "h":
		if m.secCursor > 0 {
			m.secCursor--
			m.cmdCursor = 0
			m.updateViewportContent()
		}
		return m, nil

	case "right", "l":
		res := m.currentResource()
		if res != nil && m.secCursor < len(res.sections)-1 {
			m.secCursor++
			m.cmdCursor = 0
			m.updateViewportContent()
		}
		return m, nil

	case "up", "k":
		if len(m.commands) > 0 {
			if m.cmdCursor > 0 {
				m.cmdCursor--
			} else {
				m.cmdCursor = len(m.commands) - 1
			}
			m.refreshCommandListDisplay()
		}
		return m, nil

	case "down", "j":
		if len(m.commands) > 0 {
			if m.cmdCursor < len(m.commands)-1 {
				m.cmdCursor++
			} else {
				m.cmdCursor = 0
			}
			m.refreshCommandListDisplay()
		}
		return m, nil

	case "a":
		// Open Ask AI panel
		if m.config.AI.DefaultProvider == "" {
			return m, m.showNotification("!", "Configure a provider first", "warning")
		}
		m.askPanel = &AskPanel{
			Active: true,
			Input:  "",
		}
		return m, nil

	case "ctrl+y":
		if len(m.commands) > 0 && m.cmdCursor < len(m.commands) {
			cmdText := m.commands[m.cmdCursor].raw
			if err := clipboard.WriteAll(cmdText); err != nil {
				return m, m.showNotification("!", "Copy failed: "+err.Error(), "error")
			}
			displayCmd := cmdText
			if len(displayCmd) > 25 {
				displayCmd = displayCmd[:22] + "..."
			}
			return m, m.showNotification("", "Copied: "+displayCmd, "success")
		}
		return m, nil

	case "enter":
		if len(m.commands) > 0 && m.cmdCursor < len(m.commands) {
			cmd := m.commands[m.cmdCursor]
			finalCmd := cmd.cmd
			if cmd.inputVar != "" {
				var inputValue string

				inputField := huh.NewInput().
					Title(fmt.Sprintf("Enter %s:", cmd.inputVar)).
					Placeholder(cmd.inputVar).
					Value(&inputValue)

				form := huh.NewForm(huh.NewGroup(inputField)).
					WithTheme(huh.ThemeCatppuccin())

				if err := form.Run(); err != nil || inputValue == "" {
					return m, nil
				}

				finalCmd = strings.Replace(finalCmd, "{{INPUT}}", inputValue, -1)
			}

			mode := CommandEmbedded
			if isInteractiveCommand(finalCmd) {
				mode = CommandInteractive
			}

			return m, m.runCommand(CommandSpec{
				Command: finalCmd,
				Mode:    mode,
			})
		}
		return m, nil

	case "ctrl+d", "pgdown":
		m.contentView.HalfViewDown()
		return m, nil

	case "ctrl+u", "pgup":
		m.contentView.HalfViewUp()
		return m, nil

	case "g":
		m.contentView.GotoTop()
		return m, nil

	case "G":
		m.contentView.GotoBottom()
		return m, nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		res := m.currentResource()
		if res != nil && idx < len(res.sections) {
			m.secCursor = idx
			m.cmdCursor = 0
			m.updateViewportContent()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.contentView, cmd = m.contentView.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// hasActiveWizard returns true if any wizard is currently active
func (m *model) hasActiveWizard() bool {
	return (m.addResourceWizard != nil && m.addResourceWizard.InputForm != nil) ||
		(m.runAgentWizard != nil && m.runAgentWizard.InputForm != nil) ||
		(m.preferencesWizard != nil && m.preferencesWizard.InputForm != nil) ||
		(m.providersWizard != nil && m.providersWizard.InputForm != nil) ||
		(m.deleteResourceWizard != nil && m.deleteResourceWizard.InputForm != nil)
}

// handleWizardKeys handles keyboard input for wizard forms
func (m *model) handleWizardKeys(msg tea.KeyMsg) tea.Cmd {
	keyStr := msg.String()

	// Handle Add Resource wizard form if active
	if m.addResourceWizard != nil && m.addResourceWizard.InputForm != nil {
		if keyStr == "esc" {
			m.addResourceWizard = nil
			return nil
		}

		form, cmd := m.addResourceWizard.InputForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.addResourceWizard.InputForm = f
			if f.State == huh.StateCompleted {
				return m.nextAddResourceStep()
			}
		}
		return cmd
	}

	// Handle Run Agent wizard form if active
	if m.runAgentWizard != nil && m.runAgentWizard.InputForm != nil {
		if keyStr == "esc" {
			m.runAgentWizard = nil
			return nil
		}

		form, cmd := m.runAgentWizard.InputForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.runAgentWizard.InputForm = f
			if f.State == huh.StateCompleted {
				return m.nextRunAgentStep()
			}
		}
		return cmd
	}

	// Handle Preferences wizard form if active
	if m.preferencesWizard != nil && m.preferencesWizard.InputForm != nil {
		if keyStr == "esc" {
			m.preferencesWizard = nil
			return nil
		}

		form, cmd := m.preferencesWizard.InputForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.preferencesWizard.InputForm = f
			if f.State == huh.StateCompleted {
				return m.nextPreferencesStep()
			}
		}
		return cmd
	}

	// Handle Providers wizard form if active
	if m.providersWizard != nil && m.providersWizard.InputForm != nil {
		if keyStr == "esc" {
			m.providersWizard = nil
			return nil
		}

		form, cmd := m.providersWizard.InputForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.providersWizard.InputForm = f
			if f.State == huh.StateCompleted {
				return m.nextProvidersStep()
			}
		}
		return cmd
	}

	// Handle Delete Resource wizard form if active
	if m.deleteResourceWizard != nil && m.deleteResourceWizard.InputForm != nil {
		if keyStr == "esc" {
			m.deleteResourceWizard = nil
			return nil
		}

		form, cmd := m.deleteResourceWizard.InputForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.deleteResourceWizard.InputForm = f
			if f.State == huh.StateCompleted {
				return m.confirmDeleteResource()
			}
		}
		return cmd
	}

	return nil
}

// handleAgentsTabKeys handles keyboard input for the Agents tab
func (m *model) handleAgentsTabKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	// In detail view
	if m.agentViewMode == 1 {
		switch keyStr {
		case "esc", "q":
			m.agentViewMode = 0
			m.agentDetailScroll = 0
			return m, nil
		case "ctrl+y":
			// Copy output to clipboard
			if m.selectedAgentIdx < len(m.agentHistory) {
				output := m.agentHistory[m.selectedAgentIdx].Output
				if err := clipboard.WriteAll(output); err != nil {
					return m, m.showNotification("!", "Copy failed: "+err.Error(), "error")
				}
				return m, m.showNotification("", "Output copied to clipboard", "success")
			}
			return m, nil
		case "j", "down":
			m.agentDetailScroll++
			return m, nil
		case "k", "up":
			if m.agentDetailScroll > 0 {
				m.agentDetailScroll--
			}
			return m, nil
		case "g":
			m.agentDetailScroll = 0
			return m, nil
		case "G":
			m.agentDetailScroll = 9999 // Will be clamped in render
			return m, nil
		case "ctrl+d", "pgdown":
			m.agentDetailScroll += 10
			return m, nil
		case "ctrl+u", "pgup":
			m.agentDetailScroll -= 10
			if m.agentDetailScroll < 0 {
				m.agentDetailScroll = 0
			}
			return m, nil
		}
		return m, nil
	}

	// In list view
	totalItems := len(m.activeAgents) + len(m.agentHistory)

	switch keyStr {
	case "up", "k":
		if m.agentCursor > 0 {
			m.agentCursor--
		}
		return m, nil

	case "down", "j":
		if m.agentCursor < totalItems-1 {
			m.agentCursor++
		}
		return m, nil

	case "enter":
		// Can only view details for history items, not active agents
		historyIdx := m.agentCursor - len(m.activeAgents)
		if historyIdx >= 0 && historyIdx < len(m.agentHistory) {
			m.selectedAgentIdx = historyIdx
			m.agentViewMode = 1
		}
		return m, nil

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(keyStr[0] - '1')
		if idx < totalItems {
			m.agentCursor = idx
			// If it's a history item, go to detail view
			historyIdx := idx - len(m.activeAgents)
			if historyIdx >= 0 && historyIdx < len(m.agentHistory) {
				m.selectedAgentIdx = historyIdx
				m.agentViewMode = 1
			}
		}
		return m, nil
	}

	return m, nil
}

// handleDashboardKeys handles keyboard input in the dashboard view
func (m *model) handleDashboardKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Route to agents tab handler if on agents tab (except for global keys)
	if m.dashboardTab == 2 {
		keyStr := msg.String()
		// Handle global keys first
		switch keyStr {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "shift+tab":
			if keyStr == "tab" {
				m.dashboardTab = (m.dashboardTab + 1) % 3
			} else {
				m.dashboardTab = (m.dashboardTab + 2) % 3
			}
			m.agentCursor = 0
			m.agentViewMode = 0
			return m, nil
		default:
			return m.handleAgentsTabKeys(msg)
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab", "shift+tab":
		// Switch between Resources, Actions, and Agents tabs
		if msg.String() == "tab" {
			m.dashboardTab = (m.dashboardTab + 1) % 3
		} else {
			m.dashboardTab = (m.dashboardTab + 2) % 3 // Go backwards
		}
		// Reset cursors when switching tabs
		m.agentCursor = 0
		m.agentViewMode = 0
		return m, nil

	case "enter":
		if m.dashboardTab == 0 {
			// Resources tab - open resource detail
			m.currentView = viewDetail
			m.secCursor = 0
			m.initViewComponents()
		} else if m.dashboardTab == 1 {
			// Actions tab - execute action
			if len(m.actionItems) > 0 && m.actionCursor < len(m.actionItems) {
				action := m.actionItems[m.actionCursor]
				if action.Handler != nil {
					return m, action.Handler(m)
				}
			}
		}
		return m, nil

	case "e":
		// Edit selected resource in external editor
		if m.dashboardTab == 0 {
			return m, m.editResource()
		}
		return m, nil

	case "d":
		// Delete selected resource
		if m.dashboardTab == 0 {
			return m, m.startDeleteResourceWizard()
		}
		return m, nil

	case "up", "k":
		if m.dashboardTab == 0 {
			if m.resCursor > 0 {
				m.resCursor--
			}
		} else if m.dashboardTab == 1 {
			if m.actionCursor > 0 {
				m.actionCursor--
			}
		}

	case "down", "j":
		if m.dashboardTab == 0 {
			if m.resCursor < len(m.resources)-1 {
				m.resCursor++
			}
		} else if m.dashboardTab == 1 {
			if m.actionCursor < len(m.actionItems)-1 {
				m.actionCursor++
			}
		}

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		if m.dashboardTab == 0 {
			if idx < len(m.resources) {
				m.resCursor = idx
				m.currentView = viewDetail
				m.secCursor = 0
				m.initViewComponents()
			}
		} else if m.dashboardTab == 1 {
			if idx < len(m.actionItems) {
				m.actionCursor = idx
				action := m.actionItems[m.actionCursor]
				if action.Handler != nil {
					return m, action.Handler(m)
				}
			}
		}
	}

	return m, nil
}
