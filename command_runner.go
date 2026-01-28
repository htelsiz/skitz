package main

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type CommandMode string

const (
	CommandEmbedded    CommandMode = "embedded"
	CommandInteractive CommandMode = "interactive"
)

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
		return nil
	}

	switch spec.Mode {
	case CommandInteractive:
		return m.executeInteractive(command{cmd: spec.Command}, spec.Command)
	default:
		return m.executeEmbedded(spec.Command)
	}
}
