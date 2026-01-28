package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config types
type Config struct {
	Version      int                `yaml:"version"`
	QuickActions QuickActionsConfig `yaml:"quick_actions"`
	History      HistoryConfig      `yaml:"history"`
	Favorites    []string           `yaml:"favorites"`
	AI           AIConfig           `yaml:"ai,omitempty"`
	MCP          MCPConfig          `yaml:"mcp"`
}

type QuickActionsConfig struct {
	Enabled bool                  `yaml:"enabled"`
	Builtin []BuiltinActionConfig `yaml:"builtin"`
	Custom  []CustomActionConfig  `yaml:"custom"`
}

type BuiltinActionConfig struct {
	ID       string `yaml:"id"`
	Enabled  bool   `yaml:"enabled"`
	Shortcut string `yaml:"shortcut"`
}

type CustomActionConfig struct {
	Name     string       `yaml:"name"`
	Icon     string       `yaml:"icon"`
	Shortcut string       `yaml:"shortcut"`
	Action   CustomAction `yaml:"action"`
}

type CustomAction struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command"`
}

type HistoryConfig struct {
	Enabled      bool `yaml:"enabled"`
	MaxItems     int  `yaml:"max_items"`
	DisplayCount int  `yaml:"display_count"`
	Persist      bool `yaml:"persist"`
}

type AIConfig struct {
	OpenAIAPIKey string `yaml:"openai_api_key,omitempty"` // Optional, falls back to mods config if not set
}

type MCPConfig struct {
	Enabled        bool              `yaml:"enabled"`
	RefreshSeconds int               `yaml:"refresh_seconds"`
	Servers        []MCPServerConfig `yaml:"servers"`
}

type MCPServerConfig struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// HistoryEntry for tracking executed commands
type HistoryEntry struct {
	Command   string    `json:"command"`
	Tool      string    `json:"tool"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// AgentInteraction tracks interactions with AI agents
type AgentInteraction struct {
	Agent     string    `json:"agent"`      // e.g., "BIA Junior"
	Action    string    `json:"action"`     // e.g., "Code Review"
	Input     string    `json:"input"`      // Summary of input (truncated)
	Output    string    `json:"output"`     // Summary of output (truncated)
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// loadConfig loads the configuration from disk
func loadConfig() Config {
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := createDefaultConfig()
		saveConfig(cfg)
		return cfg
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return createDefaultConfig()
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return createDefaultConfig()
	}

	if cfg.Version < 2 {
		cfg.Version = 2
		if len(cfg.MCP.Servers) == 0 {
			cfg.MCP = defaultMCPConfig()
		}
	}

	if cfg.MCP.Enabled && len(cfg.MCP.Servers) == 0 {
		cfg.MCP.Servers = defaultMCPConfig().Servers
	}

	return cfg
}

// saveConfig saves the configuration to disk
func saveConfig(cfg Config) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0644)
}

// createDefaultConfig creates the default configuration
func createDefaultConfig() Config {
	return Config{
		Version: 2,
		QuickActions: QuickActionsConfig{
			Enabled: true,
			Builtin: []BuiltinActionConfig{
				{ID: "repeat_last", Enabled: true, Shortcut: "ctrl+r"},
				{ID: "copy_command", Enabled: true, Shortcut: "ctrl+y"},
				{ID: "search", Enabled: true, Shortcut: "/"},
				{ID: "edit_file", Enabled: true, Shortcut: "ctrl+e"},
				{ID: "favorite", Enabled: true, Shortcut: "ctrl+f"},
				{ID: "refresh", Enabled: true, Shortcut: "ctrl+l"},
			},
			Custom: []CustomActionConfig{},
		},
		History: HistoryConfig{
			Enabled:      true,
			MaxItems:     50,
			DisplayCount: 5,
			Persist:      true,
		},
		Favorites: []string{},
		AI: AIConfig{
			OpenAIAPIKey: "", // Optional: Set your OpenAI API key here, or leave empty to use mods config
		},
		MCP: defaultMCPConfig(),
	}
}

func defaultMCPConfig() MCPConfig {
	return MCPConfig{
		Enabled:        true,
		RefreshSeconds: 60,
		Servers: []MCPServerConfig{
			{
				Name: "bldrspec-ai",
				URL:  GetDefaultMCPServerURL(),
			},
		},
	}
}

// loadHistory loads command history from disk
func loadHistory() []HistoryEntry {
	historyPath := filepath.Join(dataDir, "history.json")

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return []HistoryEntry{}
	}

	var history []HistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return []HistoryEntry{}
	}

	return history
}

// saveHistory saves command history to disk
func saveHistory(history []HistoryEntry) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dataDir, "history.json"), data, 0644)
}

// addToHistory adds an entry to history and maintains max size
func addToHistory(history []HistoryEntry, entry HistoryEntry, maxItems int) []HistoryEntry {
	// Prepend new entry
	history = append([]HistoryEntry{entry}, history...)

	// Trim to max
	if len(history) > maxItems {
		history = history[:maxItems]
	}

	return history
}

// loadAgentHistory loads agent interaction history from disk
func loadAgentHistory() []AgentInteraction {
	historyPath := filepath.Join(dataDir, "agent_history.json")

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return []AgentInteraction{}
	}

	var history []AgentInteraction
	if err := json.Unmarshal(data, &history); err != nil {
		return []AgentInteraction{}
	}

	return history
}

// saveAgentHistory saves agent interaction history to disk
func saveAgentHistory(history []AgentInteraction) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dataDir, "agent_history.json"), data, 0644)
}

// addAgentInteraction adds an interaction to history and maintains max size
func addAgentInteraction(history []AgentInteraction, entry AgentInteraction, maxItems int) []AgentInteraction {
	// Prepend new entry
	history = append([]AgentInteraction{entry}, history...)

	// Trim to max
	if len(history) > maxItems {
		history = history[:maxItems]
	}

	return history
}
