package app

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aaronjanse/3mux/ecma48"
	"github.com/aaronjanse/3mux/vterm"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"

	"github.com/htelsiz/skitz/internal/config"
	mcppkg "github.com/htelsiz/skitz/internal/mcp"
)

type model struct {
	resources     []resource
	resCursor     int
	secCursor     int
	width, height int

	// View state
	currentView int // viewDashboard or viewDetail

	// Dashboard tabs
	dashboardTab          int                   // 0=Resources, 1=Actions
	actionItems           []DashboardAction     // Available actions
	actionCursor          int                   // Selected action
	addResourceWizard     *AddResourceWizard    // Add Resource wizard state
	preferencesWizard     *PreferencesWizard    // Preferences wizard state
	providersWizard       *ProvidersWizard      // Configure Providers wizard state
	deleteResourceWizard  *DeleteResourceWizard // Delete Resource confirmation state
	runAgentWizard        *RunAgentWizard       // Run Agent wizard state
	pendingResourceReload bool                  // Reload resources after editor closes
	pendingConfigReload   bool                  // Reload config after editor closes

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

	// Agents tab state
	activeAgents       []ActiveAgent // Currently running agents
	agentCursor        int           // Selection cursor for agents tab
	agentViewMode      int           // 0=list, 1=detail
	selectedAgentIdx   int           // Index for detail view
	agentDetailScroll  int           // Scroll offset for detail view

	// Notification/Toast
	notification *Notification

	// Command Palette (cmd+k)
	palette Palette

	// MCP status
	mcpStatus []mcppkg.ServerStatus

	// Embedded terminal
	term EmbeddedTerm

	// AI Ask panel state
	askPanel *AskPanel
}

// AskPanel holds state for the AI ask feature
type AskPanel struct {
	Active       bool
	Input        string
	Response     string
	Loading      bool
	Error        string
	GeneratedCmd string // If AI generated a runnable command
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

// aiResponseMsg is sent when AI finishes responding
type aiResponseMsg struct {
	response     string
	generatedCmd string
	err          error
}

// agentInteractionMsg is sent when an agent interaction completes
type agentInteractionMsg struct {
	interaction config.AgentInteraction
}

// agentStartedMsg is sent when an agent starts running
type agentStartedMsg struct {
	agent ActiveAgent
}

// agentCompletedMsg is sent when an agent finishes
type agentCompletedMsg struct {
	agentID  string
	success  bool
	output   string
	duration int64
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
			ID:          "run_agent",
			Name:        "Run Agent",
			Icon:        "⚡",
			Description: "Run AI agent in Docker or E2B",
			Handler: func(m *model) tea.Cmd {
				return m.startRunAgentWizard()
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

// Functions moved to wizards.go, resources.go, ask_panel.go, view_handlers.go, mcp_status.go

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

	// Forward non-key messages to delete resource wizard form
	if m.deleteResourceWizard != nil && m.deleteResourceWizard.InputForm != nil {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			form, cmd := m.deleteResourceWizard.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.deleteResourceWizard.InputForm = f
				if f.State == huh.StateCompleted {
					return m, m.confirmDeleteResource()
				}
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Forward non-key messages to run agent wizard form
	if m.runAgentWizard != nil && m.runAgentWizard.InputForm != nil {
		if _, isKey := msg.(tea.KeyMsg); !isKey {
			form, cmd := m.runAgentWizard.InputForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.runAgentWizard.InputForm = f
				if f.State == huh.StateCompleted {
					return m, m.nextRunAgentStep()
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
		log.Printf("termStartMsg received: command=%s", msg.command)
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

	case agentStartedMsg:
		m.activeAgents = append(m.activeAgents, msg.agent)
		return m, nil

	case agentCompletedMsg:
		// Find and remove the agent from active list
		for i, agent := range m.activeAgents {
			if agent.ID == msg.agentID {
				// Create history entry
				interaction := config.AgentInteraction{
					ID:        agent.ID,
					Agent:     agent.Name,
					Action:    agent.Task,
					Input:     agent.Task,
					Output:    msg.output,
					Timestamp: agent.StartTime,
					Success:   msg.success,
					Runtime:   agent.Runtime,
					Provider:  agent.Provider,
					Duration:  msg.duration,
				}
				m.agentHistory = config.AddAgentInteraction(m.agentHistory, interaction, 50)
				config.SaveAgentHistory(m.agentHistory)

				// Remove from active agents
				m.activeAgents = append(m.activeAgents[:i], m.activeAgents[i+1:]...)
				break
			}
		}
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

	case aiResponseMsg:
		if m.askPanel != nil {
			m.askPanel.Loading = false
			if msg.err != nil {
				m.askPanel.Error = msg.err.Error()
			} else {
				m.askPanel.Response = msg.response
				m.askPanel.GeneratedCmd = msg.generatedCmd
			}
		}
		return m, nil

	case providerTestMsg:
		if m.providersWizard != nil {
			m.providersWizard.Testing = false
			if msg.success {
				m.providersWizard.TestResult = "Connection successful!"
				m.providersWizard.TestError = ""
				// Auto-save after successful test
				return m, m.saveProvider()
			} else {
				errMsg := "Connection failed"
				if msg.err != nil {
					errMsg = msg.err.Error()
					// Parse common errors for friendlier messages
					if strings.Contains(errMsg, "401") {
						errMsg = "Authentication failed - check your API key"
					} else if strings.Contains(errMsg, "connection refused") {
						errMsg = "Connection refused - is the server running?"
					}
				}
				m.providersWizard.TestError = errMsg
				m.providersWizard.TestResult = ""
			}
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
		return m.handleKeyMsg(msg)
	}

	return m, tea.Batch(cmds...)
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

	// If embedded terminal is active, show it regardless of view
	if m.term.active {
		return m.renderTerminalFullscreen()
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

// providerTestMsg is sent when provider test completes
type providerTestMsg struct {
	success bool
	err     error
}
