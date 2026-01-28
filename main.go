package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

var resourcesDir string
var configDir string
var dataDir string

func init() {
	home, _ := os.UserHomeDir()
	resourcesDir = filepath.Join(home, "dotfiles", "resources")
	configDir = filepath.Join(home, ".config", "skitz")
	dataDir = filepath.Join(home, ".local", "share", "skitz")
}

func main() {
	resource := ""
	if len(os.Args) > 1 {
		resource = os.Args[1]
		if _, err := os.Stat(filepath.Join(resourcesDir, resource+".md")); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Resource not found: %s\n", resource)
			os.Exit(1)
		}
	}

	if _, err := tea.NewProgram(newModel(resource), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
