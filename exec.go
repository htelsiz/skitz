package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aaronjanse/3mux/vterm"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
)

// commandDoneMsg signals that command execution is complete
type commandDoneMsg struct {
	command string
	tool    string
	success bool
}

// interactiveCmd implements tea.ExecCommand for interactive execution
type interactiveCmd struct {
	cmd        string
	needsInput bool
	inputVar   string
	tool       string
	finalCmd   string // Populated after execution for history
	success    bool
}

func (c *interactiveCmd) Run() error {
	finalCmd := c.cmd

	// Clear screen
	fmt.Print("\033[H\033[2J")

	// Store the final command for history
	c.finalCmd = finalCmd

	// Execute directly - input already handled by caller
	cmd := newShellCommand(finalCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()

	fmt.Println()
	if err != nil {
		fmt.Printf("\033[31mCommand failed: %v\033[0m\n", err)
		c.success = false
	} else {
		fmt.Println("\033[32mCommand completed.\033[0m")
		c.success = true
	}
	fmt.Print("\nPress Enter to return to skitz...")
	bufio.NewReader(os.Stdin).ReadLine()

	return nil
}

func (c interactiveCmd) SetStdin(r io.Reader)  {}
func (c interactiveCmd) SetStdout(w io.Writer) {}
func (c interactiveCmd) SetStderr(w io.Writer) {}

// isInteractiveCommand checks if a command needs full terminal control
func isInteractiveCommand(cmd string) bool {
	// Commands that need full TUI/interactive terminal
	interactivePatterns := []string{
		"claude",      // Claude Code CLI
		"vim", "nvim", "vi", // Editors
		"htop", "top", "btop", // System monitors
		"less", "more", // Pagers
		"ssh",         // Remote shells
		"docker run",  // Interactive containers
		"-it",         // Docker interactive flag
		"--interactive",
	}

	cmdLower := strings.ToLower(cmd)
	for _, pattern := range interactivePatterns {
		if strings.Contains(cmdLower, pattern) {
			return true
		}
	}
	return false
}

// executeInteractive runs a command with full terminal control using tea.Exec
func (m *model) executeInteractive(cmd command, finalCmd string) tea.Cmd {
	toolName := ""
	if res := m.currentResource(); res != nil {
		toolName = res.name
	}

	ic := &interactiveCmd{
		cmd:        finalCmd,
		needsInput: false, // Already handled above
		inputVar:   "",
		tool:       toolName,
	}
	return tea.Exec(ic, func(err error) tea.Msg {
		return commandDoneMsg{
			command: ic.finalCmd,
			tool:    ic.tool,
			success: ic.success,
		}
	})
}

// executeCommandDirect executes a command directly (used by actions)
func executeCommandDirect(cmd string) {
	c := newShellCommand(cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.Run()
}

// termStartMsg is sent when terminal is ready
type termStartMsg struct {
	vt     *vterm.VTerm
	pty    *os.File
	cmd    *exec.Cmd
	width  int
	height int
}

// executeEmbedded runs a command in an embedded terminal pane
func (m *model) executeEmbedded(cmdStr string) tea.Cmd {
	// Terminal pane dimensions
	termW := m.width - 6
	termH := 20
	if termW < 40 {
		termW = 40
	}
	if termH < 10 {
		termH = 10
	}

	return func() tea.Msg {
		// Silence vterm debug logs temporarily, then restore
		oldLogOutput := log.Writer()
		log.SetOutput(io.Discard)
		defer log.SetOutput(oldLogOutput)

		// Create the command - use user's shell to inherit environment (Azure CLI auth, etc.)
		c := newShellCommand(cmdStr)
		c.Env = append(os.Environ(),
			"TERM=xterm-256color",
			fmt.Sprintf("COLUMNS=%d", termW),
			fmt.Sprintf("LINES=%d", termH),
		)

		// Start PTY
		ptmx, err := pty.StartWithSize(c, &pty.Winsize{
			Rows: uint16(termH),
			Cols: uint16(termW),
		})
		if err != nil {
			return termExitMsg{err: err}
		}

		// Create vterm
		renderer := &termRenderer{}
		vt := vterm.NewVTerm(renderer, func(x, y int) {})
		vt.Reshape(0, 0, termW, termH)

		return termStartMsg{
			vt:     vt,
			pty:    ptmx,
			cmd:    c,
			width:  termW,
			height: termH,
		}
	}
}

