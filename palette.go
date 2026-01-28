package main

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
)

// PaletteItem represents an item in the command palette
type PaletteItem struct {
	ID          string
	Icon        string
	Title       string
	Subtitle    string
	Category    string // "action", "resource", "history", "favorite"
	Shortcut    string // Optional shortcut hint
	Handler     func(m *model) tea.Cmd
	ResourceIdx int // For resource items
	// MCP tool info (for tools that need parameters)
	MCPTool      *mcp.Tool
	MCPServer    string
	MCPServerURL string
}

// PaletteState represents the current state of the command palette
type PaletteState int

const (
	PaletteStateIdle            PaletteState = iota // Closed
	PaletteStateSearching                            // Open, showing command list
	PaletteStateCollectingParams                     // Collecting tool parameters with form
	PaletteStateAIInput                              // Collecting AI task description
	PaletteStateExecuting                            // Executing tool (showing spinner)
	PaletteStateShowingResult                        // Showing execution result
)

// Palette state
type Palette struct {
	State       PaletteState
	Query       string
	Items       []PaletteItem
	Filtered    []PaletteItem
	Cursor      int
	InputForm   *huh.Form
	InputValue  string // Bound to form input
	PendingTool *mcpPendingTool
	// Wizard state for multi-step actions
	WizardState *wizardState
	// Loading/execution state
	LoadingText string
	ResultTitle string
	ResultText  string
}

// mcpPendingTool holds state for an MCP tool waiting for parameter input
type mcpPendingTool struct {
	ServerName string
	ServerURL  string
	Tool       mcp.Tool
	Args       map[string]any
	// Form field values (string pointers for huh binding)
	FormValues map[string]*string
	// Natural language task description (for AI mode)
	AITask string
}

// wizardState holds state for multi-step wizards (BIA, Deploy, etc.)
type wizardState struct {
	Type    string         // "bia" or "deploy"
	Step    int            // Current step number
	Data    map[string]any // Collected data
	Options []string       // For select options
}

// buildPaletteItems creates all available palette items
func (m *model) buildPaletteItems() []PaletteItem {
	var items []PaletteItem

	// BIA Code Review action
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

	// Deploy Agent action
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

	// Create Script action
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

	// Dynamic MCP tools
	items = append(items, m.getMCPToolItems()...)

	return items
}

// getMCPToolItems fetches tools from all configured MCP servers and returns palette items
func (m *model) getMCPToolItems() []PaletteItem {
	var items []PaletteItem
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for _, server := range m.config.MCP.Servers {
		tools, err := FetchMCPTools(ctx, server.URL)
		if err != nil {
			continue
		}
		for _, tool := range tools {
			items = append(items, m.mcpToolToPaletteItem(server.Name, server.URL, tool))
		}
	}
	return items
}

// mcpToolToPaletteItem converts an MCP tool to a PaletteItem
func (m *model) mcpToolToPaletteItem(serverName string, serverURL string, tool mcp.Tool) PaletteItem {
	// Store tool info for parameter handling
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
		// Handler is nil - we handle MCP tools specially to support parameter input
	}
}

// executeMCPToolWithArgs executes an MCP tool with pre-collected args and displays output
func executeMCPToolWithArgs(serverURL string, toolName string, args map[string]any) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// Connect to MCP server
		client, err := NewMCPClient(serverURL)
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

		// Call the tool
		result, err := client.CallTool(ctx, toolName, args)
		if err != nil {
			return staticOutputMsg{
				title:  toolName,
				output: fmt.Sprintf("Error: %v", err),
			}
		}

		// Extract text from result
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

// startMCPToolWithAI invokes an MCP tool using an AI agent
func (m *model) startMCPToolWithAI(item PaletteItem) tea.Cmd {
	tool := item.MCPTool
	if tool == nil {
		return nil
	}

	// Store the tool for later execution
	m.palette.PendingTool = &mcpPendingTool{
		ServerName: item.MCPServer,
		ServerURL:  item.MCPServerURL,
		Tool:       *tool,
		Args:       make(map[string]any),
		FormValues: make(map[string]*string),
	}

	// Transition to AI input state
	m.palette.State = PaletteStateAIInput
	m.palette.Query = "" // Clear any existing search

	return nil
}

// startMCPToolInput initializes input mode for an MCP tool that needs parameters
func (m *model) startMCPToolInput(item PaletteItem) tea.Cmd {
	tool := item.MCPTool
	if tool == nil {
		return nil
	}

	if len(tool.InputSchema.Properties) == 0 {
		// No parameters needed, execute directly
		return executeMCPToolWithArgs(item.MCPServerURL, tool.Name, nil)
	}

	// Initialize pending tool state with form values
	formValues := make(map[string]*string)

	// Create string pointers for each parameter (for huh binding)
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

	// Build comprehensive form with all parameters
	return m.buildParameterForm()
}

// buildParameterForm creates a comprehensive form with all parameters
// buildParameterFormWithValues builds the parameter form with AI-prefilled values
func (m *model) buildParameterFormWithValues(aiParams map[string]interface{}) tea.Cmd {
	pt := m.palette.PendingTool
	if pt == nil {
		return nil
	}

	// First initialize all parameters with empty strings
	for paramName := range pt.Tool.InputSchema.Properties {
		if pt.FormValues[paramName] == nil {
			val := ""
			pt.FormValues[paramName] = &val
		}
	}

	// Then fill in AI-determined values
	for paramName, paramValue := range aiParams {
		// Convert to string
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
			// For complex types, use JSON encoding
			bytes, _ := json.Marshal(v)
			strValue = string(bytes)
		}

		// Update the value if this parameter exists
		if ptr := pt.FormValues[paramName]; ptr != nil {
			*ptr = strValue
		}
	}

	// Now build the form (which will use the pre-filled values)
	return m.buildParameterForm()
}

func (m *model) buildParameterForm() tea.Cmd {
	pt := m.palette.PendingTool
	if pt == nil {
		return nil
	}

	// Build required map for quick lookup
	required := make(map[string]bool)
	for _, r := range pt.Tool.InputSchema.Required {
		required[r] = true
	}

	// Collect field definitions in stable order: required first (alphabetically), then optional (alphabetically)
	var fields []huh.Field
	var paramNames []string

	// Get all parameter names
	for paramName := range pt.Tool.InputSchema.Properties {
		paramNames = append(paramNames, paramName)
	}

	// Sort: required params first (alphabetically), then optional params (alphabetically)
	sort.Slice(paramNames, func(i, j int) bool {
		reqI := required[paramNames[i]]
		reqJ := required[paramNames[j]]
		if reqI != reqJ {
			return reqI // Required comes first
		}
		return paramNames[i] < paramNames[j] // Alphabetical within each group
	})

	// Build form fields based on JSON schema
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

	// Create form with all fields in a single group
	m.palette.InputForm = huh.NewForm(huh.NewGroup(fields...)).
		WithWidth(100).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(huh.ThemeCatppuccin())

	m.palette.State = PaletteStateCollectingParams

	// Initialize the form
	return m.palette.InputForm.Init()
}

// createFormField creates the appropriate huh field based on JSON schema type
func (m *model) createFormField(paramName string, paramMap map[string]interface{}, isRequired bool) huh.Field {
	pt := m.palette.PendingTool
	if pt == nil {
		return nil
	}

	// Extract schema properties
	description := ""
	if desc, ok := paramMap["description"].(string); ok {
		description = desc
	}

	paramType := "string"
	if t, ok := paramMap["type"].(string); ok {
		paramType = t
	}

	// Build title with required indicator
	title := paramName
	if isRequired {
		title = paramName + " *"
	}

	// Get value pointer for binding
	valuePtr := pt.FormValues[paramName]

	// Handle enum fields (select dropdown)
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

	// Handle boolean fields (confirm)
	if paramType == "boolean" {
		boolPtr := new(bool)
		return huh.NewConfirm().
			Title(title).
			Description(description).
			Value(boolPtr)
	}

	// Handle text area for long strings
	if paramType == "string" {
		// Check if this is likely a long text field
		maxLength := 0
		if ml, ok := paramMap["maxLength"].(float64); ok {
			maxLength = int(ml)
		}

		// Get examples/placeholder
		placeholder := description
		if examples, ok := paramMap["examples"].([]interface{}); ok && len(examples) > 0 {
			placeholder = fmt.Sprintf("%v", examples[0])
		}

		// Use text area for long content or if description suggests it
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

		// Regular input with validation
		input := huh.NewInput().
			Title(title).
			Description(description).
			Placeholder(placeholder).
			Value(valuePtr)

		// Add validation for required fields
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

	// Handle number fields
	if paramType == "number" || paramType == "integer" {
		input := huh.NewInput().
			Title(title).
			Description(description).
			Placeholder("Enter a number").
			Value(valuePtr)

		// Add numeric validation
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

	// Default: string input
	return huh.NewInput().
		Title(title).
		Description(description).
		Value(valuePtr)
}

// handleParameterSubmit processes the submitted parameter values
func (m *model) handleParameterSubmit() tea.Cmd {
	pt := m.palette.PendingTool
	if pt == nil {
		return m.showNotification("⚠️", "No pending tool found", "error")
	}

	// Parse and validate all parameter values
	required := make(map[string]bool)
	for _, r := range pt.Tool.InputSchema.Required {
		required[r] = true
	}

	// Extract values from form and convert to proper types
	for paramName, valuePtr := range pt.FormValues {
		if valuePtr == nil {
			continue
		}

		value := strings.TrimSpace(*valuePtr)

		// Skip empty optional parameters
		if value == "" && !required[paramName] {
			continue
		}

		// Get parameter schema
		paramDef := pt.Tool.InputSchema.Properties[paramName]
		paramMap, ok := paramDef.(map[string]interface{})
		if !ok {
			continue
		}

		// Convert value based on type
		paramType := "string"
		if t, ok := paramMap["type"].(string); ok {
			paramType = t
		}

		switch paramType {
		case "boolean":
			// Parse boolean
			boolVal := value == "true" || value == "yes" || value == "1"
			pt.Args[paramName] = boolVal

		case "number":
			// Parse float
			var floatVal float64
			if _, err := fmt.Sscanf(value, "%f", &floatVal); err == nil {
				pt.Args[paramName] = floatVal
			} else if value != "" {
				return m.showNotification("⚠️", fmt.Sprintf("Invalid number for %s", paramName), "warning")
			}

		case "integer":
			// Parse int
			var intVal int
			if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
				pt.Args[paramName] = intVal
			} else if value != "" {
				return m.showNotification("⚠️", fmt.Sprintf("Invalid integer for %s", paramName), "warning")
			}

		case "array", "object":
			// Parse JSON for complex types
			if value != "" {
				var jsonValue interface{}
				if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
					pt.Args[paramName] = jsonValue
				} else {
					return m.showNotification("⚠️", fmt.Sprintf("Invalid JSON for %s: %v", paramName, err), "warning")
				}
			}

		default:
			// String (default)
			if value != "" {
				pt.Args[paramName] = value
			}
		}
	}

	// Execute the tool directly without preview/confirmation
	m.palette.InputForm = nil
	serverURL := pt.ServerURL
	toolName := pt.Tool.Name
	args := pt.Args
	m.palette.PendingTool = nil

	// Transition to executing state
	m.palette.State = PaletteStateExecuting
	m.palette.LoadingText = "Executing tool..."

	return executeMCPToolWithArgs(serverURL, toolName, args)
}

// filterPaletteItems filters items based on query using simple substring matching
func filterPaletteItems(items []PaletteItem, query string) []PaletteItem {
	if query == "" {
		return items
	}

	query = strings.ToLower(query)
	var filtered []PaletteItem

	for _, item := range items {
		// Match against title, subtitle, or category
		if strings.Contains(strings.ToLower(item.Title), query) ||
			strings.Contains(strings.ToLower(item.Subtitle), query) ||
			strings.Contains(strings.ToLower(item.Category), query) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// openPalette opens the command palette
func (m *model) openPalette() {
	m.palette.State = PaletteStateSearching
	m.palette.Query = ""
	m.palette.Items = m.buildPaletteItems()
	m.palette.Filtered = m.palette.Items
	m.palette.Cursor = 0
}

// closePalette closes the command palette
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

// truncate truncates a string to max length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// renderPalette renders the command palette with modern split-view layout
func (m model) renderPalette() string {
	// Make palette much larger - 85% of screen width, 80% of height
	paletteWidth := int(float64(m.width) * 0.85)
	paletteHeight := int(float64(m.height) * 0.80)

	if paletteWidth < 100 {
		paletteWidth = 100
	}
	if paletteHeight < 30 {
		paletteHeight = 30
	}

	accentColor := lipgloss.Color("99") // Purple accent

	var lines []string

	// Handle different palette states
	switch m.palette.State {
	case PaletteStateCollectingParams:
		// Show parameter/wizard input form
		if m.palette.InputForm == nil {
			return lipgloss.NewStyle().Render("Error: No form available")
		}
		// Header styles
		textStyle := lipgloss.NewStyle().Foreground(subtle)
		keyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238")).
			Padding(0, 1)

		var headerContent string

		// Check if this is a wizard or MCP parameter collection
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

			// Parameter input mode
			// Count required vs optional
			required := make(map[string]bool)
			for _, r := range pt.Tool.InputSchema.Required {
				required[r] = true
			}
			reqCount := len(pt.Tool.InputSchema.Required)
			totalCount := len(pt.Tool.InputSchema.Properties)
			optCount := totalCount - reqCount

			// Check if AI pre-filled the values
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

			// Tool icon based on server name
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

		// Add tool description subtitle if available (for MCP tools)
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

		// Render the huh form
		formView := m.palette.InputForm.View()
		lines = append(lines, formView)

		// If there's terminal output, show it below the form
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
		// Show execution result
		return m.renderPaletteResult(paletteWidth, paletteHeight, accentColor)

	default:
		// Normal command list mode with split view (searching, AI input, executing)
		return m.renderPaletteSplitView(paletteWidth, paletteHeight, accentColor)
	}
}

// renderPaletteResult renders the result view after tool execution
func (m model) renderPaletteResult(paletteWidth, paletteHeight int, accentColor lipgloss.Color) string {
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("234")).
		Foreground(accentColor).
		Bold(true).
		Width(paletteWidth - 4).
		Padding(0, 1)

	lines = append(lines, headerStyle.Render("✓ "+m.palette.ResultTitle))
	lines = append(lines, "")

	// Render result with glamour
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

	// Footer hint
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

// renderPaletteSplitView renders the command palette with split view (list + preview)
func (m model) renderPaletteSplitView(paletteWidth, paletteHeight int, accentColor lipgloss.Color) string {
	// Split: 45% list, 55% preview
	listWidth := int(float64(paletteWidth) * 0.45)
	previewWidth := paletteWidth - listWidth - 4 // Account for borders and spacing

	// Render left side (list)
	leftPanel := m.renderPaletteList(listWidth, paletteHeight, accentColor)

	// Render right side (preview)
	rightPanel := m.renderPalettePreview(previewWidth, paletteHeight, accentColor)

	// Join panels horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		rightPanel,
	)

	// Container with border
	container := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1)

	return container.Render(content)
}

// renderPaletteList renders the command list (left panel)
func (m model) renderPaletteList(width, height int, accentColor lipgloss.Color) string {
	var lines []string

	// Header with count and keyboard hints
	countStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(subtle)
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("238")).
		Padding(0, 1)

	var infoContent string
	switch m.palette.State {
	case PaletteStateExecuting:
		// Loading state header
		toolName := "AI Agent"
		if m.palette.PendingTool != nil {
			toolName = m.palette.PendingTool.Tool.Name
		}
		infoContent = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("🤖 " + toolName) +
			textStyle.Render("  Executing...")

	case PaletteStateAIInput:
		// AI Agent mode header
		toolName := "AI Agent"
		if m.palette.PendingTool != nil {
			toolName = m.palette.PendingTool.Tool.Name
		}
		infoContent = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("🤖 " + toolName) +
			textStyle.Render("  ") +
			keyStyle.Render("enter") + textStyle.Render(" execute  ") +
			keyStyle.Render("esc") + textStyle.Render(" cancel")

	default:
		// Normal mode header
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

	// Search input
	var queryDisplay string
	var searchLine string

	switch m.palette.State {
	case PaletteStateExecuting:
		// Loading state - show what was submitted
		queryDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(m.palette.Query)
		searchLine = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("🤖 ") + queryDisplay

	case PaletteStateAIInput:
		// AI Agent input mode
		if m.palette.Query == "" {
			queryDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Italic(true).Render("🤖 Describe what you want the AI to do...")
		} else {
			queryDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Render(m.palette.Query) +
				lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("▌")
		}
		searchLine = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("🤖 ") + queryDisplay

	default:
		// Normal search mode
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

	// Divider
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render(strings.Repeat("─", width-2))
	lines = append(lines, divider)

	// Command list with grouping (or AI tool info in AI mode)
	maxVisibleItems := height - 8 // Leave room for header/search/divider

	switch m.palette.State {
	case PaletteStateExecuting:
		// Show loading indicator
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true).
			Padding(2, 1).
			Width(width - 4).
			Align(lipgloss.Center)

		spinner := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
		frame := spinner[0:1] // TODO: animate this
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
		// Show AI agent instructions instead of command list
		if m.palette.PendingTool != nil {
			tool := m.palette.PendingTool.Tool

			// Tool description
			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(1, 1).
				Width(width - 4)
			lines = append(lines, descStyle.Render(tool.Description))

			// Instructions
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
		// Searching mode - show command list
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
		// Group items by category
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

		// Render grouped items
		currentIndex := 0
		for _, category := range categories {
			catItems := grouped[category]

			// Category header
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

			// Render items in this category
			for _, item := range catItems {
				if len(lines) >= maxVisibleItems+3 {
					break
				}

				isSelected := currentIndex == m.palette.Cursor

				// Item title (truncate if needed)
				title := item.Title
				maxTitleLen := width - 10
				if len(title) > maxTitleLen {
					title = title[:maxTitleLen-3] + "..."
				}

				// Item icon
				icon := item.Icon
				if icon == "" {
					icon = "•"
				}

				if isSelected {
					// Selected item
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
					// Normal item
					itemLine := lipgloss.NewStyle().
						Foreground(lipgloss.Color("252")).
						Padding(0, 1).
						Render(fmt.Sprintf(" %s %s", icon, title))
					lines = append(lines, "    "+itemLine)
				}

				currentIndex++
			}

			// Add spacing between categories
			if category != categories[len(categories)-1] {
				lines = append(lines, "")
			}
		}

		// "More items" indicator
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

	// Panel container
	panel := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("238"))

	return panel.Render(content)
}

// renderPalettePreview renders the preview panel (right side)
func (m model) renderPalettePreview(width, height int, accentColor lipgloss.Color) string {
	var lines []string

	// If executing, show progress info
	if m.palette.State == PaletteStateExecuting && m.palette.PendingTool != nil {
		tool := m.palette.PendingTool.Tool

		// Title
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true).
			Padding(1, 1, 0, 1)
		lines = append(lines, titleStyle.Render(fmt.Sprintf("🤖 Executing %s", tool.Name)))

		// Divider
		divider := lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Padding(0, 1).
			Render(strings.Repeat("─", width-2))
		lines = append(lines, divider)

		// Steps
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

	// If in AI input mode, show parameter details
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

	// Get selected item
	var selectedItem *PaletteItem
	if len(m.palette.Filtered) > 0 && m.palette.Cursor < len(m.palette.Filtered) {
		selectedItem = &m.palette.Filtered[m.palette.Cursor]
	}

	if selectedItem == nil {
		// No selection
		emptyStyle := lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true).
			Padding(height/2, 2).
			Width(width).
			Align(lipgloss.Center)
		return emptyStyle.Render("Select a command to see details")
	}

	// Header with icon and title
	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Padding(1, 1, 0, 1)

	icon := selectedItem.Icon
	if icon == "" {
		icon = "⚡"
	}

	lines = append(lines, titleStyle.Render(fmt.Sprintf("%s %s", icon, selectedItem.Title)))

	// Category badge
	if selectedItem.Category != "" {
		badgeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			MarginLeft(1).
			MarginBottom(1)
		lines = append(lines, badgeStyle.Render(strings.ToUpper(selectedItem.Category)))
	}

	// Divider
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Padding(0, 1).
		Render(strings.Repeat("─", width-2))
	lines = append(lines, divider)

	// Render specific content based on item type
	if selectedItem.MCPTool != nil {
		// MCP Tool preview
		lines = append(lines, m.renderMCPToolPreview(selectedItem.MCPTool, width)...)
	} else {
		// Regular action/command preview
		// Description
		if selectedItem.Subtitle != "" {
			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(1, 1).
				Width(width - 2)
			lines = append(lines, descStyle.Render(selectedItem.Subtitle))
		}

		// Shortcut if available
		if selectedItem.Shortcut != "" {
			shortcutStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("114")).
				Padding(1, 1)
			lines = append(lines, shortcutStyle.Render(fmt.Sprintf("Shortcut: %s", selectedItem.Shortcut)))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Panel container
	panel := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1)

	return panel.Render(content)
}

// renderMCPToolPreview renders detailed preview for an MCP tool
func (m model) renderMCPToolPreview(tool *mcp.Tool, width int) []string {
	var lines []string

	// Description
	if tool.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1, 1, 1).
			Width(width - 2)
		lines = append(lines, descStyle.Render(tool.Description))
	}

	// Parameters section
	if len(tool.InputSchema.Properties) > 0 {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true).
			Padding(1, 1, 0, 1)
		lines = append(lines, headerStyle.Render("Parameters:"))

		// Build required map
		required := make(map[string]bool)
		for _, r := range tool.InputSchema.Required {
			required[r] = true
		}

		// Sort parameter names for stable rendering (required first, then alphabetical)
		var paramNames []string
		for paramName := range tool.InputSchema.Properties {
			paramNames = append(paramNames, paramName)
		}

		// Sort: required params first (alphabetically), then optional params (alphabetically)
		sort.Slice(paramNames, func(i, j int) bool {
			reqI := required[paramNames[i]]
			reqJ := required[paramNames[j]]
			if reqI != reqJ {
				return reqI // Required comes first
			}
			return paramNames[i] < paramNames[j] // Alphabetical within each group
		})

		// List parameters in sorted order
		paramCount := 0
		maxParams := 8 // Limit display
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

			// Parameter name with required indicator
			paramLabel := paramName
			if required[paramName] {
				paramLabel = paramName + " *"
			}

			// Parameter type
			paramType := "string"
			if t, ok := paramMap["type"].(string); ok {
				paramType = t
			}

			// Build parameter line
			paramStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")).
				Padding(0, 1)
			typeStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Italic(true)

			lines = append(lines, paramStyle.Render(fmt.Sprintf("  • %s", paramLabel))+" "+typeStyle.Render(fmt.Sprintf("(%s)", paramType)))

			// Parameter description (if available)
			if desc, ok := paramMap["description"].(string); ok && desc != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("246")).
					Padding(0, 1).
					MarginLeft(4).
					Width(width - 8)
				// Truncate long descriptions
				if len(desc) > 80 {
					desc = desc[:77] + "..."
				}
				lines = append(lines, descStyle.Render(desc))
			}

			paramCount++
		}

		// Summary
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

		// AI agent hint
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

// executeMCPToolWithAIAgent executes an MCP tool using mods AI to determine parameters
// aiProgressMsg is sent during AI agent execution to update progress
type aiProgressMsg struct {
	step string
}

// aiAgentResultMsg carries the result from AI agent execution
type aiAgentResultMsg struct {
	title  string
	output string
	err    error
}

// aiPrefilledParamsMsg carries AI-determined parameters for user review
type aiPrefilledParamsMsg struct {
	params map[string]interface{}
}

func (m *model) executeMCPToolWithAIAgent(pt *mcpPendingTool) tea.Cmd {
	return func() tea.Msg {
		// Small delay to ensure loading state renders
		time.Sleep(100 * time.Millisecond)

		// Check if OpenAI API key is configured
		apiKey := m.config.AI.OpenAIAPIKey
		if apiKey == "" {
			// Fall back to environment variable
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

		// Build prompt for AI
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

		// Call OpenAI API directly
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

		// Parse JSON response
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(result), &params); err != nil {
			// Show the AI response as formatted markdown
			return aiAgentResultMsg{
				title:  "🤖 " + pt.Tool.Name,
				output: "**The AI response couldn't be parsed as JSON:**\n\n" + result + "\n\n---\n\n*Try entering parameters manually by pressing Enter on the tool.*",
				err:    err,
			}
		}

		// Return the AI-determined parameters for user review
		// The user can edit and then submit the form
		return aiPrefilledParamsMsg{
			params: params,
		}
	}
}

// formatToolSchema formats the tool's parameter schema for the AI prompt
func formatToolSchema(tool mcp.Tool) string {
	var schema strings.Builder

	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}

	// Sort parameters
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

// formatJSON formats a map as pretty JSON
func formatJSON(data map[string]interface{}) string {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(bytes)
}

// startBIAWizard starts the BIA Code Review wizard
func (m *model) startBIAWizard() tea.Cmd {
	m.palette.WizardState = &wizardState{
		Type: "bia",
		Step: 0,
		Data: make(map[string]any),
	}
	return m.nextWizardStep()
}

// startDeployWizard starts the Deploy Agent wizard
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

// nextWizardStep advances to the next step in the wizard
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

// nextBIAStep handles BIA wizard steps
func (m *model) nextBIAStep() tea.Cmd {
	ws := m.palette.WizardState

	switch ws.Step {
	case 0: // Ask for input method
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

	case 1: // Get code (paste or file path)
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

	case 2: // Execute BIA review
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

		// Run BIA review async
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

// nextDeployStep handles Deploy wizard steps
func (m *model) nextDeployStep() tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil {
		return nil
	}

	switch ws.Step {
	case 0: // Load Azure AI accounts and select one
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

		// Store selectedIdx temporarily for the next step
		ws.Data["selected_account_idx"] = &selectedIdx

		return tea.Batch(m.palette.InputForm.Init(), loadCmd)

	case 1: // Load deployments for selected account
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

	case 2: // Select deployment method
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

	case 3: // Enter task prompt
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

	case 4: // Execute deployment
		m.palette.State = PaletteStateExecuting
		m.palette.LoadingText = "Deploying agent..."
		m.palette.InputForm = nil

		prompt := strings.TrimSpace(ws.Data["prompt"].(string))
		if prompt == "" {
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

		// Execute deployment asynchronously
		return func() tea.Msg {
			// Determine agent type from model
			agentType := AgentCustom
			modelLower := strings.ToLower(deployment.Model)
			if strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "openai") {
				agentType = AgentCursor
			} else if strings.Contains(modelLower, "claude") {
				agentType = AgentClaude
			}

			// Get API key
			apiKey := getAzureAIKey(account.ResourceGroup, account.Name)
			if apiKey == "" {
				return aiAgentResultMsg{
					title:  "Deploy Agent",
					output: "Failed to retrieve Azure AI API key",
					err:    fmt.Errorf("failed to retrieve API key"),
				}
			}

			config := DeployConfig{
				AgentType:     agentType,
				DeployMethod:  DeployMethod(method),
				AgentName:     fmt.Sprintf("agent-%d", time.Now().Unix()),
				ResourceGroup: account.ResourceGroup,
				Location:      account.Location,
				Prompt:        prompt,
				AIAccount:     account.Name,
				AIEndpoint:    account.Endpoint,
				AIDeployment:  deployment.Name,
				AIModel:       deployment.Model,
			}

			var result string
			var err error

			switch DeployMethod(method) {
			case DeployACI:
				result, err = deployToACIFromPalette(config, apiKey)
			case DeployPipeline:
				result, err = deployToPipelineFromPalette(config, apiKey)
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

// deployWizardAccountsMsg carries loaded Azure AI accounts
type deployWizardAccountsMsg struct {
	accounts []AzureAIAccount
}

// deployWizardDeploymentsMsg carries loaded model deployments
type deployWizardDeploymentsMsg struct {
	deployments []AzureAIDeployment
}

// deployWizardErrorMsg carries wizard errors without leaving the form flow
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

// handleWizardSubmit processes wizard form submission
func (m *model) handleWizardSubmit() tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil {
		return nil
	}

	value := m.palette.InputValue
	advanceStep := true

	// Store the value based on current step
	switch ws.Type {
	case "bia":
		switch ws.Step {
		case 0: // Input method selected
			ws.Data["method"] = value
		case 1: // Code provided
			if ws.Data["method"] == "file" {
				// Read file
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
		case 3: // Task prompt entered
			ws.Data["prompt"] = value
		}
	}

	// Move to next step
	if advanceStep {
		ws.Step++
		return m.nextWizardStep()
	}
	return nil
}
