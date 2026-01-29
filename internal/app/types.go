package app

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// DashboardAction represents an action available in the Actions tab
type DashboardAction struct {
	ID          string
	Name        string
	Icon        string
	Description string
	Handler     func(m *model) tea.Cmd
}

// AddResourceWizard holds state for the Add Resource wizard
type AddResourceWizard struct {
	Step      int       // 0=name, 1=template, 2=confirm
	Name      string
	Template  string    // "blank", "commands", "detailed"
	InputForm *huh.Form
}

// PreferencesWizard holds state for the Preferences wizard
type PreferencesWizard struct {
	Step      int       // 0=menu, 1+=subsections
	Section   string    // "history", "mcp", "editor"
	InputForm *huh.Form
	// History settings
	HistoryEnabled      bool
	HistoryMaxItems     string // stored as string for form input
	HistoryDisplayCount string // stored as string for form input
	// MCP settings
	MCPEnabled bool
	MCPAction  string // "add", "remove", "edit"
	MCPName    string
	MCPURL     string
	// Editor setting
	Editor string
}

// ProvidersWizard holds state for the Configure Providers wizard
type ProvidersWizard struct {
	Step         int       // 0=menu, 1=provider form, 2=set default
	Action       string    // "add", "edit:name", "remove:name", "default"
	InputForm    *huh.Form
	// Provider fields
	ProviderType string // "openai", "anthropic", "ollama", "custom"
	Name         string
	APIKey       string
	BaseURL      string
	DefaultModel string
	Enabled      bool
}

// section represents a documentation section within a resource
type section struct {
	title   string
	content string
}

// resource represents a tool/documentation resource
type resource struct {
	name        string
	description string // First line of content
	content     string
	sections    []section
	embedded    bool // true if loaded from embedded FS (not user dir)
}

// command represents a parsed command from markdown
type command struct {
	lineNum     int
	raw         string
	cmd         string
	runnable    bool
	inputVar    string
	description string
}

// toolMeta contains metadata for enhanced card rendering
type toolMeta struct {
	icon        string
	asciiArt    string
	color       lipgloss.Color
	category    string
	status      string
	cmdCount    int
	lastUsed    string
	topCommands []string
}

// toolMetadata maps tool names to their metadata
var toolMetadata = map[string]toolMeta{
	"azure": {
		icon: "☁",
		asciiArt: `╭───╮
│ ▲ │
╰───╯`,
		color:       lipgloss.Color("39"),
		category:    "Cloud",
		status:      "active",
		cmdCount:    29,
		lastUsed:    "",
		topCommands: []string{"az login", "az group list", "az vm list"},
	},
	"claude": {
		icon: "◐",
		asciiArt: `╭───╮
│ ◐ │
╰───╯`,
		color:       lipgloss.Color("99"),
		category:    "AI",
		status:      "active",
		cmdCount:    24,
		lastUsed:    "2m ago",
		topCommands: []string{"claude chat", "claude --help", "claude config"},
	},
	"cursor": {
		icon: "▶",
		asciiArt: `╭───╮
│ ▶ │
╰───╯`,
		color:       lipgloss.Color("51"),
		category:    "AI Agent",
		status:      "active",
		cmdCount:    15,
		lastUsed:    "",
		topCommands: []string{"cursor", "cursor ls", "cursor mcp list"},
	},
	"docker": {
		icon: "▣",
		asciiArt: `╭───╮
│▣▣▣│
╰───╯`,
		color:       lipgloss.Color("39"),
		category:    "Containers",
		status:      "active",
		cmdCount:    42,
		lastUsed:    "5m ago",
		topCommands: []string{"docker ps", "docker compose up", "docker build"},
	},
	"git": {
		icon: "⎇",
		asciiArt: `╭───╮
│ ⎇ │
╰───╯`,
		color:       lipgloss.Color("208"),
		category:    "VCS",
		status:      "active",
		cmdCount:    38,
		lastUsed:    "1m ago",
		topCommands: []string{"git status", "git push", "git commit"},
	},
	"mcp": {
		icon: "◈",
		asciiArt: `╭───╮
│ ◈ │
╰───╯`,
		color:       lipgloss.Color("114"),
		category:    "Protocol",
		status:      "new",
		cmdCount:    12,
		lastUsed:    "1h ago",
		topCommands: []string{"mcp list", "mcp connect", "mcp inspect"},
	},
	"fast-agent": {
		icon: "⚡",
		asciiArt: `╭───╮
│ ⚡ │
╰───╯`,
		color:       lipgloss.Color("220"),
		category:    "AI Agent",
		status:      "active",
		cmdCount:    6,
		lastUsed:    "",
		topCommands: []string{"fast-agent run", "fast-agent chat", "fast-agent init"},
	},
}

// parseCommands parses commands from markdown content looking for ^run annotations
func parseCommands(content string) []command {
	var commands []command
	lines := strings.Split(content, "\n")

	cmdRe := regexp.MustCompile("`" + `([^` + "`" + `]+)` + "`" + `\s*([^^]*)\s*\^run(?::(\w+))?`)

	for i, line := range lines {
		matches := cmdRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		rawCmd := strings.TrimSpace(matches[1])
		desc := strings.TrimSpace(matches[2])
		inputVar := ""
		if len(matches) > 3 {
			inputVar = matches[3]
		}

		execCmd := rawCmd
		if inputVar != "" {
			varPattern := regexp.MustCompile(`\{\{` + inputVar + `\}\}`)
			execCmd = varPattern.ReplaceAllString(rawCmd, "{{INPUT}}")
		}

		commands = append(commands, command{
			lineNum:     i + 1,
			raw:         rawCmd,
			cmd:         execCmd,
			runnable:    true,
			inputVar:    inputVar,
			description: desc,
		})
	}

	return commands
}

// highlightShellCommand applies syntax highlighting to shell commands
func highlightShellCommand(cmd string) string {
	commandColor := lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true)
	subcommandColor := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	flagColor := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	valueColor := lipgloss.NewStyle().Foreground(lipgloss.Color("228"))
	variableColor := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)

	tokens := strings.Fields(cmd)
	if len(tokens) == 0 {
		return cmd
	}

	var highlighted []string
	firstToken := true
	expectingValue := false

	for _, token := range tokens {
		if strings.Contains(token, "{{") && strings.Contains(token, "}}") {
			highlighted = append(highlighted, variableColor.Render(token))
			expectingValue = false
			continue
		}

		if firstToken {
			highlighted = append(highlighted, commandColor.Render(token))
			firstToken = false
			continue
		}

		if strings.HasPrefix(token, "--") || (strings.HasPrefix(token, "-") && len(token) > 1 && token != "-") {
			highlighted = append(highlighted, flagColor.Render(token))
			expectingValue = true
			continue
		}

		if expectingValue && !strings.HasPrefix(token, "-") {
			highlighted = append(highlighted, valueColor.Render(token))
			expectingValue = false
			continue
		}

		highlighted = append(highlighted, subcommandColor.Render(token))
		expectingValue = false
	}

	return strings.Join(highlighted, " ")
}
