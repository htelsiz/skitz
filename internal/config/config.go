package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Directories used by config and data persistence.
var (
	ConfigDir    string
	DataDir      string
	ResourcesDir string
)

func init() {
	home, _ := os.UserHomeDir()
	ConfigDir = filepath.Join(home, ".config", "skitz")
	DataDir = filepath.Join(home, ".local", "share", "skitz")
	ResourcesDir = filepath.Join(home, ".config", "skitz", "resources")
}

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
	OpenAIAPIKey    string           `yaml:"openai_api_key,omitempty"` // deprecated, use Providers
	DefaultProvider string           `yaml:"default_provider,omitempty"`
	Providers       []ProviderConfig `yaml:"providers,omitempty"`
}

type ProviderConfig struct {
	Name         string `yaml:"name"`
	ProviderType string `yaml:"provider_type,omitempty"` // "openai", "anthropic", "ollama", "openai-compatible"
	APIKey       string `yaml:"api_key,omitempty"`
	BaseURL      string `yaml:"base_url,omitempty"` // for custom endpoints
	DefaultModel string `yaml:"default_model,omitempty"`
	Enabled      bool   `yaml:"enabled"`
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
	ID        string    `json:"id"`          // UUID for tracking
	Agent     string    `json:"agent"`
	Action    string    `json:"action"`
	Input     string    `json:"input"`
	Output    string    `json:"output"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Runtime   string    `json:"runtime"`      // "docker", "e2b"
	Provider  string    `json:"provider"`     // provider name
	Duration  int64     `json:"duration_ms"`  // execution time in milliseconds
}

// Load loads the configuration from disk. defaultMCPURL is used when
// creating the default MCP server entry.
func Load(defaultMCPURL string) Config {
	configPath := filepath.Join(ConfigDir, "config.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := CreateDefault(defaultMCPURL)
		Save(cfg)
		return cfg
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return CreateDefault(defaultMCPURL)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return CreateDefault(defaultMCPURL)
	}

	if cfg.Version < 2 {
		cfg.Version = 2
		if len(cfg.MCP.Servers) == 0 {
			cfg.MCP = defaultMCPConfig(defaultMCPURL)
		}
	}

	if cfg.MCP.Enabled && len(cfg.MCP.Servers) == 0 {
		cfg.MCP.Servers = defaultMCPConfig(defaultMCPURL).Servers
	}

	return cfg
}

// Save saves the configuration to disk.
func Save(cfg Config) error {
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(ConfigDir, "config.yaml"), data, 0644)
}

// CreateDefault creates the default configuration.
func CreateDefault(defaultMCPURL string) Config {
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
			OpenAIAPIKey: "",
		},
		MCP: defaultMCPConfig(defaultMCPURL),
	}
}

func defaultMCPConfig(defaultMCPURL string) MCPConfig {
	return MCPConfig{
		Enabled:        true,
		RefreshSeconds: 60,
		Servers: []MCPServerConfig{
			{
				Name: "bldrspec-ai",
				URL:  defaultMCPURL,
			},
		},
	}
}

// LoadHistory loads command history from disk.
func LoadHistory() []HistoryEntry {
	historyPath := filepath.Join(DataDir, "history.json")

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

// SaveHistory saves command history to disk.
func SaveHistory(history []HistoryEntry) error {
	if err := os.MkdirAll(DataDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(DataDir, "history.json"), data, 0644)
}

// AddToHistory adds an entry to history and maintains max size.
func AddToHistory(history []HistoryEntry, entry HistoryEntry, maxItems int) []HistoryEntry {
	history = append([]HistoryEntry{entry}, history...)

	if len(history) > maxItems {
		history = history[:maxItems]
	}

	return history
}

// LoadAgentHistory loads agent interaction history from disk.
func LoadAgentHistory() []AgentInteraction {
	historyPath := filepath.Join(DataDir, "agent_history.json")

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

// SaveAgentHistory saves agent interaction history to disk.
func SaveAgentHistory(history []AgentInteraction) error {
	if err := os.MkdirAll(DataDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(DataDir, "agent_history.json"), data, 0644)
}

// AddAgentInteraction adds an interaction to history and maintains max size.
func AddAgentInteraction(history []AgentInteraction, entry AgentInteraction, maxItems int) []AgentInteraction {
	history = append([]AgentInteraction{entry}, history...)

	if len(history) > maxItems {
		history = history[:maxItems]
	}

	return history
}
