package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/htelsiz/skitz/internal/config"
	"github.com/htelsiz/skitz/internal/resources"
)

// QuickAction represents an action that can be triggered
type QuickAction struct {
	ID       string
	Name     string
	Icon     string
	Shortcut string
	Builtin  bool
	Handler  func(m *model) (tea.Cmd, bool)
	Command  string
}

// buildQuickActions builds quick actions from config
func buildQuickActions(cfg config.Config) []QuickAction {
	var actions []QuickAction

	builtinHandlers := map[string]struct {
		name    string
		icon    string
		handler func(m *model) (tea.Cmd, bool)
	}{
		"repeat_last":  {"Repeat Last", "⚡", actionRepeatLast},
		"copy_command": {"Copy Command", "📋", actionCopyCommand},
		"search":       {"Search", "🔍", actionSearch},
		"edit_file":    {"Edit File", "📝", actionEditFile},
		"favorite":     {"Favorite", "⭐", actionToggleFavorite},
		"refresh":      {"Refresh", "🔄", actionRefresh},
	}

	for _, b := range cfg.QuickActions.Builtin {
		if !b.Enabled {
			continue
		}
		if handler, ok := builtinHandlers[b.ID]; ok {
			actions = append(actions, QuickAction{
				ID:       b.ID,
				Name:     handler.name,
				Icon:     handler.icon,
				Shortcut: b.Shortcut,
				Builtin:  true,
				Handler:  handler.handler,
			})
		}
	}

	for _, c := range cfg.QuickActions.Custom {
		actions = append(actions, QuickAction{
			ID:       "custom_" + c.Name,
			Name:     c.Name,
			Icon:     c.Icon,
			Shortcut: c.Shortcut,
			Builtin:  false,
			Command:  c.Action.Command,
		})
	}

	return actions
}

// Built-in action handlers

func actionRepeatLast(m *model) (tea.Cmd, bool) {
	if len(m.history) == 0 {
		return m.showNotification("⚠️", "No command history yet", "warning"), true
	}
	lastCmd := m.history[0].Command
	lastTool := m.history[0].Tool

	displayCmd := lastCmd
	if len(displayCmd) > 30 {
		displayCmd = displayCmd[:27] + "..."
	}
	notifyCmd := m.showNotification("⚡", "Repeating: "+displayCmd, "info")

	ic := &interactiveCmd{
		cmd:        lastCmd,
		needsInput: false,
		tool:       lastTool,
	}
	execCmd := tea.Exec(ic, func(err error) tea.Msg {
		return commandDoneMsg{
			command: ic.finalCmd,
			tool:    ic.tool,
			success: ic.success,
		}
	})
	return tea.Batch(notifyCmd, execCmd), true
}

func actionCopyCommand(m *model) (tea.Cmd, bool) {
	var cmdText string
	var source string
	if m.currentView == viewDetail && len(m.commands) > 0 && m.cmdCursor < len(m.commands) {
		cmdText = m.commands[m.cmdCursor].cmd
		source = "command"
	} else if len(m.history) > 0 {
		cmdText = m.history[0].Command
		source = "last command"
	}

	if cmdText == "" {
		return m.showNotification("⚠️", "Nothing to copy", "warning"), true
	}

	if err := clipboard.WriteAll(cmdText); err != nil {
		return m.showNotification("❌", "Failed to copy: "+err.Error(), "error"), true
	}

	displayCmd := cmdText
	if len(displayCmd) > 25 {
		displayCmd = displayCmd[:22] + "..."
	}
	return m.showNotification("📋", "Copied "+source+": "+displayCmd, "success"), true
}

func actionSearch(m *model) (tea.Cmd, bool) {
	return m.showNotification("🔍", "Search coming soon...", "info"), true
}

func actionEditFile(m *model) (tea.Cmd, bool) {
	res := m.currentResource()
	if res == nil {
		return m.showNotification("⚠️", "No resource selected", "warning"), true
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	// If the resource is embedded-only, copy it to user dir first
	filePath := filepath.Join(config.ResourcesDir, res.name+".md")
	if res.embedded {
		if err := os.MkdirAll(config.ResourcesDir, 0755); err != nil {
			return m.showNotification("❌", "Failed to create resources dir: "+err.Error(), "error"), true
		}
		// Only copy if not already on disk
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			data, readErr := resources.Default.ReadFile(res.name + ".md")
			if readErr != nil {
				return m.showNotification("❌", "Failed to read embedded resource: "+readErr.Error(), "error"), true
			}
			if writeErr := os.WriteFile(filePath, data, 0644); writeErr != nil {
				return m.showNotification("❌", "Failed to write resource: "+writeErr.Error(), "error"), true
			}
		}
	}

	notifyCmd := m.showNotification("📝", "Opening "+res.name+".md in "+editor, "info")

	c := exec.Command(editor, filePath)
	execCmd := tea.ExecProcess(c, func(err error) tea.Msg {
		return commandDoneMsg{}
	})
	return tea.Batch(notifyCmd, execCmd), true
}

func actionToggleFavorite(m *model) (tea.Cmd, bool) {
	if m.currentView != viewDetail || len(m.commands) == 0 || m.cmdCursor >= len(m.commands) {
		return m.showNotification("⚠️", "Select a command first", "warning"), true
	}

	cmdText := m.commands[m.cmdCursor].cmd
	displayCmd := cmdText
	if len(displayCmd) > 20 {
		displayCmd = displayCmd[:17] + "..."
	}

	if m.favorites[cmdText] {
		delete(m.favorites, cmdText)
		newFavs := []string{}
		for _, f := range m.config.Favorites {
			if f != cmdText {
				newFavs = append(newFavs, f)
			}
		}
		m.config.Favorites = newFavs
		config.Save(m.config)
		return m.showNotification("☆", "Unfavorited: "+displayCmd, "info"), true
	}

	m.favorites[cmdText] = true
	m.config.Favorites = append(m.config.Favorites, cmdText)
	config.Save(m.config)
	return m.showNotification("⭐", "Favorited: "+displayCmd, "success"), true
}

func actionRefresh(m *model) (tea.Cmd, bool) {
	m.resources = nil
	m.loadResources()
	if m.currentView == viewDetail {
		m.updateViewportContent()
	}
	return m.showNotification("🔄", fmt.Sprintf("Refreshed %d resources", len(m.resources)), "success"), true
}
