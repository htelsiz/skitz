package app

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aaronjanse/3mux/ecma48"
	"github.com/aaronjanse/3mux/vterm"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/htelsiz/skitz/internal/config"
	mcppkg "github.com/htelsiz/skitz/internal/mcp"
	"github.com/htelsiz/skitz/internal/resources"
)

type model struct {
	resources     []resource
	resCursor     int
	secCursor     int
	width, height int

	// View state
	currentView int // viewDashboard or viewDetail

	// Dashboard tabs
	dashboardTab         int                // 0=Resources, 1=Actions
	actionItems          []DashboardAction  // Available actions
	actionCursor         int                // Selected action
	addResourceWizard    *AddResourceWizard  // Add Resource wizard state
	preferencesWizard    *PreferencesWizard  // Preferences wizard state
	providersWizard      *ProvidersWizard    // Configure Providers wizard state
	pendingResourceReload bool               // Reload resources after editor closes
	pendingConfigReload   bool               // Reload config after editor closes

	// View components (bubbles)
	contentView viewport.Model
	viewReady   bool

	// Command execution state
	commands  []command // Parsed commands from current section
	cmdCursor int       // Currently selected command (0-based)

	// Cached rendered markdown for non-command content (avoids re-rendering on cursor change)
	cachedMarkdownContext string

	// Animation state
	quotePos    float64          // Current character position (animated)
	quoteVel    float64          // Velocity for spring
	quoteTarget float64          // Target position (full quote length)
	spring      harmonica.Spring // Spring for smooth animation

	// Config
	config       config.Config
	history      []config.HistoryEntry
	agentHistory []config.AgentInteraction
	favorites    map[string]bool

	// Notification/Toast
	notification *Notification

	// Command Palette (cmd+k)
	palette Palette

	// MCP status
	mcpStatus []mcppkg.ServerStatus

	// Embedded terminal
	term EmbeddedTerm
}

// EmbeddedTerm holds the state for the embedded terminal pane
type EmbeddedTerm struct {
	active  bool
	focused bool
	vt      *vterm.VTerm
	pty     *os.File
	width   int
	height  int
	exitErr error
	exited  bool
	command string // The command that was executed
	// Static output mode (for MCP tools, etc.)
	staticOutput string
	staticTitle  string
}

type tickMsg time.Time

type mcpStatusMsg struct {
	Statuses []mcppkg.ServerStatus
}

type mcpRefreshTickMsg struct{}

// Terminal messages
type termOutputMsg struct{}
type termExitMsg struct{ err error }

// staticOutputMsg displays static text in the terminal pane
type staticOutputMsg struct {
	title  string
	output string
}

// agentInteractionMsg is sent when an agent interaction completes
type agentInteractionMsg struct {
	interaction config.AgentInteraction
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/60, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func newModel(startResource string) model {
	cfg := config.Load(mcppkg.GetDefaultMCPServerURL())
	history := config.LoadHistory()
	agentHistory := config.LoadAgentHistory()

	favorites := make(map[string]bool)
	for _, f := range cfg.Favorites {
		favorites[f] = true
	}

	m := model{
		spring:       harmonica.NewSpring(harmonica.FPS(60), 6.0, 0.7),
		config:       cfg,
		history:      history,
		agentHistory: agentHistory,
		favorites:    favorites,
	}
	m.loadResources()
	m.actionItems = m.buildDashboardActions()

	if startResource != "" {
		for i, r := range m.resources {
			if r.name == startResource {
				m.resCursor = i
				m.currentView = viewDetail
				break
			}
		}
	}

	return m
}

// buildDashboardActions creates the list of available dashboard actions
func (m *model) buildDashboardActions() []DashboardAction {
	return []DashboardAction{
		{
			ID:          "add_resource",
			Name:        "Add Resource",
			Icon:        "+",
			Description: "Create a new resource file",
			Handler: func(m *model) tea.Cmd {
				return m.startAddResourceWizard()
			},
		},
		{
			ID:          "providers",
			Name:        "Configure Providers",
			Icon:        "◈",
			Description: "Set up LLM providers",
			Handler: func(m *model) tea.Cmd {
				return m.startProvidersWizard()
			},
		},
		{
			ID:          "preferences",
			Name:        "Preferences",
			Icon:        "⚙",
			Description: "Edit skitz configuration",
			Handler: func(m *model) tea.Cmd {
				return m.editPreferences()
			},
		},
	}
}

// startAddResourceWizard begins the Add Resource wizard flow
func (m *model) startAddResourceWizard() tea.Cmd {
	m.addResourceWizard = &AddResourceWizard{
		Step:     0,
		Name:     "",
		Template: "blank",
	}
	return m.buildAddResourceForm()
}

// buildAddResourceForm creates the huh form for the current wizard step
func (m *model) buildAddResourceForm() tea.Cmd {
	wizard := m.addResourceWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0: // Name input
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Resource Name").
					Description("Enter a name for your new resource").
					Placeholder("my-resource").
					Value(&wizard.Name),
			),
		).
			WithWidth(80).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 1: // Template selection
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Template").
					Description("Choose a starting template").
					Options(
						huh.NewOption("Blank - Empty resource file", "blank"),
						huh.NewOption("Commands - Basic command structure", "commands"),
						huh.NewOption("Detailed - Full sections layout", "detailed"),
					).
					Value(&wizard.Template),
			),
		).
			WithWidth(80).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 2: // Confirmation
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Create Resource?").
					Description(fmt.Sprintf("Create '%s' with '%s' template?", wizard.Name, wizard.Template)).
					Affirmative("Create").
					Negative("Cancel"),
			),
		).
			WithWidth(80).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()
	}

	return nil
}

// nextAddResourceStep advances the wizard to the next step
func (m *model) nextAddResourceStep() tea.Cmd {
	wizard := m.addResourceWizard
	if wizard == nil {
		return nil
	}

	wizard.Step++
	if wizard.Step > 2 {
		// Wizard complete, create the file
		return m.createResourceFile()
	}

	return m.buildAddResourceForm()
}

// editPreferences starts the preferences wizard
func (m *model) editPreferences() tea.Cmd {
	m.preferencesWizard = &PreferencesWizard{
		Step:                0,
		HistoryEnabled:      m.config.History.Enabled,
		HistoryMaxItems:     fmt.Sprintf("%d", m.config.History.MaxItems),
		HistoryDisplayCount: fmt.Sprintf("%d", m.config.History.DisplayCount),
		MCPEnabled:          m.config.MCP.Enabled,
		Editor:              os.Getenv("EDITOR"),
	}
	return m.buildPreferencesForm()
}

// buildPreferencesForm creates the form for the current preferences step
func (m *model) buildPreferencesForm() tea.Cmd {
	wizard := m.preferencesWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0: // Main menu
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Preferences").
					Description("What would you like to configure?").
					Options(
						huh.NewOption("History Settings", "history"),
						huh.NewOption("MCP Servers", "mcp"),
						huh.NewOption("Edit Config File", "editor"),
					).
					Value(&wizard.Section),
			),
		).
			WithWidth(80).
			WithShowHelp(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 1: // Section-specific forms
		switch wizard.Section {
		case "history":
			wizard.InputForm = huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Enable History").
						Description("Track command execution history").
						Value(&wizard.HistoryEnabled),
					huh.NewInput().
						Title("Max Items").
						Description("Maximum history entries to keep").
						Value(&wizard.HistoryMaxItems),
					huh.NewInput().
						Title("Display Count").
						Description("Number of items shown in sidebar").
						Value(&wizard.HistoryDisplayCount),
				),
			).
				WithWidth(80).
				WithShowHelp(true).
				WithTheme(huh.ThemeCatppuccin())
			return wizard.InputForm.Init()

		case "mcp":
			// Build options from current servers
			var serverOptions []huh.Option[string]
			serverOptions = append(serverOptions, huh.NewOption("Add New Server", "add"))
			for _, srv := range m.config.MCP.Servers {
				serverOptions = append(serverOptions, huh.NewOption("Edit: "+srv.Name, "edit:"+srv.Name))
				serverOptions = append(serverOptions, huh.NewOption("Remove: "+srv.Name, "remove:"+srv.Name))
			}
			serverOptions = append(serverOptions, huh.NewOption("Toggle MCP (currently "+boolToOnOff(wizard.MCPEnabled)+")", "toggle"))

			wizard.InputForm = huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("MCP Servers").
						Description("Manage Model Context Protocol servers").
						Options(serverOptions...).
						Value(&wizard.MCPAction),
				),
			).
				WithWidth(80).
				WithShowHelp(true).
				WithTheme(huh.ThemeCatppuccin())
			return wizard.InputForm.Init()

		case "editor":
			// Open config file in editor
			m.preferencesWizard = nil
			return m.openConfigInEditor()
		}

	case 2: // MCP add/edit form
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Server Name").
					Description("A friendly name for this server").
					Placeholder("my-server").
					Value(&wizard.MCPName),
				huh.NewInput().
					Title("Server URL").
					Description("The MCP server endpoint").
					Placeholder("http://localhost:8001/mcp/").
					Value(&wizard.MCPURL),
			),
		).
			WithWidth(80).
			WithShowHelp(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()
	}

	return nil
}

// nextPreferencesStep advances the preferences wizard
func (m *model) nextPreferencesStep() tea.Cmd {
	wizard := m.preferencesWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0: // After menu selection
		wizard.Step = 1
		return m.buildPreferencesForm()

	case 1: // After section form
		switch wizard.Section {
		case "history":
			// Save history settings
			m.config.History.Enabled = wizard.HistoryEnabled
			if maxItems, err := strconv.Atoi(wizard.HistoryMaxItems); err == nil && maxItems > 0 {
				m.config.History.MaxItems = maxItems
			}
			if displayCount, err := strconv.Atoi(wizard.HistoryDisplayCount); err == nil && displayCount > 0 {
				m.config.History.DisplayCount = displayCount
			}
			config.Save(m.config)
			m.preferencesWizard = nil
			return m.showNotification("✓", "History settings saved", "success")

		case "mcp":
			if wizard.MCPAction == "toggle" {
				wizard.MCPEnabled = !wizard.MCPEnabled
				m.config.MCP.Enabled = wizard.MCPEnabled
				config.Save(m.config)
				m.preferencesWizard = nil
				status := "disabled"
				if wizard.MCPEnabled {
					status = "enabled"
				}
				return m.showNotification("✓", "MCP "+status, "success")
			} else if wizard.MCPAction == "add" {
				wizard.MCPName = ""
				wizard.MCPURL = ""
				wizard.Step = 2
				return m.buildPreferencesForm()
			} else if strings.HasPrefix(wizard.MCPAction, "edit:") {
				serverName := strings.TrimPrefix(wizard.MCPAction, "edit:")
				for _, srv := range m.config.MCP.Servers {
					if srv.Name == serverName {
						wizard.MCPName = srv.Name
						wizard.MCPURL = srv.URL
						break
					}
				}
				wizard.Step = 2
				return m.buildPreferencesForm()
			} else if strings.HasPrefix(wizard.MCPAction, "remove:") {
				serverName := strings.TrimPrefix(wizard.MCPAction, "remove:")
				var newServers []config.MCPServerConfig
				for _, srv := range m.config.MCP.Servers {
					if srv.Name != serverName {
						newServers = append(newServers, srv)
					}
				}
				m.config.MCP.Servers = newServers
				config.Save(m.config)
				m.preferencesWizard = nil
				return m.showNotification("✓", "Removed "+serverName, "success")
			}
		}

	case 2: // After MCP add/edit form
		if wizard.MCPName == "" || wizard.MCPURL == "" {
			m.preferencesWizard = nil
			return m.showNotification("!", "Name and URL are required", "error")
		}

		if strings.HasPrefix(wizard.MCPAction, "edit:") {
			// Update existing server
			oldName := strings.TrimPrefix(wizard.MCPAction, "edit:")
			for i, srv := range m.config.MCP.Servers {
				if srv.Name == oldName {
					m.config.MCP.Servers[i].Name = wizard.MCPName
					m.config.MCP.Servers[i].URL = wizard.MCPURL
					break
				}
			}
		} else {
			// Add new server
			m.config.MCP.Servers = append(m.config.MCP.Servers, config.MCPServerConfig{
				Name: wizard.MCPName,
				URL:  wizard.MCPURL,
			})
		}
		config.Save(m.config)
		m.preferencesWizard = nil
		return m.showNotification("✓", "MCP server saved", "success")
	}

	m.preferencesWizard = nil
	return nil
}

// openConfigInEditor opens the config file in the user's editor
func (m *model) openConfigInEditor() tea.Cmd {
	if err := os.MkdirAll(config.ConfigDir, 0755); err != nil {
		return m.showNotification("!", "Failed to create config directory: "+err.Error(), "error")
	}

	configPath := filepath.Join(config.ConfigDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config.Save(m.config)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		for _, e := range []string{"vim", "vi", "nano"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return m.showNotification("!", "No editor found. Set $EDITOR", "error")
	}

	m.pendingConfigReload = true
	return m.runCommand(CommandSpec{
		Command: fmt.Sprintf("%s %q", editor, configPath),
		Mode:    CommandInteractive,
	})
}

// startProvidersWizard begins the Configure Providers wizard
func (m *model) startProvidersWizard() tea.Cmd {
	m.providersWizard = &ProvidersWizard{
		Step:    0,
		Enabled: true,
	}
	return m.buildProvidersForm()
}

// buildProvidersForm creates the form for the current providers step
func (m *model) buildProvidersForm() tea.Cmd {
	wizard := m.providersWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0: // Main menu
		var options []huh.Option[string]
		options = append(options, huh.NewOption("Add Provider", "add"))

		// List existing providers
		for _, p := range m.config.AI.Providers {
			status := "disabled"
			if p.Enabled {
				status = "enabled"
			}
			if p.Name == m.config.AI.DefaultProvider {
				status = "default"
			}
			options = append(options, huh.NewOption(fmt.Sprintf("Edit: %s (%s)", p.Name, status), "edit:"+p.Name))
			options = append(options, huh.NewOption(fmt.Sprintf("Remove: %s", p.Name), "remove:"+p.Name))
		}

		if len(m.config.AI.Providers) > 0 {
			options = append(options, huh.NewOption("Set Default Provider", "default"))
		}

		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Configure Providers").
					Description("Manage your LLM provider connections").
					Options(options...).
					Value(&wizard.Action),
			),
		).
			WithWidth(80).
			WithShowHelp(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 1: // Provider type selection (for new provider)
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Provider Type").
					Description("Select the LLM provider").
					Options(
						huh.NewOption("OpenAI", "openai"),
						huh.NewOption("Anthropic", "anthropic"),
						huh.NewOption("Ollama (Local)", "ollama"),
						huh.NewOption("OpenAI Compatible", "openai-compatible"),
					).
					Value(&wizard.ProviderType),
			),
		).
			WithWidth(80).
			WithShowHelp(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 2: // Provider details form
		var fields []huh.Field

		fields = append(fields,
			huh.NewInput().
				Title("Name").
				Description("A friendly name for this provider").
				Placeholder(wizard.ProviderType).
				Value(&wizard.Name),
		)

		// API key for cloud providers
		if wizard.ProviderType != "ollama" {
			fields = append(fields,
				huh.NewInput().
					Title("API Key").
					Description("Your API key (stored locally)").
					EchoMode(huh.EchoModePassword).
					Value(&wizard.APIKey),
			)
		}

		// Base URL for custom/ollama
		if wizard.ProviderType == "ollama" || wizard.ProviderType == "openai-compatible" {
			placeholder := "http://localhost:11434"
			if wizard.ProviderType == "openai-compatible" {
				placeholder = "https://api.example.com/v1"
			}
			fields = append(fields,
				huh.NewInput().
					Title("Base URL").
					Description("API endpoint URL").
					Placeholder(placeholder).
					Value(&wizard.BaseURL),
			)
		}

		// Default model
		modelPlaceholder := "gpt-4"
		switch wizard.ProviderType {
		case "anthropic":
			modelPlaceholder = "claude-sonnet-4-20250514"
		case "ollama":
			modelPlaceholder = "llama3"
		}
		fields = append(fields,
			huh.NewInput().
				Title("Default Model").
				Description("Model to use by default").
				Placeholder(modelPlaceholder).
				Value(&wizard.DefaultModel),
		)

		fields = append(fields,
			huh.NewConfirm().
				Title("Enabled").
				Description("Enable this provider").
				Value(&wizard.Enabled),
		)

		wizard.InputForm = huh.NewForm(huh.NewGroup(fields...)).
			WithWidth(80).
			WithShowHelp(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 3: // Set default provider
		var options []huh.Option[string]
		for _, p := range m.config.AI.Providers {
			if p.Enabled {
				label := p.Name
				if p.Name == m.config.AI.DefaultProvider {
					label += " (current)"
				}
				options = append(options, huh.NewOption(label, p.Name))
			}
		}

		if len(options) == 0 {
			m.providersWizard = nil
			return m.showNotification("!", "No enabled providers", "error")
		}

		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Default Provider").
					Description("Select the default LLM provider").
					Options(options...).
					Value(&wizard.Name),
			),
		).
			WithWidth(80).
			WithShowHelp(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()
	}

	return nil
}

// nextProvidersStep advances the providers wizard
func (m *model) nextProvidersStep() tea.Cmd {
	wizard := m.providersWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0: // After menu selection
		if wizard.Action == "add" {
			wizard.Step = 1
			return m.buildProvidersForm()
		} else if wizard.Action == "default" {
			wizard.Step = 3
			return m.buildProvidersForm()
		} else if strings.HasPrefix(wizard.Action, "edit:") {
			// Load existing provider data
			providerName := strings.TrimPrefix(wizard.Action, "edit:")
			for _, p := range m.config.AI.Providers {
				if p.Name == providerName {
					wizard.Name = p.Name
					wizard.APIKey = p.APIKey
					wizard.BaseURL = p.BaseURL
					wizard.DefaultModel = p.DefaultModel
					wizard.Enabled = p.Enabled
					// Determine provider type from existing data
					wizard.ProviderType = "openai-compatible"
					if strings.Contains(strings.ToLower(p.Name), "openai") || p.BaseURL == "" {
						wizard.ProviderType = "openai"
					} else if strings.Contains(strings.ToLower(p.Name), "anthropic") {
						wizard.ProviderType = "anthropic"
					} else if strings.Contains(strings.ToLower(p.Name), "ollama") || strings.Contains(p.BaseURL, "11434") {
						wizard.ProviderType = "ollama"
					}
					break
				}
			}
			wizard.Step = 2
			return m.buildProvidersForm()
		} else if strings.HasPrefix(wizard.Action, "remove:") {
			providerName := strings.TrimPrefix(wizard.Action, "remove:")
			var newProviders []config.ProviderConfig
			for _, p := range m.config.AI.Providers {
				if p.Name != providerName {
					newProviders = append(newProviders, p)
				}
			}
			m.config.AI.Providers = newProviders
			if m.config.AI.DefaultProvider == providerName {
				m.config.AI.DefaultProvider = ""
			}
			config.Save(m.config)
			m.providersWizard = nil
			return m.showNotification("✓", "Removed "+providerName, "success")
		}

	case 1: // After provider type selection
		// Set defaults based on provider type
		if wizard.Name == "" {
			wizard.Name = wizard.ProviderType
		}
		switch wizard.ProviderType {
		case "ollama":
			if wizard.BaseURL == "" {
				wizard.BaseURL = "http://localhost:11434"
			}
		case "anthropic":
			if wizard.DefaultModel == "" {
				wizard.DefaultModel = "claude-sonnet-4-20250514"
			}
		case "openai":
			if wizard.DefaultModel == "" {
				wizard.DefaultModel = "gpt-4"
			}
		}
		wizard.Step = 2
		return m.buildProvidersForm()

	case 2: // After provider details form
		if wizard.Name == "" {
			wizard.Name = wizard.ProviderType
		}

		newProvider := config.ProviderConfig{
			Name:         wizard.Name,
			APIKey:       wizard.APIKey,
			BaseURL:      wizard.BaseURL,
			DefaultModel: wizard.DefaultModel,
			Enabled:      wizard.Enabled,
		}

		// Check if editing existing or adding new
		found := false
		if strings.HasPrefix(wizard.Action, "edit:") {
			oldName := strings.TrimPrefix(wizard.Action, "edit:")
			for i, p := range m.config.AI.Providers {
				if p.Name == oldName {
					m.config.AI.Providers[i] = newProvider
					found = true
					break
				}
			}
		}

		if !found {
			m.config.AI.Providers = append(m.config.AI.Providers, newProvider)
		}

		// Set as default if it's the first provider
		if len(m.config.AI.Providers) == 1 && newProvider.Enabled {
			m.config.AI.DefaultProvider = newProvider.Name
		}

		config.Save(m.config)
		m.providersWizard = nil
		return m.showNotification("✓", "Provider saved: "+wizard.Name, "success")

	case 3: // After default provider selection
		m.config.AI.DefaultProvider = wizard.Name
		config.Save(m.config)
		m.providersWizard = nil
		return m.showNotification("✓", "Default provider: "+wizard.Name, "success")
	}

	m.providersWizard = nil
	return nil
}

// Helper function for preferences wizard
func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// editResource opens the selected resource in the user's external editor
func (m *model) editResource() tea.Cmd {
	res := m.currentResource()
	if res == nil {
		return m.showNotification("!", "No resource selected", "error")
	}

	// Ensure user resources directory exists
	if err := os.MkdirAll(config.ResourcesDir, 0755); err != nil {
		return m.showNotification("!", "Failed to create directory: "+err.Error(), "error")
	}

	filePath := filepath.Join(config.ResourcesDir, res.name+".md")

	// If resource is embedded and doesn't exist in user dir, copy it first
	if res.embedded {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := os.WriteFile(filePath, []byte(res.content), 0644); err != nil {
				return m.showNotification("!", "Failed to copy resource: "+err.Error(), "error")
			}
		}
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Fallback to common editors
		for _, e := range []string{"vim", "vi", "nano"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return m.showNotification("!", "No editor found. Set $EDITOR", "error")
	}

	// Set flag to reload resources when editor closes
	m.pendingResourceReload = true

	// Run editor in interactive mode
	return m.runCommand(CommandSpec{
		Command: fmt.Sprintf("%s %q", editor, filePath),
		Mode:    CommandInteractive,
	})
}

// createResourceFile writes the new resource file to disk
func (m *model) createResourceFile() tea.Cmd {
	wizard := m.addResourceWizard
	if wizard == nil || wizard.Name == "" {
		m.addResourceWizard = nil
		return m.showNotification("!", "Resource name cannot be empty", "error")
	}

	// Sanitize name
	name := strings.TrimSpace(wizard.Name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")

	// Build content based on template
	var content string
	switch wizard.Template {
	case "commands":
		content = fmt.Sprintf("# %s\n\n## Commands\n\n`example-command` Example description ^run\n", name)
	case "detailed":
		content = fmt.Sprintf("# %s\n\n## Overview\n\nAdd overview here.\n\n## Commands\n\n`example-command` Example description ^run\n\n## Configuration\n\nAdd configuration notes here.\n", name)
	default: // blank
		content = fmt.Sprintf("# %s\n\n", name)
	}

	// Ensure directory exists
	if err := os.MkdirAll(config.ResourcesDir, 0755); err != nil {
		m.addResourceWizard = nil
		return m.showNotification("!", "Failed to create directory: "+err.Error(), "error")
	}

	// Write file
	filePath := filepath.Join(config.ResourcesDir, name+".md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		m.addResourceWizard = nil
		return m.showNotification("!", "Failed to create file: "+err.Error(), "error")
	}

	// Reload resources and clear wizard
	m.loadResources()
	m.addResourceWizard = nil
	m.dashboardTab = 0 // Switch back to Resources tab

	return m.showNotification("", fmt.Sprintf("Created resource: %s", name), "success")
}

func (m *model) loadResources() {
	// Clear existing resources before reloading
	m.resources = nil
	seen := make(map[string]bool)

	descriptions := map[string]string{
		"claude":     "AI coding assistant CLI",
		"docker":     "Container management",
		"git":        "Version control & GitHub CLI",
		"mcp":        "Model Context Protocol",
		"azure":      "Cloud resource management",
		"cursor":     "AI-powered code editor",
		"fast-agent": "MCP-native AI agent framework",
	}

	// 1. Read user resources from ~/.config/skitz/resources/ (override embedded)
	userDir := config.ResourcesDir
	if files, err := os.ReadDir(userDir); err == nil {
		for _, f := range files {
			name := f.Name()
			if strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, "-detail.md") {
				resName := strings.TrimSuffix(name, ".md")
				content, _ := os.ReadFile(filepath.Join(userDir, name))

				res := resource{
					name:        resName,
					description: descriptions[resName],
					content:     string(content),
					embedded:    false,
				}
				res.sections = append(res.sections, section{
					title:   "Commands",
					content: string(content),
				})

				// Load detail sections
				detailPath := filepath.Join(userDir, resName+"-detail.md")
				if file, err := os.Open(detailPath); err == nil {
					var cur *section
					var buf strings.Builder
					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						line := scanner.Text()
						if strings.HasPrefix(line, "## ") {
							if cur != nil {
								cur.content = buf.String()
								res.sections = append(res.sections, *cur)
							}
							cur = &section{title: strings.TrimPrefix(line, "## ")}
							buf.Reset()
							buf.WriteString(line + "\n")
						} else if cur != nil {
							buf.WriteString(line + "\n")
						}
					}
					if cur != nil {
						cur.content = buf.String()
						res.sections = append(res.sections, *cur)
					}
					file.Close()
				}

				m.resources = append(m.resources, res)
				seen[resName] = true
			}
		}
	}

	// 2. Read embedded defaults for any not already loaded from user dir
	entries, err := resources.Default.ReadDir(".")
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, "-detail.md") {
				resName := strings.TrimSuffix(name, ".md")
				if seen[resName] {
					continue // User override takes precedence
				}

				content, readErr := resources.Default.ReadFile(name)
				if readErr != nil {
					continue
				}

				res := resource{
					name:        resName,
					description: descriptions[resName],
					content:     string(content),
					embedded:    true,
				}
				res.sections = append(res.sections, section{
					title:   "Commands",
					content: string(content),
				})

				// Load embedded detail sections
				detailName := resName + "-detail.md"
				if detailContent, err := resources.Default.ReadFile(detailName); err == nil {
					var cur *section
					var buf strings.Builder
					for _, line := range strings.Split(string(detailContent), "\n") {
						if strings.HasPrefix(line, "## ") {
							if cur != nil {
								cur.content = buf.String()
								res.sections = append(res.sections, *cur)
							}
							cur = &section{title: strings.TrimPrefix(line, "## ")}
							buf.Reset()
							buf.WriteString(line + "\n")
						} else if cur != nil {
							buf.WriteString(line + "\n")
						}
					}
					if cur != nil {
						cur.content = buf.String()
						res.sections = append(res.sections, *cur)
					}
				}

				m.resources = append(m.resources, res)
				seen[resName] = true
			}
		}
	}
}

func (m model) currentResource() *resource {
	if m.resCursor < len(m.resources) {
		return &m.resources[m.resCursor]
	}
	return nil
}

func (m model) currentSection() *section {
	if res := m.currentResource(); res != nil {
		if m.secCursor < len(res.sections) {
			return &res.sections[m.secCursor]
		}
	}
	return nil
}

func (m *model) initViewComponents() {
	res := m.currentResource()
	if res == nil {
		return
	}

	contentW := m.width - 4
	contentH := m.height - 8

	if contentW < 60 {
		contentW = 60
	}
	if contentH < 10 {
		contentH = 10
	}

	m.contentView = viewport.New(contentW, contentH)
	m.contentView.Style = lipgloss.NewStyle()

	m.updateViewportContent()
	m.viewReady = true
}

func (m *model) updateViewportContent() {
	sec := m.currentSection()
	if sec == nil {
		m.contentView.SetContent("No content")
		m.cachedMarkdownContext = ""
		return
	}

	res := m.currentResource()
	meta := toolMetadata[res.name]

	m.commands = parseCommands(sec.content)
	if m.cmdCursor >= len(m.commands) {
		m.cmdCursor = 0
	}

	// Build the interactive command list as viewport content
	commandList := m.renderCommandList(m.contentView.Width, meta.color)

	// Check for non-command content to render as markdown context below
	m.cachedMarkdownContext = ""
	lines := strings.Split(sec.content, "\n")
	cmdRunRe := regexp.MustCompile("`[^`]+`\\s*[^^]*\\s*\\^run")
	var contextLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || cmdRunRe.MatchString(line) {
			continue
		}
		contextLines = append(contextLines, line)
	}
	if len(contextLines) > 0 {
		m.cachedMarkdownContext = strings.Join(contextLines, "\n")
	}

	if m.cachedMarkdownContext != "" {
		m.contentView.SetContent(commandList + "\n\n" + m.cachedMarkdownContext)
	} else {
		m.contentView.SetContent(commandList)
	}
	m.contentView.GotoTop()
}

// refreshCommandListDisplay re-renders the viewport when the cursor changes
// without re-parsing commands or re-processing markdown.
func (m *model) refreshCommandListDisplay() {
	res := m.currentResource()
	if res == nil || len(m.commands) == 0 {
		return
	}
	meta := toolMetadata[res.name]
	commandList := m.renderCommandList(m.contentView.Width, meta.color)

	if m.cachedMarkdownContext != "" {
		m.contentView.SetContent(commandList + "\n\n" + m.cachedMarkdownContext)
	} else {
		m.contentView.SetContent(commandList)
	}

	// Keep selected command visible in the viewport
	// Each command row is ~1 line, header takes ~4 lines
	headerLines := 4
	selectedLine := headerLines + m.cmdCursor
	viewTop := m.contentView.YOffset
	viewBottom := viewTop + m.contentView.Height

	if selectedLine < viewTop {
		m.contentView.SetYOffset(selectedLine)
	} else if selectedLine >= viewBottom {
		m.contentView.SetYOffset(selectedLine - m.contentView.Height + 1)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		fetchMCPStatusCmd(m.config.MCP),
		scheduleMCPRefreshCmd(m.config.MCP.RefreshSeconds),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Forward non-key messages to palette form
	if m.palette.State == PaletteStateCollectingParams && m.palette.InputForm != nil {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			form, cmd := m.palette.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.palette.InputForm = f
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Forward non-key messages to add resource wizard form
	if m.addResourceWizard != nil && m.addResourceWizard.InputForm != nil {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			form, cmd := m.addResourceWizard.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.addResourceWizard.InputForm = f
				// Check for form completion after non-key message processing
				if f.State == huh.StateCompleted {
					return m, m.nextAddResourceStep()
				}
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Forward non-key messages to preferences wizard form
	if m.preferencesWizard != nil && m.preferencesWizard.InputForm != nil {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			form, cmd := m.preferencesWizard.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.preferencesWizard.InputForm = f
				if f.State == huh.StateCompleted {
					return m, m.nextPreferencesStep()
				}
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Forward non-key messages to providers wizard form
	if m.providersWizard != nil && m.providersWizard.InputForm != nil {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			form, cmd := m.providersWizard.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.providersWizard.InputForm = f
				if f.State == huh.StateCompleted {
					return m, m.nextProvidersStep()
				}
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	switch msg := msg.(type) {
	case clearNotificationMsg:
		m.notification = nil
		return m, nil

	case mcpStatusMsg:
		m.mcpStatus = msg.Statuses
		return m, nil

	case mcpRefreshTickMsg:
		return m, tea.Batch(
			fetchMCPStatusCmd(m.config.MCP),
			scheduleMCPRefreshCmd(m.config.MCP.RefreshSeconds),
		)

	case commandDoneMsg:
		if msg.command != "" && m.config.History.Enabled {
			entry := config.HistoryEntry{
				Command:   msg.command,
				Tool:      msg.tool,
				Timestamp: time.Now(),
				Success:   msg.success,
			}
			m.history = config.AddToHistory(m.history, entry, m.config.History.MaxItems)
			if m.config.History.Persist {
				config.SaveHistory(m.history)
			}
		}
		// Reload resources if we were editing
		if m.pendingResourceReload {
			m.pendingResourceReload = false
			m.loadResources()
		}
		// Reload config if we were editing preferences
		if m.pendingConfigReload {
			m.pendingConfigReload = false
			m.config = config.Load(mcppkg.GetDefaultMCPServerURL())
			// Update favorites map
			m.favorites = make(map[string]bool)
			for _, f := range m.config.Favorites {
				m.favorites[f] = true
			}
		}
		return m, nil

	case termStartMsg:
		m.term = EmbeddedTerm{
			active:  true,
			focused: true,
			vt:      msg.vt,
			pty:     msg.pty,
			width:   msg.width,
			height:  msg.height,
			command: msg.command,
		}

		go func() {
			// Redirect vterm debug logs to file instead of stdout
			logPath := filepath.Join(config.DataDir, "terminal.log")
			os.MkdirAll(config.DataDir, 0755)
			if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
				log.SetOutput(logFile)
				defer logFile.Close()
				defer log.SetOutput(os.Stderr)
			}
			reader := bufio.NewReader(msg.pty)
			msg.vt.ProcessStdout(reader)
		}()

		waitCmd := func() tea.Msg {
			err := msg.cmd.Wait()
			return termExitMsg{err: err}
		}

		return m, tea.Batch(m.waitForTermOutput(), waitCmd)

	case termOutputMsg:
		if m.term.active && !m.term.exited {
			return m, m.waitForTermOutput()
		}
		return m, nil

	case termExitMsg:
		m.term.exited = true
		m.term.exitErr = msg.err
		m.term.focused = false
		return m, nil

	case agentInteractionMsg:
		m.agentHistory = config.AddAgentInteraction(m.agentHistory, msg.interaction, 20)
		config.SaveAgentHistory(m.agentHistory)
		return m, nil

	case aiAgentResultMsg:
		m.palette.State = PaletteStateShowingResult
		m.palette.ResultTitle = msg.title
		m.palette.ResultText = msg.output
		return m, nil

	case aiPrefilledParamsMsg:
		if m.palette.PendingTool != nil {
			m.palette.PendingTool.Args = msg.params
			return m, m.buildParameterFormWithValues(msg.params)
		}
		return m, nil

	case staticOutputMsg:
		m.term = EmbeddedTerm{
			active:       true,
			focused:      false,
			staticOutput: msg.output,
			staticTitle:  msg.title,
			exited:       true,
		}

		if m.palette.State == PaletteStateExecuting {
			m.closePalette()
		}

		return m, nil

	case deployWizardAccountsMsg:
		if m.palette.WizardState != nil {
			m.palette.WizardState.Data["accounts"] = msg.accounts
			m.palette.WizardState.Data["accounts_loaded"] = true
			m.palette.WizardState.Data["accounts_loading"] = false
			m.palette.WizardState.Data["accounts_error"] = ""
			return m, m.nextDeployStep()
		}
		return m, nil

	case deployWizardDeploymentsMsg:
		if m.palette.WizardState != nil {
			m.palette.WizardState.Data["deployments"] = msg.deployments
			m.palette.WizardState.Data["deployments_loaded"] = true
			m.palette.WizardState.Data["deployments_loading"] = false
			m.palette.WizardState.Data["deployments_error"] = ""
			return m, m.nextDeployStep()
		}
		return m, nil

	case deployWizardErrorMsg:
		if m.palette.WizardState != nil {
			ws := m.palette.WizardState
			if msg.step == 0 {
				ws.Data["accounts_error"] = msg.message
				ws.Data["accounts_loading"] = false
			}
			if msg.step == 1 {
				ws.Data["deployments_error"] = msg.message
				ws.Data["deployments_loading"] = false
			}
			return m, m.nextDeployStep()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.currentView == viewDetail {
			m.viewReady = false
			m.initViewComponents()
		}

	case tickMsg:
		if m.currentView == viewDashboard {
			quote := `"It is with us and in control"`
			m.quoteTarget = float64(len(quote))
			m.quotePos, m.quoteVel = m.spring.Update(m.quotePos, m.quoteVel, m.quoteTarget)
		}
		return m, tickCmd()

	case tea.KeyMsg:
		keyStr := msg.String()

		if keyStr == "f1" && m.term.active {
			m.term.focused = !m.term.focused
			return m, nil
		}

		if m.term.active && m.term.focused && !m.term.exited {
			return m, m.sendKeyToTerminal(msg)
		}

		if keyStr == "esc" && m.term.active && !m.term.focused {
			m.closeTerminal()
			return m, nil
		}

		if m.palette.State != PaletteStateIdle {
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
							cmd := item.Handler(&m)
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

		if keyStr == "ctrl+k" {
			m.openPalette()
			return m, nil
		}

		if m.currentView == viewDetail && m.viewReady {
			switch msg.String() {
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

		// Handle Add Resource wizard form if active
		if m.addResourceWizard != nil && m.addResourceWizard.InputForm != nil {
			switch keyStr {
			case "esc":
				m.addResourceWizard = nil
				return m, nil
			}

			form, cmd := m.addResourceWizard.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.addResourceWizard.InputForm = f
				if f.State == huh.StateCompleted {
					return m, m.nextAddResourceStep()
				}
			}
			return m, cmd
		}

		// Handle Preferences wizard form if active
		if m.preferencesWizard != nil && m.preferencesWizard.InputForm != nil {
			switch keyStr {
			case "esc":
				m.preferencesWizard = nil
				return m, nil
			}

			form, cmd := m.preferencesWizard.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.preferencesWizard.InputForm = f
				if f.State == huh.StateCompleted {
					return m, m.nextPreferencesStep()
				}
			}
			return m, cmd
		}

		// Handle Providers wizard form if active
		if m.providersWizard != nil && m.providersWizard.InputForm != nil {
			switch keyStr {
			case "esc":
				m.providersWizard = nil
				return m, nil
			}

			form, cmd := m.providersWizard.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.providersWizard.InputForm = f
				if f.State == huh.StateCompleted {
					return m, m.nextProvidersStep()
				}
			}
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab", "shift+tab":
			// Switch between Resources and Actions tabs
			if msg.String() == "tab" {
				m.dashboardTab = (m.dashboardTab + 1) % 2
			} else {
				m.dashboardTab = (m.dashboardTab + 1) % 2
			}
			return m, nil

		case "enter":
			if m.dashboardTab == 0 {
				// Resources tab - open resource detail
				m.currentView = viewDetail
				m.secCursor = 0
				m.initViewComponents()
			} else {
				// Actions tab - execute action
				if len(m.actionItems) > 0 && m.actionCursor < len(m.actionItems) {
					action := m.actionItems[m.actionCursor]
					if action.Handler != nil {
						return m, action.Handler(&m)
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

		case "up", "k":
			if m.dashboardTab == 0 {
				if m.resCursor > 0 {
					m.resCursor--
				}
			} else {
				if m.actionCursor > 0 {
					m.actionCursor--
				}
			}

		case "down", "j":
			if m.dashboardTab == 0 {
				if m.resCursor < len(m.resources)-1 {
					m.resCursor++
				}
			} else {
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
			} else {
				if idx < len(m.actionItems) {
					m.actionCursor = idx
					action := m.actionItems[m.actionCursor]
					if action.Handler != nil {
						return m, action.Handler(&m)
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

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

func (m *model) sendKeyToTerminal(msg tea.KeyMsg) tea.Cmd {
	if m.term.pty == nil {
		return nil
	}

	var b []byte
	switch msg.Type {
	case tea.KeyEnter:
		b = []byte{'\r'}
	case tea.KeyBackspace:
		b = []byte{127}
	case tea.KeyTab:
		b = []byte{'\t'}
	case tea.KeyEscape:
		b = []byte{27}
	case tea.KeyUp:
		b = []byte{27, '[', 'A'}
	case tea.KeyDown:
		b = []byte{27, '[', 'B'}
	case tea.KeyRight:
		b = []byte{27, '[', 'C'}
	case tea.KeyLeft:
		b = []byte{27, '[', 'D'}
	case tea.KeyCtrlC:
		b = []byte{3}
	case tea.KeyCtrlD:
		b = []byte{4}
	case tea.KeyCtrlZ:
		b = []byte{26}
	case tea.KeyCtrlL:
		b = []byte{12}
	case tea.KeyCtrlA:
		b = []byte{1}
	case tea.KeyCtrlE:
		b = []byte{5}
	case tea.KeyCtrlU:
		b = []byte{21}
	case tea.KeyCtrlK:
		b = []byte{11}
	case tea.KeyCtrlW:
		b = []byte{23}
	case tea.KeyRunes:
		b = []byte(string(msg.Runes))
	case tea.KeySpace:
		b = []byte{' '}
	default:
		s := msg.String()
		if len(s) == 1 {
			b = []byte(s)
		}
	}

	if len(b) > 0 {
		m.term.pty.Write(b)
	}
	return nil
}

func (m *model) closeTerminal() {
	if m.term.pty != nil {
		m.term.pty.Close()
	}
	if m.term.vt != nil {
		m.term.vt.Kill()
	}
	m.term = EmbeddedTerm{}
}

type termRenderer struct{}

func (r *termRenderer) HandleCh(ch ecma48.PositionedChar) {}
func (r *termRenderer) SetCursor(x, y int)                {}

func (m *model) waitForTermOutput() tea.Cmd {
	return tea.Tick(time.Millisecond*16, func(t time.Time) tea.Msg {
		return termOutputMsg{}
	})
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}

	var content string

	switch m.currentView {
	case viewDashboard:
		content = m.renderDashboard()
	case viewDetail:
		content = m.renderResourceView()
	default:
		content = m.renderDashboard()
	}

	status := m.renderStatusBar()
	background := lipgloss.JoinVertical(lipgloss.Left, content, status)

	if m.palette.State != PaletteStateIdle {
		palette := m.renderPalette()
		background = overlay.Composite(palette, background, overlay.Center, overlay.Center, 0, 0)
	}

	if m.notification != nil {
		toast := m.renderNotification()
		toastW := lipgloss.Width(toast)
		offsetX := m.width - toastW - 4
		if offsetX < 0 {
			offsetX = 0
		}
		background = overlay.Composite(toast, background, overlay.Top, overlay.Left, offsetX, 1)
	}

	return background
}

// Run is the public entry point for the TUI application.
func Run(startResource string) error {
	_, err := tea.NewProgram(newModel(startResource), tea.WithAltScreen()).Run()
	return err
}
