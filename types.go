package main

import (
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

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
}

// command represents a parsed command from markdown
// Commands are annotated with ^run or ^run:varname tags
// Example: `docker ps` list containers ^run
// Example: `claude "{{prompt}}"` start with prompt ^run:prompt
type command struct {
	lineNum     int    // Original line number in content
	raw         string // Raw command text (e.g., `claude "{{prompt}}"`)
	cmd         string // Executable command template
	runnable    bool   // Has ^run annotation
	inputVar    string // Variable name from ^run:varname (empty if no input needed)
	description string // Description text
}

type MCPServerStatus struct {
	Name                   string
	URL                    string
	Connected              bool
	Tools                  []string
	Prompts                []string
	Resources              []string
	ResourceTemplates      []string
	Error                  string
	ToolsError             string
	PromptsError           string
	ResourcesError         string
	ResourceTemplatesError string
	LastUpdated            time.Time
}

// toolMeta contains metadata for enhanced card rendering
type toolMeta struct {
	icon        string
	asciiArt    string // Small ASCII art for card
	color       lipgloss.Color
	category    string
	status      string // "active", "coming_soon", "new"
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
		color:       lipgloss.Color("39"),  // Blue
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
		color:       lipgloss.Color("99"),  // Purple
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
		color:       lipgloss.Color("51"),  // Cyan
		category:    "AI Editor",
		status:      "active",
		cmdCount:    28,
		lastUsed:    "",
		topCommands: []string{"⌘ L", "⌘ K", "⌘ I"},
	},
	"docker": {
		icon: "▣",
		asciiArt: `╭───╮
│▣▣▣│
╰───╯`,
		color:       lipgloss.Color("39"),  // Blue
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
		color:       lipgloss.Color("208"), // Orange
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
		color:       lipgloss.Color("114"), // Green
		category:    "Protocol",
		status:      "new",
		cmdCount:    12,
		lastUsed:    "1h ago",
		topCommands: []string{"mcp list", "mcp connect", "mcp inspect"},
	},
}

// parseCommands parses commands from markdown content looking for ^run annotations
// Syntax:
//   `command` description ^run           - executable command
//   `command {{var}}` description ^run:var - command with input prompt
func parseCommands(content string) []command {
	var commands []command
	lines := strings.Split(content, "\n")

	// Match: `command` description ^run or ^run:varname
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

		// Build the executable command - replace {{var}} with placeholder
		execCmd := rawCmd
		if inputVar != "" {
			// Replace {{varname}} with %s for later substitution
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
	// Define colors using lipgloss
	commandColor := lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true)   // green for main command
	subcommandColor := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))           // cyan for subcommands
	flagColor := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))                // pink for flags
	valueColor := lipgloss.NewStyle().Foreground(lipgloss.Color("228"))               // yellow for values
	variableColor := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true) // bright pink for {{variables}}

	// Split command into tokens
	tokens := strings.Fields(cmd)
	if len(tokens) == 0 {
		return cmd
	}

	var highlighted []string
	firstToken := true
	expectingValue := false

	for _, token := range tokens {
		// Check for {{variable}} patterns first
		if strings.Contains(token, "{{") && strings.Contains(token, "}}") {
			highlighted = append(highlighted, variableColor.Render(token))
			expectingValue = false
			continue
		}

		// First token is the main command (e.g., "az", "git", "docker")
		if firstToken {
			highlighted = append(highlighted, commandColor.Render(token))
			firstToken = false
			continue
		}

		// Flags (start with - or --)
		if strings.HasPrefix(token, "--") || (strings.HasPrefix(token, "-") && len(token) > 1 && token != "-") {
			highlighted = append(highlighted, flagColor.Render(token))
			expectingValue = true
			continue
		}

		// Values after flags
		if expectingValue && !strings.HasPrefix(token, "-") {
			highlighted = append(highlighted, valueColor.Render(token))
			expectingValue = false
			continue
		}

		// Subcommands (no prefix, not after a flag)
		highlighted = append(highlighted, subcommandColor.Render(token))
		expectingValue = false
	}

	return strings.Join(highlighted, " ")
}
