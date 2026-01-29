package app

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

func (m *model) loadResources() {
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

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "enter":
			m.currentView = viewDetail
			m.secCursor = 0
			m.initViewComponents()
			return m, nil

		case "up", "k":
			if m.resCursor > 0 {
				m.resCursor--
			}

		case "down", "j":
			if m.resCursor < len(m.resources)-1 {
				m.resCursor++
			}

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx < len(m.resources) {
				m.resCursor = idx
				m.currentView = viewDetail
				m.secCursor = 0
				m.initViewComponents()
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
