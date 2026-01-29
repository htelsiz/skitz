package app

import (
	"log"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandMode determines how a command is executed.
type CommandMode string

const (
	CommandEmbedded    CommandMode = "embedded"
	CommandInteractive CommandMode = "interactive"
)

// CommandSpec describes a command to execute.
type CommandSpec struct {
	Command string
	Mode    CommandMode
}

func resolveShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "/bin/sh"
	}
	return shell
}

func newShellCommand(command string) *exec.Cmd {
	return exec.Command(resolveShell(), "-c", command)
}

func (m *model) runCommand(spec CommandSpec) tea.Cmd {
	if strings.TrimSpace(spec.Command) == "" {
		log.Println("runCommand: empty command")
		return nil
	}

	log.Printf("runCommand: mode=%s cmd=%s", spec.Mode, spec.Command)

	switch spec.Mode {
	case CommandInteractive:
		log.Println("runCommand: using interactive mode")
		return m.executeInteractive(command{cmd: spec.Command}, spec.Command)
	default:
		log.Println("runCommand: using embedded mode")
		return m.executeEmbedded(spec.Command)
	}
}
