package app

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

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

	commandList := m.renderCommandList(m.contentView.Width, meta.color)

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
