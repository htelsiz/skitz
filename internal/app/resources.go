package app

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/htelsiz/skitz/internal/config"
	"github.com/htelsiz/skitz/internal/resources"
)

func (m *model) loadResources() {
	m.resources = nil
	seen := make(map[string]bool)

	descriptions := map[string]string{
		"claude":     "AI coding assistant CLI",
		"docker":     "Container management",
		"git":        "Version control & GitHub CLI",
		"mcp":        "Model Context Protocol",
		"azure":      "Cloud resource management",
		"cursor":     "AI-powered code editor",
		"fast-agent": "MCP-native AI agent framework",
		"e2b":        "Cloud sandbox for AI agents",
		"gcp":        "Google Cloud CLI commands",
		"codex":      "OpenAI CLI coding agent",
		"nixos":      "NixOS system configuration",
		"go":         "Go programming language",
		"rust":       "Rust programming language",
		"tailscale":  "Mesh VPN & network management",
	}

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

	entries, err := resources.Default.ReadDir(".")
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, "-detail.md") {
				resName := strings.TrimSuffix(name, ".md")
				if seen[resName] {
					continue
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

func (m *model) editResource() tea.Cmd {
	res := m.currentResource()
	if res == nil {
		return m.showNotification("!", "No resource selected", "error")
	}

	if err := os.MkdirAll(config.ResourcesDir, 0755); err != nil {
		return m.showNotification("!", "Failed to create directory: "+err.Error(), "error")
	}

	filePath := filepath.Join(config.ResourcesDir, res.name+".md")

	if res.embedded {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := os.WriteFile(filePath, []byte(res.content), 0644); err != nil {
				return m.showNotification("!", "Failed to copy resource: "+err.Error(), "error")
			}
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		for _, e := range []string{"vim", "vi", "nano"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return m.showNotification("!", "No editor found. Set $EDITOR", "error")
	}

	m.pendingResourceReload = true
	return m.runCommand(CommandSpec{
		Command: fmt.Sprintf("%s %q", editor, filePath),
		Mode:    CommandInteractive,
	})
}

func (m *model) applyAIEdit(editedContent string) tea.Cmd {
	res := m.currentResource()
	if res == nil {
		return m.showNotification("!", "No resource selected", "error")
	}

	if err := os.MkdirAll(config.ResourcesDir, 0755); err != nil {
		return m.showNotification("!", "Failed to create directory: "+err.Error(), "error")
	}

	filePath := filepath.Join(config.ResourcesDir, res.name+".md")

	if err := os.WriteFile(filePath, []byte(editedContent), 0644); err != nil {
		return m.showNotification("!", "Failed to save: "+err.Error(), "error")
	}

	m.loadResources()
	m.askPanel = nil
	m.initViewComponents()

	return m.showNotification("", "Resource updated via AI", "success")
}

func (m *model) addCommandToResource(cmd string) tea.Cmd {
	res := m.currentResource()
	if res == nil {
		return m.showNotification("!", "No resource selected", "error")
	}

	if err := os.MkdirAll(config.ResourcesDir, 0755); err != nil {
		return m.showNotification("!", "Failed to create directory: "+err.Error(), "error")
	}

	filePath := filepath.Join(config.ResourcesDir, res.name+".md")

	var content string
	if res.embedded {
		content = res.content
	} else {
		data, err := os.ReadFile(filePath)
		if err != nil {
			content = res.content
		} else {
			content = string(data)
		}
	}

	newLine := fmt.Sprintf("\n`%s` AI generated ^run\n", cmd)
	content += newLine

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return m.showNotification("!", "Failed to save: "+err.Error(), "error")
	}

	m.loadResources()
	m.askPanel = nil

	m.initViewComponents()

	return m.showNotification("✓", "Command added to resource", "success")
}
