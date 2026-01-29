package app

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/htelsiz/skitz/internal/ai"
	"github.com/htelsiz/skitz/internal/config"
)

func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func (m *model) startAddResourceWizard() tea.Cmd {
	m.addResourceWizard = &AddResourceWizard{
		Step:     0,
		Name:     "",
		Template: "blank",
	}
	return m.buildAddResourceForm()
}

func (m *model) buildAddResourceForm() tea.Cmd {
	wizard := m.addResourceWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0:
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

	case 1:
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

	case 2:
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

func (m *model) nextAddResourceStep() tea.Cmd {
	wizard := m.addResourceWizard
	if wizard == nil {
		return nil
	}

	wizard.Step++
	if wizard.Step > 2 {
		return m.createResourceFile()
	}

	return m.buildAddResourceForm()
}

func (m *model) createResourceFile() tea.Cmd {
	wizard := m.addResourceWizard
	if wizard == nil || wizard.Name == "" {
		m.addResourceWizard = nil
		return m.showNotification("!", "Resource name cannot be empty", "error")
	}

	name := strings.TrimSpace(wizard.Name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")

	var content string
	switch wizard.Template {
	case "commands":
		content = fmt.Sprintf("# %s\n\n## Commands\n\n`example-command` Example description ^run\n", name)
	case "detailed":
		content = fmt.Sprintf("# %s\n\n## Overview\n\nAdd overview here.\n\n## Commands\n\n`example-command` Example description ^run\n\n## Configuration\n\nAdd configuration notes here.\n", name)
	default:
		content = fmt.Sprintf("# %s\n\n", name)
	}

	if err := os.MkdirAll(config.ResourcesDir, 0755); err != nil {
		m.addResourceWizard = nil
		return m.showNotification("!", "Failed to create directory: "+err.Error(), "error")
	}

	filePath := filepath.Join(config.ResourcesDir, name+".md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		m.addResourceWizard = nil
		return m.showNotification("!", "Failed to create file: "+err.Error(), "error")
	}

	m.loadResources()
	m.addResourceWizard = nil
	m.dashboardTab = 0

	return m.showNotification("", fmt.Sprintf("Created resource: %s", name), "success")
}

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

func (m *model) buildPreferencesForm() tea.Cmd {
	wizard := m.preferencesWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0:
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

	case 1:
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
			m.preferencesWizard = nil
			return m.openConfigInEditor()
		}

	case 2:
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

func (m *model) nextPreferencesStep() tea.Cmd {
	wizard := m.preferencesWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0:
		wizard.Step = 1
		return m.buildPreferencesForm()

	case 1:
		switch wizard.Section {
		case "history":
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

	case 2:
		if wizard.MCPName == "" || wizard.MCPURL == "" {
			m.preferencesWizard = nil
			return m.showNotification("!", "Name and URL are required", "error")
		}

		if strings.HasPrefix(wizard.MCPAction, "edit:") {
			oldName := strings.TrimPrefix(wizard.MCPAction, "edit:")
			for i, srv := range m.config.MCP.Servers {
				if srv.Name == oldName {
					m.config.MCP.Servers[i].Name = wizard.MCPName
					m.config.MCP.Servers[i].URL = wizard.MCPURL
					break
				}
			}
		} else {
			m.config.MCP.Servers = append(m.config.MCP.Servers, config.MCPServerConfig{
				Name: wizard.MCPName,
				URL:  wizard.MCPURL,
			})
		}
		config.Save(m.config)
		m.preferencesWizard = nil
		return m.showNotification("✓", "MCP server saved", "success")
	}

	return nil
}

func (m *model) startProvidersWizard() tea.Cmd {
	m.providersWizard = &ProvidersWizard{
		Step:    0,
		Enabled: true,
	}
	return m.buildProvidersForm()
}

func (m *model) buildProvidersForm() tea.Cmd {
	wizard := m.providersWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0:
		var options []huh.Option[string]
		options = append(options, huh.NewOption("Add Provider", "add"))

		for _, p := range m.config.AI.Providers {
			status := "disabled"
			if p.Enabled {
				status = "enabled"
			}
			if p.Name == m.config.AI.DefaultProvider {
				status = "default"
			}
			provType := p.ProviderType
			if provType == "" {
				provType = ai.DetectProviderType(p.APIKey, p.BaseURL, p.Name)
			}
			options = append(options, huh.NewOption(fmt.Sprintf("Edit: %s [%s] (%s)", p.Name, provType, status), "edit:"+p.Name))
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

	case 1:
		if wizard.ProviderType == "" && wizard.APIKey != "" {
			wizard.ProviderType = ai.DetectProviderType(wizard.APIKey, wizard.BaseURL, wizard.Name)
		}

		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Provider Type").
					Description("Select the LLM provider API type").
					Options(
						huh.NewOption("OpenAI", "openai"),
						huh.NewOption("Anthropic (Claude)", "anthropic"),
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

	case 2:
		var fields []huh.Field

		fields = append(fields,
			huh.NewInput().
				Title("Name").
				Description("A friendly name for this provider").
				Placeholder(wizard.ProviderType).
				Value(&wizard.Name),
		)

		if wizard.ProviderType != "ollama" {
			keyDesc := "Your API key (stored locally)"
			if wizard.ProviderType == "anthropic" {
				keyDesc = "Anthropic API key (starts with sk-ant-)"
			} else if wizard.ProviderType == "openai" {
				keyDesc = "OpenAI API key (starts with sk-)"
			}
			fields = append(fields,
				huh.NewInput().
					Title("API Key").
					Description(keyDesc).
					EchoMode(huh.EchoModePassword).
					Value(&wizard.APIKey),
			)
		}

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

	case 3:
		wizard.InputForm = nil
		wizard.Testing = true
		return m.testProviderConnection()

	case 4:
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

func (m *model) testProviderConnection() tea.Cmd {
	wizard := m.providersWizard
	if wizard == nil {
		return nil
	}

	return func() tea.Msg {
		provider := config.ProviderConfig{
			Name:         wizard.Name,
			ProviderType: wizard.ProviderType,
			APIKey:       wizard.APIKey,
			BaseURL:      wizard.BaseURL,
			DefaultModel: wizard.DefaultModel,
			Enabled:      true,
		}

		client := ai.NewClient(provider)
		err := client.TestConnection()

		return providerTestMsg{
			success: err == nil,
			err:     err,
		}
	}
}

func (m *model) nextProvidersStep() tea.Cmd {
	wizard := m.providersWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0:
		if wizard.Action == "add" {
			wizard.Step = 1
			return m.buildProvidersForm()
		} else if wizard.Action == "default" {
			wizard.Step = 4
			return m.buildProvidersForm()
		} else if strings.HasPrefix(wizard.Action, "edit:") {
			providerName := strings.TrimPrefix(wizard.Action, "edit:")
			for _, p := range m.config.AI.Providers {
				if p.Name == providerName {
					wizard.Name = p.Name
					wizard.APIKey = p.APIKey
					wizard.BaseURL = p.BaseURL
					wizard.DefaultModel = p.DefaultModel
					wizard.Enabled = p.Enabled
					wizard.ProviderType = p.ProviderType
					if wizard.ProviderType == "" {
						wizard.ProviderType = ai.DetectProviderType(p.APIKey, p.BaseURL, p.Name)
					}
					break
				}
			}
			wizard.Step = 1
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

	case 1:
		if wizard.Name == "" {
			wizard.Name = wizard.ProviderType
		}
		switch wizard.ProviderType {
		case "ollama":
			if wizard.BaseURL == "" {
				wizard.BaseURL = "http://localhost:11434"
			}
			if wizard.DefaultModel == "" {
				wizard.DefaultModel = "llama3"
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

	case 2:
		if wizard.Name == "" {
			wizard.Name = wizard.ProviderType
		}

		if wizard.ProviderType == "anthropic" && wizard.APIKey != "" && !strings.HasPrefix(wizard.APIKey, "sk-ant-") {
			return m.showNotification("!", "Anthropic keys start with sk-ant-", "warning")
		}
		if wizard.ProviderType == "openai" && wizard.APIKey != "" && !strings.HasPrefix(wizard.APIKey, "sk-") {
			return m.showNotification("!", "OpenAI keys start with sk-", "warning")
		}

		wizard.Step = 3
		wizard.Testing = true
		wizard.TestResult = ""
		wizard.TestError = ""
		return m.buildProvidersForm()

	case 3:
		return nil

	case 4:
		m.config.AI.DefaultProvider = wizard.Name
		config.Save(m.config)
		m.providersWizard = nil
		return m.showNotification("✓", "Default provider: "+wizard.Name, "success")
	}

	m.providersWizard = nil
	return nil
}

func (m *model) saveProvider() tea.Cmd {
	wizard := m.providersWizard
	if wizard == nil {
		return nil
	}

	newProvider := config.ProviderConfig{
		Name:         wizard.Name,
		ProviderType: wizard.ProviderType,
		APIKey:       wizard.APIKey,
		BaseURL:      wizard.BaseURL,
		DefaultModel: wizard.DefaultModel,
		Enabled:      wizard.Enabled,
	}

	isEdit := strings.HasPrefix(wizard.Action, "edit:")
	if isEdit {
		oldName := strings.TrimPrefix(wizard.Action, "edit:")
		found := false
		for i, p := range m.config.AI.Providers {
			if p.Name == oldName {
				m.config.AI.Providers[i] = newProvider
				found = true
				if m.config.AI.DefaultProvider == oldName && oldName != wizard.Name {
					m.config.AI.DefaultProvider = wizard.Name
				}
				break
			}
		}
		if !found {
			m.config.AI.Providers = append(m.config.AI.Providers, newProvider)
		}
	} else {
		m.config.AI.Providers = append(m.config.AI.Providers, newProvider)
	}

	if m.config.AI.DefaultProvider == "" && newProvider.Enabled {
		m.config.AI.DefaultProvider = newProvider.Name
	}

	config.Save(m.config)
	m.providersWizard = nil

	action := "added"
	if isEdit {
		action = "updated"
	}
	return m.showNotification("✓", fmt.Sprintf("Provider %s %s", wizard.Name, action), "success")
}

func (m *model) startDeleteResourceWizard() tea.Cmd {
	res := m.currentResource()
	if res == nil {
		return m.showNotification("!", "No resource selected", "error")
	}

	m.deleteResourceWizard = &DeleteResourceWizard{
		ResourceName: res.name,
		IsEmbedded:   res.embedded,
		Confirmed:    false,
	}

	return m.buildDeleteResourceForm()
}

func (m *model) buildDeleteResourceForm() tea.Cmd {
	wizard := m.deleteResourceWizard
	if wizard == nil {
		return nil
	}

	title := "Confirm Deletion"
	description := fmt.Sprintf("Are you sure you want to delete '%s'?", wizard.ResourceName)
	if wizard.IsEmbedded {
		description = fmt.Sprintf("Delete your customizations to '%s'?\nThe default version will be restored.", wizard.ResourceName)
	}

	wizard.InputForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Delete").
				Negative("Cancel").
				Value(&wizard.Confirmed),
		),
	).
		WithWidth(50).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(huh.ThemeCatppuccin())

	return wizard.InputForm.Init()
}

func (m *model) confirmDeleteResource() tea.Cmd {
	wizard := m.deleteResourceWizard
	if wizard == nil {
		return nil
	}

	if !wizard.Confirmed {
		m.deleteResourceWizard = nil
		return nil
	}

	filePath := filepath.Join(config.ResourcesDir, wizard.ResourceName+".md")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		m.deleteResourceWizard = nil
		if wizard.IsEmbedded {
			return m.showNotification("!", "Cannot delete built-in resource", "error")
		}
		return m.showNotification("!", "Resource file not found", "error")
	}

	if err := os.Remove(filePath); err != nil {
		m.deleteResourceWizard = nil
		return m.showNotification("!", "Failed to delete: "+err.Error(), "error")
	}

	detailPath := filepath.Join(config.ResourcesDir, wizard.ResourceName+"-detail.md")
	os.Remove(detailPath)

	resourceName := wizard.ResourceName
	wasEmbedded := wizard.IsEmbedded
	m.deleteResourceWizard = nil

	m.loadResources()

	if m.resCursor >= len(m.resources) {
		m.resCursor = max(0, len(m.resources)-1)
	}

	if wasEmbedded {
		return m.showNotification("✓", fmt.Sprintf("Restored default: %s", resourceName), "success")
	}
	return m.showNotification("✓", fmt.Sprintf("Deleted: %s", resourceName), "success")
}

// Run Agent Wizard

func (m *model) startRunAgentWizard() tea.Cmd {
	// Check if any providers are configured
	var enabledProviders []config.ProviderConfig
	for _, p := range m.config.AI.Providers {
		if p.Enabled {
			enabledProviders = append(enabledProviders, p)
		}
	}

	if len(enabledProviders) == 0 {
		return m.showNotification("!", "No providers configured. Go to Configure Providers first.", "error")
	}

	m.runAgentWizard = &RunAgentWizard{
		Step:     0,
		Provider: m.config.AI.DefaultProvider,
		Runtime:  "docker",
		Image:    "skitz-fastagent",
	}
	return m.buildRunAgentForm()
}

func (m *model) buildRunAgentForm() tea.Cmd {
	wizard := m.runAgentWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0:
		// Step 0: Select provider
		var options []huh.Option[string]
		for _, p := range m.config.AI.Providers {
			if p.Enabled {
				label := p.Name
				if p.Name == m.config.AI.DefaultProvider {
					label += " (default)"
				}
				options = append(options, huh.NewOption(label, p.Name))
			}
		}

		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Provider").
					Description("Which AI provider should the agent use?").
					Options(options...).
					Value(&wizard.Provider),
			),
		).
			WithWidth(60).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 1:
		// Step 1: Select runtime
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Runtime").
					Description("Where should the agent run?").
					Options(
						huh.NewOption("Docker - Local container", "docker"),
						huh.NewOption("E2B - Cloud sandbox", "e2b"),
					).
					Value(&wizard.Runtime),
			),
		).
			WithWidth(60).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 2:
		// Step 2: Configure agent
		var fields []huh.Field

		fields = append(fields,
			huh.NewInput().
				Title("Agent Name").
				Description("A name for this agent run").
				Placeholder("my-agent").
				Value(&wizard.AgentName),
			huh.NewText().
				Title("Prompt").
				Description("What should the agent do? (sent directly to the AI)").
				Placeholder("Analyze the code and suggest improvements...").
				CharLimit(2000).
				Value(&wizard.Task),
		)

		if wizard.Runtime == "docker" {
			fields = append(fields,
				huh.NewInput().
					Title("Docker Image").
					Description("Image with fast-agent (build with: docker build -t skitz-fastagent docker/fastagent)").
					Placeholder("skitz-fastagent").
					Value(&wizard.Image),
			)
		}

		wizard.InputForm = huh.NewForm(huh.NewGroup(fields...)).
			WithWidth(60).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 3:
		// Step 3: Confirm
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Run Agent?").
					Description(fmt.Sprintf("Run '%s' with %s using %s?", wizard.AgentName, wizard.Provider, wizard.Runtime)).
					Affirmative("Run").
					Negative("Cancel").
					Value(&wizard.Confirmed),
			),
		).
			WithWidth(60).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()
	}

	return nil
}

func (m *model) nextRunAgentStep() tea.Cmd {
	wizard := m.runAgentWizard
	if wizard == nil {
		return nil
	}

	wizard.Step++
	if wizard.Step > 3 {
		return m.executeRunAgent()
	}

	return m.buildRunAgentForm()
}

func (m *model) executeRunAgent() tea.Cmd {
	wizard := m.runAgentWizard
	if wizard == nil {
		log.Println("executeRunAgent: wizard is nil")
		return nil
	}

	log.Printf("executeRunAgent: confirmed=%v runtime=%s agent=%s provider=%s", wizard.Confirmed, wizard.Runtime, wizard.AgentName, wizard.Provider)

	if !wizard.Confirmed {
		log.Println("executeRunAgent: not confirmed, cancelling")
		m.runAgentWizard = nil
		return nil
	}

	// Find the selected provider
	var provider *config.ProviderConfig
	for _, p := range m.config.AI.Providers {
		if p.Name == wizard.Provider {
			provider = &p
			break
		}
	}

	if provider == nil {
		m.runAgentWizard = nil
		return m.showNotification("!", "Provider not found: "+wizard.Provider, "error")
	}

	agentName := wizard.AgentName
	if agentName == "" {
		agentName = "skitz-agent"
	}

	task := wizard.Task
	if task == "" {
		task = "Say hello and introduce yourself."
	}

	runtime := wizard.Runtime
	providerName := wizard.Provider
	m.runAgentWizard = nil

	// Generate unique ID for this agent run
	agentID := uuid.New().String()

	// Create ActiveAgent entry
	activeAgent := ActiveAgent{
		ID:        agentID,
		Name:      agentName,
		Provider:  providerName,
		Runtime:   runtime,
		StartTime: time.Now(),
		Status:    "running",
		Task:      task,
	}

	// Add to active agents immediately
	m.activeAgents = append(m.activeAgents, activeAgent)

	if runtime == "docker" {
		if _, err := exec.LookPath("docker"); err != nil {
			// Remove from active agents on error
			m.removeActiveAgent(agentID)
			return m.showNotification("!", "Docker not found. Install from https://docs.docker.com/get-docker/", "error")
		}

		image := wizard.Image
		if image == "" {
			image = "astral/uv:python3.12-bookworm-slim"
		}

		// Determine model and env var based on provider type
		model := provider.DefaultModel
		envVar := ""
		apiKeyValue := provider.APIKey

		// Map common model names to fast-agent compatible names
		modelMap := map[string]string{
			"claude-sonnet-4-20250514": "sonnet",
			"claude-3-5-sonnet":        "sonnet",
			"claude-3-sonnet":          "sonnet",
			"claude-3-haiku":           "haiku",
		}
		if mapped, ok := modelMap[model]; ok {
			model = mapped
		}

		switch provider.ProviderType {
		case "openai":
			if model == "" {
				model = "gpt-5"
			}
			envVar = "OPENAI_API_KEY"
		case "anthropic":
			if model == "" {
				model = "sonnet"
			}
			envVar = "ANTHROPIC_API_KEY"
		default:
			if model == "" {
				model = "gpt-5"
			}
			envVar = "OPENAI_API_KEY"
		}

		log.Printf("executeRunAgent: using provider=%s type=%s model=%s agentID=%s", provider.Name, provider.ProviderType, model, agentID)

		// Use skitz-fastagent image with env vars for prompt and model
		cmd := fmt.Sprintf(`docker run --rm --name %s -e %s=%s -e AGENT_MODEL=%s -e AGENT_PROMPT=%q %s`,
			agentName, envVar, apiKeyValue, model, task, image)
		log.Printf("executeRunAgent: running docker command (key redacted)")

		// Return both the agent started message and the run command
		return tea.Batch(
			func() tea.Msg {
				return agentStartedMsg{agent: activeAgent}
			},
			m.runAgentCommand(CommandSpec{
				Command: cmd,
				Mode:    CommandEmbedded,
			}, agentID),
		)
	}

	// E2B runtime
	if _, err := exec.LookPath("e2b"); err != nil {
		m.removeActiveAgent(agentID)
		return m.showNotification("!", "E2B CLI not found. Install with: npm install -g @e2b/cli", "error")
	}

	// For E2B, mark as completed immediately since it's just preparation
	return tea.Batch(
		func() tea.Msg {
			return agentCompletedMsg{
				agentID:  agentID,
				success:  true,
				output:   "E2B sandbox ready. Use e2b CLI to interact.",
				duration: 0,
			}
		},
		m.showNotification("✓", fmt.Sprintf("E2B agent '%s' ready. Use e2b CLI to spawn sandbox.", agentName), "success"),
	)
}

// removeActiveAgent removes an agent from the active list
func (m *model) removeActiveAgent(agentID string) {
	for i, agent := range m.activeAgents {
		if agent.ID == agentID {
			m.activeAgents = append(m.activeAgents[:i], m.activeAgents[i+1:]...)
			return
		}
	}
}

// runAgentCommand runs a command and tracks agent completion
func (m *model) runAgentCommand(spec CommandSpec, agentID string) tea.Cmd {
	// Find the active agent to get start time
	var startTime time.Time
	for _, agent := range m.activeAgents {
		if agent.ID == agentID {
			startTime = agent.StartTime
			break
		}
	}

	return func() tea.Msg {
		// Run the command and capture output
		cmd := exec.Command("sh", "-c", spec.Command)
		output, err := cmd.CombinedOutput()

		duration := time.Since(startTime).Milliseconds()
		success := err == nil

		return agentCompletedMsg{
			agentID:  agentID,
			success:  success,
			output:   string(output),
			duration: duration,
		}
	}
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

// Saved Agent Wizard

func (m *model) startSavedAgentWizard(agent config.SavedAgentConfig) tea.Cmd {
	// Check if any providers are configured
	var enabledProviders []config.ProviderConfig
	for _, p := range m.config.AI.Providers {
		if p.Enabled {
			enabledProviders = append(enabledProviders, p)
		}
	}

	if len(enabledProviders) == 0 {
		return m.showNotification("!", "No AI providers configured. Go to Actions > Configure Providers first.", "error")
	}

	m.savedAgentWizard = &SavedAgentWizard{
		Step:      0,
		AgentID:   agent.ID,
		AgentName: agent.Name,
		Image:     agent.Image,
		BuildPath: agent.BuildPath,
	}
	return m.buildSavedAgentForm()
}

func (m *model) buildSavedAgentForm() tea.Cmd {
	wizard := m.savedAgentWizard
	if wizard == nil {
		return nil
	}

	switch wizard.Step {
	case 0:
		// Step 0: Select provider
		var options []huh.Option[string]
		for _, p := range m.config.AI.Providers {
			if p.Enabled {
				label := p.Name
				if p.Name == m.config.AI.DefaultProvider {
					label += " (default)"
				}
				options = append(options, huh.NewOption(label, p.Name))
			}
		}

		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Provider").
					Description("Which AI provider should the agent use?").
					Options(options...).
					Value(&wizard.Provider),
			),
		).
			WithWidth(60).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 1:
		// Step 1: Select resource
		var options []huh.Option[string]
		for _, res := range m.resources {
			options = append(options, huh.NewOption(res.name, res.name))
		}

		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Resource").
					Description("Which resource should be verified?").
					Options(options...).
					Value(&wizard.Resource),
			),
		).
			WithWidth(60).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 2:
		// Step 2: Enter additional instructions (optional)
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Additional Instructions (optional)").
					Description("Any specific commands or areas to focus on?").
					Placeholder("Focus on pod creation commands...").
					Value(&wizard.Prompt),
			),
		).
			WithWidth(60).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()

	case 3:
		// Step 3: Confirm
		desc := fmt.Sprintf("Verify %s resource with %s", wizard.Resource, wizard.Provider)
		if wizard.Prompt != "" {
			desc += fmt.Sprintf("\nInstructions: %s", truncate(wizard.Prompt, 40))
		}
		wizard.InputForm = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Run Agent?").
					Description(desc).
					Affirmative("Run").
					Negative("Cancel").
					Value(&wizard.Confirmed),
			),
		).
			WithWidth(60).
			WithShowHelp(true).
			WithShowErrors(true).
			WithTheme(huh.ThemeCatppuccin())
		return wizard.InputForm.Init()
	}

	return nil
}

func (m *model) nextSavedAgentStep() tea.Cmd {
	wizard := m.savedAgentWizard
	if wizard == nil {
		return nil
	}

	wizard.Step++
	if wizard.Step > 3 {
		return m.executeSavedAgent()
	}

	return m.buildSavedAgentForm()
}

func (m *model) executeSavedAgent() tea.Cmd {
	wizard := m.savedAgentWizard
	if wizard == nil {
		return nil
	}

	if !wizard.Confirmed {
		m.savedAgentWizard = nil
		return nil
	}

	// Find the selected provider
	var provider *config.ProviderConfig
	for _, p := range m.config.AI.Providers {
		if p.Name == wizard.Provider && p.Enabled {
			provider = &p
			break
		}
	}

	if provider == nil {
		m.savedAgentWizard = nil
		return m.showNotification("!", "Selected provider not found or disabled", "error")
	}

	// Check Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		m.savedAgentWizard = nil
		return m.showNotification("!", "Docker not found. Install from https://docs.docker.com/get-docker/", "error")
	}

	agentName := wizard.AgentName
	image := wizard.Image
	buildPath := wizard.BuildPath
	resource := wizard.Resource
	prompt := wizard.Prompt

	m.savedAgentWizard = nil

	// Determine API key env var
	envVar := "ANTHROPIC_API_KEY"
	if provider.ProviderType == "openai" {
		envVar = "OPENAI_API_KEY"
	}

	// Generate unique ID
	agentID := uuid.New().String()

	// Use container name as ID for easier tracking
	containerName := "skitz-" + agentID[:8] // Use first 8 chars of UUID

	// Create ActiveAgent entry
	activeAgent := ActiveAgent{
		ID:        containerName,
		Name:      agentName,
		Provider:  provider.Name,
		Runtime:   "docker",
		StartTime: time.Now(),
		Status:    "building",
		Task:      prompt,
	}

	// Build and run docker command
	var cmd string
	if buildPath != "" {
		// Build image first, then run with repo mounted read-only
		cmd = fmt.Sprintf(`docker build -t %s %s && docker run --name %s -v "$(pwd):/skitz:ro" -e %s=%s -e AGENT_RESOURCE=%q -e AGENT_PROMPT=%q %s`,
			image, buildPath, containerName, envVar, provider.APIKey, resource, prompt, image)
	} else {
		// Just run (image should exist)
		cmd = fmt.Sprintf(`docker run --name %s -e %s=%s -e AGENT_RESOURCE=%q -e AGENT_PROMPT=%q %s`,
			containerName, envVar, provider.APIKey, resource, prompt, image)
	}

	return tea.Batch(
		func() tea.Msg {
			return agentStartedMsg{agent: activeAgent}
		},
		m.runAgentCommand(CommandSpec{
			Command: cmd,
			Mode:    CommandEmbedded,
		}, agentID),
	)
}
