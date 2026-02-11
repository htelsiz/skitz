package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/htelsiz/skitz/internal/ai"
)

func (m *model) submitAskPanel() tea.Cmd {
	if m.askPanel == nil || m.askPanel.Input == "" {
		return nil
	}

	m.askPanel.Loading = true
	m.askPanel.Response = ""
	m.askPanel.Error = ""
	m.askPanel.GeneratedCmd = ""

	question := m.askPanel.Input
	context := ""
	if res := m.currentResource(); res != nil {
		context = res.content
	}

	return func() tea.Msg {
		client, err := ai.GetDefaultClient(m.config)
		if err != nil {
			return aiResponseMsg{err: err}
		}

		resp := client.Ask(question, context)
		if resp.Error != nil {
			return aiResponseMsg{err: resp.Error}
		}

		var generatedCmd string
		lines := strings.Split(resp.Content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "$ ") {
				generatedCmd = strings.TrimPrefix(line, "$ ")
				break
			}
		}

		return aiResponseMsg{
			response:     resp.Content,
			generatedCmd: generatedCmd,
		}
	}
}

func (m *model) submitAIEdit() tea.Cmd {
	if m.askPanel == nil || m.askPanel.Input == "" {
		return nil
	}

	m.askPanel.Loading = true
	m.askPanel.Response = ""
	m.askPanel.Error = ""
	m.askPanel.EditedContent = ""

	instruction := m.askPanel.Input
	content := ""
	if res := m.currentResource(); res != nil {
		content = res.content
	}

	return func() tea.Msg {
		client, err := ai.GetDefaultClient(m.config)
		if err != nil {
			return aiEditResponseMsg{err: err}
		}

		resp := client.EditResource(instruction, content)
		if resp.Error != nil {
			return aiEditResponseMsg{err: resp.Error}
		}

		edited := strings.TrimSpace(resp.Content)

		// Generate a brief summary of what changed
		summary := "Resource updated based on: " + instruction

		return aiEditResponseMsg{
			editedContent: edited,
			summary:       summary,
		}
	}
}

func (m *model) submitGenerateCommand() tea.Cmd {
	if m.askPanel == nil || m.askPanel.Input == "" {
		return nil
	}

	m.askPanel.Loading = true
	m.askPanel.Response = ""
	m.askPanel.Error = ""
	m.askPanel.GeneratedCmd = ""

	description := m.askPanel.Input
	context := ""
	if res := m.currentResource(); res != nil {
		for _, cmd := range m.commands {
			context += cmd.raw + "\n"
		}
	}

	return func() tea.Msg {
		client, err := ai.GetDefaultClient(m.config)
		if err != nil {
			return aiResponseMsg{err: err}
		}

		resp := client.GenerateCommand(description, context)
		if resp.Error != nil {
			return aiResponseMsg{err: resp.Error}
		}

		content := strings.TrimSpace(resp.Content)
		if strings.HasPrefix(content, "ERROR:") {
			return aiResponseMsg{
				response: content,
			}
		}

		return aiResponseMsg{
			response:     "Generated command:",
			generatedCmd: content,
		}
	}
}
