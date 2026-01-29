package app

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ActiveAgent represents a currently running agent
type ActiveAgent struct {
	ID        string
	Name      string
	Provider  string
	Runtime   string    // "docker", "e2b"
	StartTime time.Time
	Status    string    // "running", "completed", "failed"
	Task      string    // The prompt/task
}

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
	Step         int       // 0=menu, 1=type select, 2=details form, 3=test, 4=set default
	Action       string    // "add", "edit:name", "remove:name", "default"
	InputForm    *huh.Form
	// Provider fields
	ProviderType string // "openai", "anthropic", "ollama", "openai-compatible"
	Name         string
	APIKey       string
	BaseURL      string
	DefaultModel string
	Enabled      bool
	// Test connection state
	Testing    bool
	TestResult string
	TestError  string
}

// DeleteResourceWizard holds state for delete confirmation
type DeleteResourceWizard struct {
	ResourceName string
	IsEmbedded   bool
	Confirmed    bool
	InputForm    *huh.Form
}

// RunAgentWizard holds state for the Run Agent wizard
type RunAgentWizard struct {
	Step      int       // 0=provider, 1=runtime, 2=config, 3=confirm
	Provider  string    // provider name from config
	Runtime   string    // "docker" or "e2b"
	AgentName string
	Task      string
	Image     string
	Confirmed bool
	InputForm *huh.Form
}

// SavedAgentWizard holds state for running a saved agent
type SavedAgentWizard struct {
	Step      int    // 0=provider, 1=resource, 2=prompt, 3=confirm
	AgentID   string // ID of the saved agent
	AgentName string // Display name
	Image     string // Docker image
	BuildPath string // Path to build if needed
	Provider  string // Selected provider name
	Resource  string // Selected resource name
	Prompt    string
	Confirmed bool
	InputForm *huh.Form
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
	"e2b": {
		icon: "◇",
		asciiArt: `╭───╮
│ ◇ │
╰───╯`,
		color:       lipgloss.Color("43"),
		category:    "Sandbox",
		status:      "active",
		cmdCount:    9,
		lastUsed:    "",
		topCommands: []string{"e2b auth login", "e2b sandbox list", "e2b template list"},
	},
	"codex": {
		icon: "◎",
		asciiArt: `╭───╮
│ ◎ │
╰───╯`,
		color:       lipgloss.Color("79"),
		category:    "AI Agent",
		status:      "active",
		cmdCount:    32,
		lastUsed:    "",
		topCommands: []string{"codex", "codex --help", "codex -a full-auto"},
	},
	"gcp": {
		icon: "◈",
		asciiArt: `╭───╮
│ ◈ │
╰───╯`,
		color:       lipgloss.Color("33"),
		category:    "Cloud",
		status:      "active",
		cmdCount:    15,
		lastUsed:    "",
		topCommands: []string{"gcloud auth login", "gcloud projects list", "gcloud ai models list"},
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

// CardItem represents a single card in a CardGrid
type CardItem struct {
	Title       string
	Subtitle    string
	Tag         string
	TagColor    lipgloss.Color
	BorderColor lipgloss.Color
	Shortcut    int // 1-based index for [N] display
}

// CardGrid renders a responsive grid of cards
func CardGrid(items []CardItem, width int, selectedIdx int) string {
	if len(items) == 0 {
		return ""
	}

	// Calculate card width - more compact layout
	cardW := (width - 3) / 4 // Try 4 cards per row first
	if cardW < 22 {
		cardW = (width - 2) / 3 // Fall back to 3 cards
	}
	if cardW < 22 {
		cardW = (width - 2) / 2 // Fall back to 2 cards
	}
	if cardW < 22 {
		cardW = width - 2
	}

	var cards []string
	for i, item := range items {
		isSelected := i == selectedIdx

		shortcut := lipgloss.NewStyle().
			Foreground(subtle).
			Render(fmt.Sprintf("[%d]", item.Shortcut))

		titleColor := item.TagColor
		if titleColor == "" {
			titleColor = white
		}
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(titleColor)
		descStyle := lipgloss.NewStyle().Foreground(subtle)

		// Truncate subtitle if too long
		subtitle := item.Subtitle
		maxSubLen := cardW - 6
		if len(subtitle) > maxSubLen && maxSubLen > 3 {
			subtitle = subtitle[:maxSubLen-3] + "..."
		}

		var cardContent string
		if item.Tag != "" {
			tagStyle := lipgloss.NewStyle().
				Foreground(item.TagColor).
				Background(lipgloss.Color("236")).
				Padding(0, 1)
			cardContent = lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.Render(item.Title)+"  "+shortcut,
				descStyle.Render(subtitle),
				tagStyle.Render(item.Tag),
			)
		} else {
			cardContent = lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.Render(item.Title)+"  "+shortcut,
				descStyle.Render(subtitle),
			)
		}

		borderColor := item.BorderColor
		if borderColor == "" {
			borderColor = dimBorder
		}
		if isSelected {
			if item.TagColor != "" {
				borderColor = item.TagColor
			} else {
				borderColor = primary
			}
		}

		cardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1).
			Width(cardW - 2)

		cards = append(cards, cardStyle.Render(cardContent))
	}

	// Arrange in rows
	cardsPerRow := width / cardW
	if cardsPerRow < 1 {
		cardsPerRow = 1
	}

	var rows []string
	for i := 0; i < len(cards); i += cardsPerRow {
		end := i + cardsPerRow
		if end > len(cards) {
			end = len(cards)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards[i:end]...)
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// highlightShellCommand applies syntax highlighting to shell commands
func highlightShellCommand(cmd string) string {
	commandColor := lipgloss.NewStyle().Foreground(lipgloss.Color("73")).Bold(true)
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
