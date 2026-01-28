package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mark3labs/mcp-go/mcp"
	openai "github.com/sashabaranov/go-openai"

	mcppkg "github.com/htelsiz/skitz/internal/mcp"
)

// PaletteItem represents an item in the command palette
type PaletteItem struct {
	ID          string
	Icon        string
	Title       string
	Subtitle    string
	Category    string
	Shortcut    string
	Handler     func(m *model) tea.Cmd
	ResourceIdx int
	MCPTool      *mcp.Tool
	MCPServer    string
	MCPServerURL string
}

// PaletteState represents the current state of the command palette
type PaletteState int

const (
	PaletteStateIdle             PaletteState = iota
	PaletteStateSearching
	PaletteStateCollectingParams
	PaletteStateAIInput
	PaletteStateExecuting
	PaletteStateShowingResult
)

// Palette state
type Palette struct {
	State       PaletteState
	Query       string
	Items       []PaletteItem
	Filtered    []PaletteItem
	Cursor      int
	InputForm   *huh.Form
	InputValue  string
	PendingTool *mcpPendingTool
	WizardState *wizardState
	LoadingText string
	ResultTitle string
	ResultText  string
}

type mcpPendingTool struct {
	ServerName string
	ServerURL  string
	Tool       mcp.Tool
	Args       map[string]any
	FormValues map[string]*string
	AITask     string
}

type wizardState struct {
	Type    string
	Step    int
	Data    map[string]any
	Options []string
}

func (m *model) buildPaletteItems() []PaletteItem {
	var items []PaletteItem

	items = append(items, PaletteItem{
		ID:       "action:bia_review",
		Icon:     "🔍",
		Title:    "BIA Code Review",
		Subtitle: "Review Python code with BIA Junior Agent",
		Category: "action",
		Handler: func(m *model) tea.Cmd {
			return m.startBIAWizard()
		},
	})

	items = append(items, PaletteItem{
		ID:       "action:deploy_agent",
		Icon:     "🚀",
		Title:    "Deploy Agent",
		Subtitle: "Deploy an AI agent to Azure",
		Category: "action",
		Handler: func(m *model) tea.Cmd {
			return m.startDeployWizard()
		},
	})

	items = append(items, PaletteItem{
		ID:       "action:create_script",
		Icon:     "📜",
		Title:    "Create Script",
		Subtitle: "Generate a new automation script",
		Category: "action",
		Handler: func(m *model) tea.Cmd {
			return m.showNotification("📜", "Create Script - Coming soon!", "info")
		},
	})

	items = append(items, m.getMCPToolItems()...)

	return items
}

func (m *model) getMCPToolItems() []PaletteItem {
	var items []PaletteItem
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for _, server := range m.config.MCP.Servers {
		tools, err := mcppkg.FetchTools(ctx, server.URL)
		if err != nil {
			continue
		}
		for _, tool := range tools {
			items = append(items, m.mcpToolToPaletteItem(server.Name, server.URL, tool))
		}
	}
	return items
}

func (m *model) mcpToolToPaletteItem(serverName string, serverURL string, tool mcp.Tool) PaletteItem {
	toolCopy := tool
	return PaletteItem{
		ID:           fmt.Sprintf("mcp:%s:%s", serverName, tool.Name),
		Icon:         "⚡",
		Title:        tool.Name,
		Subtitle:     truncate(tool.Description, 50),
		Category:     "mcp",
		MCPTool:      &toolCopy,
		MCPServer:    serverName,
		MCPServerURL: serverURL,
	}
}

func executeMCPToolWithArgs(serverURL string, toolName string, args map[string]any) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		client, err := mcppkg.NewClient(serverURL)
		if err != nil {
			return staticOutputMsg{
				title:  toolName,
				output: fmt.Sprintf("Error: Failed to create client: %v", err),
			}
		}

		if err := client.Connect(ctx); err != nil {
			return staticOutputMsg{
				title:  toolName,
				output: fmt.Sprintf("Error: Failed to connect: %v", err),
			}
		}
		defer client.Close()

		result, err := client.CallTool(ctx, toolName, args)
		if err != nil {
			return staticOutputMsg{
				title:  toolName,
				output: fmt.Sprintf("Error: %v", err),
			}
		}

		output, err := extractTextFromResult(result)
		if err != nil {
			return staticOutputMsg{
				title:  toolName,
				output: fmt.Sprintf("Error parsing result: %v", err),
			}
		}

		return staticOutputMsg{
			title:  toolName,
			output: output,
		}
	}
}

func (m *model) startMCPToolWithAI(item PaletteItem) tea.Cmd {
	tool := item.MCPTool
	if tool == nil {
		return nil
	}

	m.palette.PendingTool = &mcpPendingTool{
		ServerName: item.MCPServer,
		ServerURL:  item.MCPServerURL,
		Tool:       *tool,
		Args:       make(map[string]any),
		FormValues: make(map[string]*string),
	}

	m.palette.State = PaletteStateAIInput
	m.palette.Query = ""

	return nil
}

func (m *model) startMCPToolInput(item PaletteItem) tea.Cmd {
	tool := item.MCPTool
	if tool == nil {
		return nil
	}

	if len(tool.InputSchema.Properties) == 0 {
		return executeMCPToolWithArgs(item.MCPServerURL, tool.Name, nil)
	}

	formValues := make(map[string]*string)

	for paramName := range tool.InputSchema.Properties {
		val := ""
		formValues[paramName] = &val
	}

	m.palette.PendingTool = &mcpPendingTool{
		ServerName: item.MCPServer,
		ServerURL:  item.MCPServerURL,
		Tool:       *tool,
		Args:       make(map[string]any),
		FormValues: formValues,
	}

	return m.buildParameterForm()
}

func (m *model) buildParameterFormWithValues(aiParams map[string]interface{}) tea.Cmd {
	pt := m.palette.PendingTool
	if pt == nil {
		return nil
	}

	for paramName := range pt.Tool.InputSchema.Properties {
		if pt.FormValues[paramName] == nil {
			val := ""
			pt.FormValues[paramName] = &val
		}
	}

	for paramName, paramValue := range aiParams {
		var strValue string
		switch v := paramValue.(type) {
		case string:
			strValue = v
		case float64:
			strValue = fmt.Sprintf("%v", v)
		case int:
			strValue = fmt.Sprintf("%d", v)
		case bool:
			strValue = fmt.Sprintf("%t", v)
		default:
			bytes, _ := json.Marshal(v)
			strValue = string(bytes)
		}

		if ptr := pt.FormValues[paramName]; ptr != nil {
			*ptr = strValue
		}
	}

	return m.buildParameterForm()
}

func (m *model) buildParameterForm() tea.Cmd {
	pt := m.palette.PendingTool
	if pt == nil {
		return nil
	}

	required := make(map[string]bool)
	for _, r := range pt.Tool.InputSchema.Required {
		required[r] = true
	}

	var fields []huh.Field
	var paramNames []string

	for paramName := range pt.Tool.InputSchema.Properties {
		paramNames = append(paramNames, paramName)
	}

	sort.Slice(paramNames, func(i, j int) bool {
		reqI := required[paramNames[i]]
		reqJ := required[paramNames[j]]
		if reqI != reqJ {
			return reqI
		}
		return paramNames[i] < paramNames[j]
	})

	for _, paramName := range paramNames {
		paramDef := pt.Tool.InputSchema.Properties[paramName]
		paramMap, ok := paramDef.(map[string]interface{})
		if !ok {
			continue
		}

		field := m.createFormField(paramName, paramMap, required[paramName])
		if field != nil {
			fields = append(fields, field)
		}
	}

	if len(fields) == 0 {
		return nil
	}

	m.palette.InputForm = huh.NewForm(huh.NewGroup(fields...)).
		WithWidth(100).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(huh.ThemeCatppuccin())

	m.palette.State = PaletteStateCollectingParams

	return m.palette.InputForm.Init()
}

func (m *model) createFormField(paramName string, paramMap map[string]interface{}, isRequired bool) huh.Field {
	pt := m.palette.PendingTool
	if pt == nil {
		return nil
	}

	description := ""
	if desc, ok := paramMap["description"].(string); ok {
		description = desc
	}

	paramType := "string"
	if t, ok := paramMap["type"].(string); ok {
		paramType = t
	}

	title := paramName
	if isRequired {
		title = paramName + " *"
	}

	valuePtr := pt.FormValues[paramName]

	if enumVal, ok := paramMap["enum"].([]interface{}); ok && len(enumVal) > 0 {
		options := make([]huh.Option[string], len(enumVal))
		for i, v := range enumVal {
			str := fmt.Sprintf("%v", v)
			options[i] = huh.NewOption(str, str)
		}
		return huh.NewSelect[string]().
			Title(title).
			Description(description).
			Options(options...).
			Value(valuePtr)
	}

	if paramType == "boolean" {
		boolPtr := new(bool)
		return huh.NewConfirm().
			Title(title).
			Description(description).
			Value(boolPtr)
	}

	if paramType == "string" {
		maxLength := 0
		if ml, ok := paramMap["maxLength"].(float64); ok {
			maxLength = int(ml)
		}

		placeholder := description
		if examples, ok := paramMap["examples"].([]interface{}); ok && len(examples) > 0 {
			placeholder = fmt.Sprintf("%v", examples[0])
		}

		useLongText := maxLength > 200 ||
			strings.Contains(strings.ToLower(paramName), "content") ||
			strings.Contains(strings.ToLower(paramName), "text") ||
			strings.Contains(strings.ToLower(paramName), "body") ||
			strings.Contains(strings.ToLower(paramName), "query")

		if useLongText {
			return huh.NewText().
				Title(title).
				Description(description).
				Placeholder(placeholder).
				CharLimit(maxLength).
				Value(valuePtr)
		}

		input := huh.NewInput().
			Title(title).
			Description(description).
			Placeholder(placeholder).
			Value(valuePtr)

		if isRequired {
			input = input.Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("%s is required", paramName)
				}
				return nil
			})
		}

		return input
	}

	if paramType == "number" || paramType == "integer" {
		input := huh.NewInput().
			Title(title).
			Description(description).
			Placeholder("Enter a number").
			Value(valuePtr)

		input = input.Validate(func(s string) error {
			if s == "" && !isRequired {
				return nil
			}
			if s == "" && isRequired {
				return fmt.Errorf("%s is required", paramName)
			}
			if paramType == "integer" {
				if _, err := fmt.Sscanf(s, "%d", new(int)); err != nil {
					return fmt.Errorf("must be an integer")
				}
			} else {
				if _, err := fmt.Sscanf(s, "%f", new(float64)); err != nil {
					return fmt.Errorf("must be a number")
				}
			}
			return nil
		})

		return input
	}

	return huh.NewInput().
		Title(title).
		Description(description).
		Value(valuePtr)
}

func (m *model) handleParameterSubmit() tea.Cmd {
	pt := m.palette.PendingTool
	if pt == nil {
		return m.showNotification("⚠️", "No pending tool found", "error")
	}

	required := make(map[string]bool)
	for _, r := range pt.Tool.InputSchema.Required {
		required[r] = true
	}

	for paramName, valuePtr := range pt.FormValues {
		if valuePtr == nil {
			continue
		}

		value := strings.TrimSpace(*valuePtr)

		if value == "" && !required[paramName] {
			continue
		}

		paramDef := pt.Tool.InputSchema.Properties[paramName]
		paramMap, ok := paramDef.(map[string]interface{})
		if !ok {
			continue
		}

		paramType := "string"
		if t, ok := paramMap["type"].(string); ok {
			paramType = t
		}

		switch paramType {
		case "boolean":
			boolVal := value == "true" || value == "yes" || value == "1"
			pt.Args[paramName] = boolVal

		case "number":
			var floatVal float64
			if _, err := fmt.Sscanf(value, "%f", &floatVal); err == nil {
				pt.Args[paramName] = floatVal
			} else if value != "" {
				return m.showNotification("⚠️", fmt.Sprintf("Invalid number for %s", paramName), "warning")
			}

		case "integer":
			var intVal int
			if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
				pt.Args[paramName] = intVal
			} else if value != "" {
				return m.showNotification("⚠️", fmt.Sprintf("Invalid integer for %s", paramName), "warning")
			}

		case "array", "object":
			if value != "" {
				var jsonValue interface{}
				if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
					pt.Args[paramName] = jsonValue
				} else {
					return m.showNotification("⚠️", fmt.Sprintf("Invalid JSON for %s: %v", paramName, err), "warning")
				}
			}

		default:
			if value != "" {
				pt.Args[paramName] = value
			}
		}
	}

	m.palette.InputForm = nil
	serverURL := pt.ServerURL
	toolName := pt.Tool.Name
	args := pt.Args
	m.palette.PendingTool = nil

	m.palette.State = PaletteStateExecuting
	m.palette.LoadingText = "Executing tool..."

	return executeMCPToolWithArgs(serverURL, toolName, args)
}

func filterPaletteItems(items []PaletteItem, query string) []PaletteItem {
	if query == "" {
		return items
	}

	query = strings.ToLower(query)
	var filtered []PaletteItem

	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Title), query) ||
			strings.Contains(strings.ToLower(item.Subtitle), query) ||
			strings.Contains(strings.ToLower(item.Category), query) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func (m *model) openPalette() {
	m.palette.State = PaletteStateSearching
	m.palette.Query = ""
	m.palette.Items = m.buildPaletteItems()
	m.palette.Filtered = m.palette.Items
	m.palette.Cursor = 0
}

func (m *model) closePalette() {
	m.palette.State = PaletteStateIdle
	m.palette.Query = ""
	m.palette.Cursor = 0
	m.palette.InputForm = nil
	m.palette.PendingTool = nil
	m.palette.WizardState = nil
	m.palette.LoadingText = ""
	m.palette.ResultTitle = ""
	m.palette.ResultText = ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (m model) renderPalette() string {
	paletteWidth := int(float64(m.width) * 0.85)
	paletteHeight := int(float64(m.height) * 0.80)

	if paletteWidth < 100 {
		paletteWidth = 100
	}
	if paletteHeight < 30 {
		paletteHeight = 30
	}

	accentColor := lipgloss.Color("99")

	var lines []string

	switch m.palette.State {
	case PaletteStateCollectingParams:
		if m.palette.InputForm == nil {
			return lipgloss.NewStyle().Render("Error: No form available")
		}
		textStyle := lipgloss.NewStyle().Foreground(subtle)
		keyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238")).
			Padding(0, 1)

		var headerContent string

		if m.palette.WizardState != nil {
			ws := m.palette.WizardState
			wizardTitle := ws.Type
			if ws.Type == "bia" {
				wizardTitle = "BIA Code Review"
			} else if ws.Type == "deploy" {
				wizardTitle = "Deploy Agent"
			}

			headerContent = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(wizardTitle) +
				textStyle.Render(fmt.Sprintf("  Step %d  ", ws.Step+1)) +
				keyStyle.Render("enter") + textStyle.Render(" submit  ") +
				keyStyle.Render("esc") + textStyle.Render(" cancel")
		} else if m.palette.PendingTool != nil {
			pt := m.palette.PendingTool

			required := make(map[string]bool)
			for _, r := range pt.Tool.InputSchema.Required {
				required[r] = true
			}
			reqCount := len(pt.Tool.InputSchema.Required)
			totalCount := len(pt.Tool.InputSchema.Properties)
			optCount := totalCount - reqCount

			aiPrefilled := pt.AITask != ""
			var paramInfo string
			if aiPrefilled {
				paramInfo = "  🤖 AI pre-filled  "
			} else {
				paramInfo = fmt.Sprintf("  %d required", reqCount)
				if optCount > 0 {
					paramInfo += fmt.Sprintf(", %d optional", optCount)
				}
				paramInfo += "  "
			}

			icon := "⚡"
			if strings.Contains(strings.ToLower(pt.ServerName), "datadog") {
				icon = "📊"
			} else if strings.Contains(strings.ToLower(pt.ServerName), "github") {
				icon = "🐙"
			}

			headerContent = lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render(icon+" "+pt.Tool.Name) +
				textStyle.Render(paramInfo) +
				keyStyle.Render("tab") + textStyle.Render(" next  ") +
				keyStyle.Render("enter") + textStyle.Render(" submit  ") +
				keyStyle.Render("esc") + textStyle.Render(" cancel")
		}

		infoBar := lipgloss.NewStyle().
			Background(lipgloss.Color("234")).
			Width(paletteWidth - 4).
			Padding(0, 1).
			Render(headerContent)
		lines = append(lines, infoBar)

		if m.palette.PendingTool != nil && m.palette.State == PaletteStateCollectingParams {
			pt := m.palette.PendingTool
			if pt.Tool.Description != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("245")).
					Italic(true).
					Padding(0, 1)
				desc := truncate(pt.Tool.Description, paletteWidth-8)
				lines = append(lines, descStyle.Render(desc))
			}
		}

		lines = append(lines, "")

		formView := m.palette.InputForm.View()
		lines = append(lines, formView)

		if m.term.active {
			lines = append(lines, "")
			lines = append(lines, m.renderTerminalPane())
		}

		content := lipgloss.JoinVertical(lipgloss.Left, lines...)

		container := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 1)

		return container.Render(content)

	case PaletteStateShowingResult:
		return m.renderPaletteResult(paletteWidth, paletteHeight, accentColor)

	default:
		return m.renderPaletteSplitView(paletteWidth, paletteHeight, accentColor)
	}
}

func (m model) renderPaletteResult(paletteWidth, paletteHeight int, accentColor lipgloss.Color) string {
	var lines []string

	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("234")).
		Foreground(accentColor).
		Bold(true).
		Width(paletteWidth - 4).
		Padding(0, 1)

	lines = append(lines, headerStyle.Render("✓ "+m.palette.ResultTitle))
	lines = append(lines, "")

	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(customStyleJSON)),
		glamour.WithWordWrap(paletteWidth-8),
	)

	var renderedOutput string
	if err == nil {
		renderedOutput, _ = r.Render(m.palette.ResultText)
	} else {
		renderedOutput = m.palette.ResultText
	}

	lines = append(lines, renderedOutput)
	lines = append(lines, "")

	hintStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("234")).
		Foreground(subtle).
		Width(paletteWidth - 4).
		Padding(0, 1)
	lines = append(lines, hintStyle.Render("Press Enter or Esc to close"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	container := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 1).
		Width(paletteWidth).
		Height(paletteHeight)

	return container.Render(content)
}

func (m model) renderPaletteSplitView(paletteWidth, paletteHeight int, accentColor lipgloss.Color) string {
	listWidth := int(float64(paletteWidth) * 0.45)
	previewWidth := paletteWidth - listWidth - 4

	leftPanel := m.renderPaletteList(listWidth, paletteHeight, accentColor)

	rightPanel := m.renderPalettePreview(previewWidth, paletteHeight, accentColor)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		rightPanel,
	)

	container := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1)

	return container.Render(content)
}

func (m model) renderPaletteList(width, height int, accentColor lipgloss.Color) string {
	var lines []string

	countStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(subtle)
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("238")).
		Padding(0, 1)

	var infoContent string
	switch m.palette.State {
	case PaletteStateExecuting:
		toolName := "AI Agent"
		if m.palette.PendingTool != nil {
			toolName = m.palette.PendingTool.Tool.Name
		}
		infoContent = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("🤖 " + toolName) +
			textStyle.Render("  Executing...")

	case PaletteStateAIInput:
		toolName := "AI Agent"
		if m.palette.PendingTool != nil {
			toolName = m.palette.PendingTool.Tool.Name
		}
		infoContent = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("🤖 " + toolName) +
			textStyle.Render("  ") +
			keyStyle.Render("enter") + textStyle.Render(" execute  ") +
			keyStyle.Render("esc") + textStyle.Render(" cancel")

	default:
		infoContent = countStyle.Render(fmt.Sprintf(" %d", len(m.palette.Filtered))) +
			textStyle.Render(" commands  ") +
			keyStyle.Render("↑↓") + textStyle.Render(" select  ") +
			keyStyle.Render("enter") + textStyle.Render(" run  ") +
			keyStyle.Render("ctrl+a") + textStyle.Render(" AI agent")
	}

	infoBar := lipgloss.NewStyle().
		Background(lipgloss.Color("234")).
		Width(width - 2).
		Padding(0, 1).
		Render(infoContent)
	lines = append(lines, infoBar)

	var queryDisplay string
	var searchLine string

	switch m.palette.State {
	case PaletteStateExecuting:
		queryDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(m.palette.Query)
		searchLine = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("🤖 ") + queryDisplay

	case PaletteStateAIInput:
		if m.palette.Query == "" {
			queryDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Italic(true).Render("🤖 Describe what you want the AI to do...")
		} else {
			queryDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render(m.palette.Query) +
				lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("▌")
		}
		searchLine = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("🤖 ") + queryDisplay

	default:
		if m.palette.Query == "" {
			queryDisplay = lipgloss.NewStyle().Foreground(subtle).Italic(true).Render("Type to filter...")
		} else {
			queryDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render(m.palette.Query) +
				lipgloss.NewStyle().Foreground(secondary).Render("▌")
		}
		searchLine = lipgloss.NewStyle().Foreground(secondary).Bold(true).Render("❯ ") + queryDisplay
	}

	searchBar := lipgloss.NewStyle().Padding(1, 1, 0, 1).Render(searchLine)
	lines = append(lines, searchBar)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render(strings.Repeat("─", width-2))
	lines = append(lines, divider)

	maxVisibleItems := height - 8

	switch m.palette.State {
	case PaletteStateExecuting:
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true).
			Padding(2, 1).
			Width(width - 4).
			Align(lipgloss.Center)

		spinner := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
		frame := spinner[0:1]
		lines = append(lines, "")
		lines = append(lines, loadingStyle.Render(frame+" "+m.palette.LoadingText))
		lines = append(lines, "")

		infoStyle := lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true).
			Padding(1, 1).
			Width(width - 4).
			Align(lipgloss.Center)
		lines = append(lines, infoStyle.Render("Please wait..."))

	case PaletteStateAIInput:
		if m.palette.PendingTool != nil {
			tool := m.palette.PendingTool.Tool

			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(1, 1).
				Width(width - 4)
			lines = append(lines, descStyle.Render(tool.Description))

			lines = append(lines, "")
			instructStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("114")).
				Padding(0, 1).
				Width(width - 4)
			lines = append(lines, instructStyle.Render("The AI will analyze your request and determine the appropriate parameter values automatically."))

			lines = append(lines, "")
			lines = append(lines, instructStyle.Render("Examples:"))
			lines = append(lines, instructStyle.Render("  • 'Find all errors from the frontend service in the last hour'"))
			lines = append(lines, instructStyle.Render("  • 'Show me warnings from the database in the past 24 hours'"))
			lines = append(lines, instructStyle.Render("  • 'Get logs for user authentication failures today'"))
		}

	default:
		if len(m.palette.Filtered) == 0 {
			emptyStyle := lipgloss.NewStyle().
				Foreground(subtle).
				Italic(true).
				Padding(2, 1).
				Width(width - 2).
				Align(lipgloss.Center)
			lines = append(lines, emptyStyle.Render("No matching commands"))
		} else {
		items := m.palette.Filtered
		grouped := make(map[string][]PaletteItem)
		var categories []string
		for _, item := range items {
			cat := item.Category
			if cat == "" {
				cat = "other"
			}
			if _, exists := grouped[cat]; !exists {
				categories = append(categories, cat)
			}
			grouped[cat] = append(grouped[cat], item)
		}

		currentIndex := 0
		for _, category := range categories {
			catItems := grouped[category]

			catIcon := "📦"
			catName := strings.Title(category)
			switch category {
			case "action":
				catIcon = "⚡"
				catName = "Actions"
			case "mcp":
				catIcon = "🔌"
				catName = "MCP Tools"
			case "history":
				catIcon = "🕐"
				catName = "Recent"
			case "favorite":
				catIcon = "⭐"
				catName = "Favorites"
			}

			catHeader := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Bold(true).
				Padding(0, 1).
				Render(fmt.Sprintf("%s %s", catIcon, catName))
			lines = append(lines, catHeader)

			for _, item := range catItems {
				if len(lines) >= maxVisibleItems+3 {
					break
				}

				isSelected := currentIndex == m.palette.Cursor

				title := item.Title
				maxTitleLen := width - 10
				if len(title) > maxTitleLen {
					title = title[:maxTitleLen-3] + "..."
				}

				icon := item.Icon
				if icon == "" {
					icon = "•"
				}

				if isSelected {
					itemLine := lipgloss.NewStyle().
						Foreground(lipgloss.Color("255")).
						Background(lipgloss.Color("237")).
						Bold(true).
						Padding(0, 1).
						Width(width - 4).
						Render(fmt.Sprintf("%s %s", icon, title))

					indicator := lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("▶")
					lines = append(lines, " "+indicator+" "+itemLine)
				} else {
					itemLine := lipgloss.NewStyle().
						Foreground(lipgloss.Color("252")).
						Padding(0, 1).
						Render(fmt.Sprintf(" %s %s", icon, title))
					lines = append(lines, "    "+itemLine)
				}

				currentIndex++
			}

			if category != categories[len(categories)-1] {
				lines = append(lines, "")
			}
		}

		if len(items) > maxVisibleItems {
			moreStyle := lipgloss.NewStyle().
				Foreground(subtle).
				Italic(true).
				Padding(1, 1, 0, 1)
			lines = append(lines, moreStyle.Render(fmt.Sprintf("↓ %d more...", len(items)-maxVisibleItems)))
		}
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	panel := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("238"))

	return panel.Render(content)
}

func (m model) renderPalettePreview(width, height int, accentColor lipgloss.Color) string {
	var lines []string

	if m.palette.State == PaletteStateExecuting && m.palette.PendingTool != nil {
		tool := m.palette.PendingTool.Tool

		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true).
			Padding(1, 1, 0, 1)
		lines = append(lines, titleStyle.Render(fmt.Sprintf("🤖 Executing %s", tool.Name)))

		divider := lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Padding(0, 1).
			Render(strings.Repeat("─", width-2))
		lines = append(lines, divider)

		stepStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(1, 1)

		lines = append(lines, stepStyle.Render("✓ Request received: "+m.palette.Query))
		lines = append(lines, stepStyle.Render("⏳ AI analyzing request..."))
		lines = append(lines, stepStyle.Render("⏳ Determining parameters..."))
		lines = append(lines, stepStyle.Render("⏳ Calling MCP tool..."))

		content := lipgloss.JoinVertical(lipgloss.Left, lines...)
		panel := lipgloss.NewStyle().
			Width(width).
			Height(height).
			Padding(0, 1)
		return panel.Render(content)
	}

	if m.palette.State == PaletteStateAIInput && m.palette.PendingTool != nil {
		tool := m.palette.PendingTool.Tool
		lines = append(lines, m.renderMCPToolPreview(&tool, width)...)

		content := lipgloss.JoinVertical(lipgloss.Left, lines...)
		panel := lipgloss.NewStyle().
			Width(width).
			Height(height).
			Padding(0, 1)
		return panel.Render(content)
	}

	var selectedItem *PaletteItem
	if len(m.palette.Filtered) > 0 && m.palette.Cursor < len(m.palette.Filtered) {
		selectedItem = &m.palette.Filtered[m.palette.Cursor]
	}

	if selectedItem == nil {
		emptyStyle := lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true).
			Padding(height/2, 2).
			Width(width).
			Align(lipgloss.Center)
		return emptyStyle.Render("Select a command to see details")
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Padding(1, 1, 0, 1)

	icon := selectedItem.Icon
	if icon == "" {
		icon = "⚡"
	}

	lines = append(lines, titleStyle.Render(fmt.Sprintf("%s %s", icon, selectedItem.Title)))

	if selectedItem.Category != "" {
		badgeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			MarginLeft(1).
			MarginBottom(1)
		lines = append(lines, badgeStyle.Render(strings.ToUpper(selectedItem.Category)))
	}

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Padding(0, 1).
		Render(strings.Repeat("─", width-2))
	lines = append(lines, divider)

	if selectedItem.MCPTool != nil {
		lines = append(lines, m.renderMCPToolPreview(selectedItem.MCPTool, width)...)
	} else {
		if selectedItem.Subtitle != "" {
			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(1, 1).
				Width(width - 2)
			lines = append(lines, descStyle.Render(selectedItem.Subtitle))
		}

		if selectedItem.Shortcut != "" {
			shortcutStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("114")).
				Padding(1, 1)
			lines = append(lines, shortcutStyle.Render(fmt.Sprintf("Shortcut: %s", selectedItem.Shortcut)))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	panel := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1)

	return panel.Render(content)
}

func (m model) renderMCPToolPreview(tool *mcp.Tool, width int) []string {
	var lines []string

	if tool.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1, 1, 1).
			Width(width - 2)
		lines = append(lines, descStyle.Render(tool.Description))
	}

	if len(tool.InputSchema.Properties) > 0 {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true).
			Padding(1, 1, 0, 1)
		lines = append(lines, headerStyle.Render("Parameters:"))

		required := make(map[string]bool)
		for _, r := range tool.InputSchema.Required {
			required[r] = true
		}

		var paramNames []string
		for paramName := range tool.InputSchema.Properties {
			paramNames = append(paramNames, paramName)
		}

		sort.Slice(paramNames, func(i, j int) bool {
			reqI := required[paramNames[i]]
			reqJ := required[paramNames[j]]
			if reqI != reqJ {
				return reqI
			}
			return paramNames[i] < paramNames[j]
		})

		paramCount := 0
		maxParams := 8
		for _, paramName := range paramNames {
			paramDef := tool.InputSchema.Properties[paramName]
			if paramCount >= maxParams {
				moreStyle := lipgloss.NewStyle().
					Foreground(subtle).
					Italic(true).
					Padding(0, 1)
				lines = append(lines, moreStyle.Render(fmt.Sprintf("  ... and %d more", len(tool.InputSchema.Properties)-maxParams)))
				break
			}

			paramMap, ok := paramDef.(map[string]interface{})
			if !ok {
				continue
			}

			paramLabel := paramName
			if required[paramName] {
				paramLabel = paramName + " *"
			}

			paramType := "string"
			if t, ok := paramMap["type"].(string); ok {
				paramType = t
			}

			paramStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")).
				Padding(0, 1)
			typeStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Italic(true)

			lines = append(lines, paramStyle.Render(fmt.Sprintf("  • %s", paramLabel))+" "+typeStyle.Render(fmt.Sprintf("(%s)", paramType)))

			if desc, ok := paramMap["description"].(string); ok && desc != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("246")).
					Padding(0, 1).
					MarginLeft(4).
					Width(width - 8)
				if len(desc) > 80 {
					desc = desc[:77] + "..."
				}
				lines = append(lines, descStyle.Render(desc))
			}

			paramCount++
		}

		reqCount := len(tool.InputSchema.Required)
		totalCount := len(tool.InputSchema.Properties)
		summaryStyle := lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true).
			Padding(1, 1, 0, 1)
		summary := fmt.Sprintf("%d total", totalCount)
		if reqCount > 0 {
			summary = fmt.Sprintf("%d required, %d optional", reqCount, totalCount-reqCount)
		}
		lines = append(lines, summaryStyle.Render(summary))

		lines = append(lines, "")
		aiHintStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Padding(0, 1)
		lines = append(lines, aiHintStyle.Render("💡 Press 'a' to call with AI agent (requires mods)"))
	} else {
		noParamsStyle := lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true).
			Padding(1, 1)
		lines = append(lines, noParamsStyle.Render("No parameters required"))
	}

	return lines
}

type aiProgressMsg struct {
	step string
}

type aiAgentResultMsg struct {
	title  string
	output string
	err    error
}

type aiPrefilledParamsMsg struct {
	params map[string]interface{}
}

func (m *model) executeMCPToolWithAIAgent(pt *mcpPendingTool) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)

		apiKey := m.config.AI.OpenAIAPIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}

		if apiKey == "" {
			return aiAgentResultMsg{
				title:  "🤖 AI Agent Not Available",
				output: "OpenAI API key not configured.\n\nAdd it to `~/.config/skitz/config.yaml`:\n\n```yaml\nai:\n  openai_api_key: \"sk-proj-...\"\n```\n\nOr set the OPENAI_API_KEY environment variable.\n\nTry entering parameters manually by pressing Enter on the tool.",
				err:    fmt.Errorf("no API key"),
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := fmt.Sprintf(`You are helping execute an MCP tool. Based on the user's request, determine the appropriate parameter values.

Tool: %s
Description: %s

Parameters Schema:
%s

User Request: %s

Respond with ONLY a JSON object containing the parameter values. Example: {"param1": "value1", "param2": 123}
Make reasonable assumptions for any missing information.`,
			pt.Tool.Name,
			pt.Tool.Description,
			formatToolSchema(pt.Tool),
			pt.AITask,
		)

		client := openai.NewClient(apiKey)
		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
			Temperature: 0.0,
		})

		if err != nil {
			return aiAgentResultMsg{
				title:  "🤖 AI Agent Error",
				output: fmt.Sprintf("Failed to call OpenAI API: %v\n\nTry entering parameters manually (press Enter on the tool).", err),
				err:    err,
			}
		}

		if len(resp.Choices) == 0 {
			return aiAgentResultMsg{
				title:  "🤖 AI Agent Error",
				output: "No response from AI. Try entering parameters manually.",
				err:    fmt.Errorf("empty response"),
			}
		}

		result := strings.TrimSpace(resp.Choices[0].Message.Content)

		var params map[string]interface{}
		if err := json.Unmarshal([]byte(result), &params); err != nil {
			return aiAgentResultMsg{
				title:  "🤖 " + pt.Tool.Name,
				output: "**The AI response couldn't be parsed as JSON:**\n\n" + result + "\n\n---\n\n*Try entering parameters manually by pressing Enter on the tool.*",
				err:    err,
			}
		}

		return aiPrefilledParamsMsg{
			params: params,
		}
	}
}

func formatToolSchema(tool mcp.Tool) string {
	var schema strings.Builder

	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}

	var paramNames []string
	for paramName := range tool.InputSchema.Properties {
		paramNames = append(paramNames, paramName)
	}
	sort.Strings(paramNames)

	for _, paramName := range paramNames {
		paramDef := tool.InputSchema.Properties[paramName]
		paramMap, ok := paramDef.(map[string]interface{})
		if !ok {
			continue
		}

		reqStr := ""
		if required[paramName] {
			reqStr = " (required)"
		}

		paramType := "string"
		if t, ok := paramMap["type"].(string); ok {
			paramType = t
		}

		desc := ""
		if d, ok := paramMap["description"].(string); ok {
			desc = d
		}

		schema.WriteString(fmt.Sprintf("- %s%s: %s", paramName, reqStr, paramType))
		if desc != "" {
			schema.WriteString(fmt.Sprintf(" - %s", desc))
		}
		schema.WriteString("\n")
	}

	return schema.String()
}

func formatJSON(data map[string]interface{}) string {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(bytes)
}

func (m *model) startBIAWizard() tea.Cmd {
	m.palette.WizardState = &wizardState{
		Type: "bia",
		Step: 0,
		Data: make(map[string]any),
	}
	return m.nextWizardStep()
}

func (m *model) startDeployWizard() tea.Cmd {
	m.palette.WizardState = &wizardState{
		Type: "deploy",
		Step: 0,
		Data: make(map[string]any),
	}
	m.term.staticOutput = ""
	m.term.staticTitle = ""
	return m.nextWizardStep()
}

func (m *model) nextWizardStep() tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil {
		return nil
	}

	switch ws.Type {
	case "bia":
		return m.nextBIAStep()
	case "deploy":
		return m.nextDeployStep()
	}
	return nil
}

func (m *model) nextBIAStep() tea.Cmd {
	ws := m.palette.WizardState

	switch ws.Step {
	case 0:
		m.palette.InputValue = ""
		input := huh.NewSelect[string]().
			Title("How would you like to provide code?").
			Options(
				huh.NewOption("Paste code directly", "paste"),
				huh.NewOption("Enter file path", "file"),
			).
			Value(&m.palette.InputValue)

		m.palette.InputForm = huh.NewForm(huh.NewGroup(input)).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(false).
			WithTheme(huh.ThemeCatppuccin())

		m.palette.State = PaletteStateCollectingParams
		return m.palette.InputForm.Init()

	case 1:
		inputMethod := ws.Data["method"].(string)
		m.palette.InputValue = ""

		if inputMethod == "file" {
			input := huh.NewInput().
				Title("File path").
				Placeholder("/path/to/file.py").
				Value(&m.palette.InputValue)

			m.palette.InputForm = huh.NewForm(huh.NewGroup(input)).
				WithWidth(80).
				WithShowHelp(false).
				WithShowErrors(false).
				WithTheme(huh.ThemeCatppuccin())
		} else {
			input := huh.NewText().
				Title("Paste your code").
				Placeholder("# Your Python code here\ndef hello():\n    print('Hello')").
				CharLimit(10000).
				Value(&m.palette.InputValue)

			m.palette.InputForm = huh.NewForm(huh.NewGroup(input)).
				WithWidth(80).
				WithHeight(15).
				WithShowHelp(false).
				WithShowErrors(false).
				WithTheme(huh.ThemeCatppuccin())
		}

		m.palette.State = PaletteStateCollectingParams
		return m.palette.InputForm.Init()

	case 2:
		m.palette.State = PaletteStateExecuting
		m.palette.LoadingText = "Reviewing code with BIA..."
		m.palette.InputForm = nil

		code := ws.Data["code"].(string)
		if strings.TrimSpace(code) == "" {
			return func() tea.Msg {
				return aiAgentResultMsg{
					title:  "BIA Code Review",
					output: "No code provided",
					err:    fmt.Errorf("no code provided"),
				}
			}
		}

		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			result, err := ReviewCodeWithBIA(ctx, code)
			if err != nil {
				return aiAgentResultMsg{
					title:  "BIA Code Review",
					output: fmt.Sprintf("Error: %v", err),
					err:    err,
				}
			}

			return aiAgentResultMsg{
				title:  "BIA Code Review",
				output: result,
				err:    nil,
			}
		}
	}

	return nil
}

func (m *model) nextDeployStep() tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil {
		return nil
	}

	switch ws.Step {
	case 0:
		var loadCmd tea.Cmd
		if _, ok := ws.Data["accounts_loaded"]; !ok {
			if _, loading := ws.Data["accounts_loading"]; !loading {
				if _, attempted := ws.Data["accounts_attempted"]; attempted {
					loadCmd = nil
				} else {
					ws.Data["accounts_loading"] = true
					ws.Data["accounts_attempted"] = true
					ws.Data["accounts_error"] = ""
					loadCmd = tea.Batch(
						m.loadAzureAIAccountsCmd(),
						m.runCommand(CommandSpec{
							Command: azureAIAccountsTableCommand(),
							Mode:    CommandEmbedded,
						}),
					)
				}
			}
		}

		accounts, accountsLoaded := ws.Data["accounts"].([]AzureAIAccount)
		options := make([]huh.Option[int], 0)
		description, _ := ws.Data["accounts_error"].(string)
		hasError := description != ""
		if !accountsLoaded {
			if hasError {
				options = append(options, huh.NewOption("Unable to load accounts", -1))
			} else {
				options = append(options, huh.NewOption("Loading accounts...", -1))
				description = "Loading Azure AI accounts..."
			}
		} else {
			options = make([]huh.Option[int], len(accounts))
			for i, acc := range accounts {
				options[i] = huh.NewOption(
					fmt.Sprintf("%s (%s, %s)", acc.Name, acc.Kind, acc.Location),
					i,
				)
			}
		}

		m.palette.InputValue = ""
		var selectedIdx int
		select_ := huh.NewSelect[int]().
			Title("Select Azure AI account").
			Options(options...).
			Value(&selectedIdx)
		if description != "" {
			select_.Description(description)
		}

		m.palette.InputForm = huh.NewForm(huh.NewGroup(select_)).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(false).
			WithTheme(huh.ThemeCatppuccin())

		m.palette.State = PaletteStateCollectingParams

		ws.Data["selected_account_idx"] = &selectedIdx

		return tea.Batch(m.palette.InputForm.Init(), loadCmd)

	case 1:
		selectedIdx := *ws.Data["selected_account_idx"].(*int)
		accounts := ws.Data["accounts"].([]AzureAIAccount)
		selectedAccount := accounts[selectedIdx]
		ws.Data["selected_account"] = selectedAccount

		var loadCmd tea.Cmd
		if _, ok := ws.Data["deployments_loaded"]; !ok {
			if _, loading := ws.Data["deployments_loading"]; !loading {
				if _, attempted := ws.Data["deployments_attempted"]; attempted {
					loadCmd = nil
				} else {
					ws.Data["deployments_loading"] = true
					ws.Data["deployments_attempted"] = true
					ws.Data["deployments_error"] = ""
					loadCmd = tea.Batch(
						m.loadAzureAIDeploymentsCmd(selectedAccount),
						m.runCommand(CommandSpec{
							Command: azureAIDeploymentsTableCommand(selectedAccount.ResourceGroup, selectedAccount.Name),
							Mode:    CommandEmbedded,
						}),
					)
				}
			}
		}

		deployments, deploymentsLoaded := ws.Data["deployments"].([]AzureAIDeployment)
		options := make([]huh.Option[int], 0)
		description, _ := ws.Data["deployments_error"].(string)
		hasError := description != ""
		if !deploymentsLoaded {
			if hasError {
				options = append(options, huh.NewOption("Unable to load deployments", -1))
			} else {
				options = append(options, huh.NewOption("Loading deployments...", -1))
				description = "Loading model deployments..."
			}
		} else {
			options = make([]huh.Option[int], len(deployments))
			for i, dep := range deployments {
				label := dep.Name
				if dep.Model != "" {
					label += fmt.Sprintf(" (%s", dep.Model)
					if dep.Version != "" {
						label += fmt.Sprintf(" v%s", dep.Version)
					}
					label += ")"
				}
				options[i] = huh.NewOption(label, i)
			}
		}

		var selectedDepIdx int
		select_ := huh.NewSelect[int]().
			Title("Select model deployment").
			Options(options...).
			Value(&selectedDepIdx)
		if description != "" {
			select_.Description(description)
		}

		m.palette.InputForm = huh.NewForm(huh.NewGroup(select_)).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(false).
			WithTheme(huh.ThemeCatppuccin())

		m.palette.State = PaletteStateCollectingParams
		ws.Data["selected_deployment_idx"] = &selectedDepIdx

		return tea.Batch(m.palette.InputForm.Init(), loadCmd)

	case 2:
		selectedDepIdx := *ws.Data["selected_deployment_idx"].(*int)
		deployments := ws.Data["deployments"].([]AzureAIDeployment)
		ws.Data["selected_deployment"] = deployments[selectedDepIdx]

		var method string
		select_ := huh.NewSelect[string]().
			Title("Deployment method").
			Options(
				huh.NewOption("Azure Container Instance (run once)", "aci"),
				huh.NewOption("Azure Pipeline (CI/CD)", "pipeline"),
			).
			Value(&method)

		m.palette.InputForm = huh.NewForm(huh.NewGroup(select_)).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(false).
			WithTheme(huh.ThemeCatppuccin())

		m.palette.State = PaletteStateCollectingParams
		ws.Data["deploy_method"] = &method

		return m.palette.InputForm.Init()

	case 3:
		method := *ws.Data["deploy_method"].(*string)
		ws.Data["method_final"] = method

		m.palette.InputValue = ""
		input := huh.NewText().
			Title("Task for the agent").
			Placeholder("Review this PR and suggest improvements...").
			CharLimit(5000).
			Value(&m.palette.InputValue)

		m.palette.InputForm = huh.NewForm(huh.NewGroup(input)).
			WithWidth(80).
			WithHeight(10).
			WithShowHelp(false).
			WithShowErrors(false).
			WithTheme(huh.ThemeCatppuccin())

		m.palette.State = PaletteStateCollectingParams
		return m.palette.InputForm.Init()

	case 4:
		m.palette.State = PaletteStateExecuting
		m.palette.LoadingText = "Deploying agent..."
		m.palette.InputForm = nil

		promptStr := strings.TrimSpace(ws.Data["prompt"].(string))
		if promptStr == "" {
			return func() tea.Msg {
				return aiAgentResultMsg{
					title:  "Deploy Agent",
					output: "No task prompt provided",
					err:    fmt.Errorf("no task prompt provided"),
				}
			}
		}

		account := ws.Data["selected_account"].(AzureAIAccount)
		deployment := ws.Data["selected_deployment"].(AzureAIDeployment)
		method := ws.Data["method_final"].(string)

		return func() tea.Msg {
			agentType := AgentCustom
			modelLower := strings.ToLower(deployment.Model)
			if strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "openai") {
				agentType = AgentCursor
			} else if strings.Contains(modelLower, "claude") {
				agentType = AgentClaude
			}

			apiKey := getAzureAIKey(account.ResourceGroup, account.Name)
			if apiKey == "" {
				return aiAgentResultMsg{
					title:  "Deploy Agent",
					output: "Failed to retrieve Azure AI API key",
					err:    fmt.Errorf("failed to retrieve API key"),
				}
			}

			dconfig := DeployConfig{
				AgentType:     agentType,
				DeployMethod:  DeployMethod(method),
				AgentName:     fmt.Sprintf("agent-%d", time.Now().Unix()),
				ResourceGroup: account.ResourceGroup,
				Location:      account.Location,
				Prompt:        promptStr,
				AIAccount:     account.Name,
				AIEndpoint:    account.Endpoint,
				AIDeployment:  deployment.Name,
				AIModel:       deployment.Model,
			}

			var result string
			var err error

			switch DeployMethod(method) {
			case DeployACI:
				result, err = deployToACIFromPalette(dconfig, apiKey)
			case DeployPipeline:
				result, err = deployToPipelineFromPalette(dconfig, apiKey)
			default:
				err = fmt.Errorf("unknown deployment method: %s", method)
			}

			if err != nil {
				return aiAgentResultMsg{
					title:  "Deploy Agent",
					output: fmt.Sprintf("Deployment failed:\n\n%v", err),
					err:    err,
				}
			}

			return aiAgentResultMsg{
				title:  "Deploy Agent",
				output: fmt.Sprintf("✓ Deployment successful!\n\n%s", result),
				err:    nil,
			}
		}
	}

	return nil
}

type deployWizardAccountsMsg struct {
	accounts []AzureAIAccount
}

type deployWizardDeploymentsMsg struct {
	deployments []AzureAIDeployment
}

type deployWizardErrorMsg struct {
	step    int
	message string
}

func (m *model) loadAzureAIAccountsCmd() tea.Cmd {
	return func() tea.Msg {
		if !checkAzureCLI() {
			return deployWizardErrorMsg{
				step:    0,
				message: "Azure CLI is required but not installed. Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli",
			}
		}

		accounts := getAzureAIAccounts()
		if len(accounts) == 0 {
			return deployWizardErrorMsg{
				step:    0,
				message: "No Azure AI accounts found. Create one at: https://ai.azure.com",
			}
		}

		return deployWizardAccountsMsg{accounts: accounts}
	}
}

func (m *model) loadAzureAIDeploymentsCmd(account AzureAIAccount) tea.Cmd {
	return func() tea.Msg {
		if !checkAzureCLI() {
			return deployWizardErrorMsg{
				step:    1,
				message: "Azure CLI is required but not installed. Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli",
			}
		}

		deployments := getAzureAIDeployments(account.ResourceGroup, account.Name)
		if len(deployments) == 0 {
			return deployWizardErrorMsg{
				step:    1,
				message: fmt.Sprintf("No model deployments found in account '%s'. Deploy a model at: https://ai.azure.com", account.Name),
			}
		}

		return deployWizardDeploymentsMsg{deployments: deployments}
	}
}

func (m *model) handleWizardSubmit() tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil {
		return nil
	}

	value := m.palette.InputValue
	advanceStep := true

	switch ws.Type {
	case "bia":
		switch ws.Step {
		case 0:
			ws.Data["method"] = value
		case 1:
			if ws.Data["method"] == "file" {
				content, err := os.ReadFile(value)
				if err != nil {
					return func() tea.Msg {
						return staticOutputMsg{
							title:  "BIA Code Review",
							output: fmt.Sprintf("Failed to read file: %v", err),
						}
					}
				}
				ws.Data["code"] = string(content)
			} else {
				ws.Data["code"] = value
			}
		}
	case "deploy":
		switch ws.Step {
		case 0:
			if _, ok := ws.Data["accounts_loaded"]; !ok {
				advanceStep = false
				return m.nextDeployStep()
			}
		case 1:
			if _, ok := ws.Data["deployments_loaded"]; !ok {
				advanceStep = false
				return m.nextDeployStep()
			}
		case 3:
			ws.Data["prompt"] = value
		}
	}

	if advanceStep {
		ws.Step++
		return m.nextWizardStep()
	}
	return nil
}
