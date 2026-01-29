package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/htelsiz/skitz/internal/config"
	mcppkg "github.com/htelsiz/skitz/internal/mcp"
)

func fetchMCPStatusCmd(cfg config.MCPConfig) tea.Cmd {
	return func() tea.Msg {
		if !cfg.Enabled || len(cfg.Servers) == 0 {
			return mcpStatusMsg{Statuses: nil}
		}

		statuses := make([]mcppkg.ServerStatus, 0, len(cfg.Servers))
		for _, server := range cfg.Servers {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			status := mcppkg.FetchServerStatus(ctx, server.Name, server.URL)
			cancel()
			statuses = append(statuses, status)
		}

		return mcpStatusMsg{Statuses: statuses}
	}
}

func scheduleMCPRefreshCmd(seconds int) tea.Cmd {
	if seconds <= 0 {
		return nil
	}

	return tea.Tick(time.Duration(seconds)*time.Second, func(time.Time) tea.Msg {
		return mcpRefreshTickMsg{}
	})
}
