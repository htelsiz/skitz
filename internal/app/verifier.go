package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/htelsiz/skitz/internal/agents/verifier"
)

func (m *model) startVerifierWizard() tea.Cmd {
	m.palette.WizardState = &wizardState{
		Type: "verifier",
		Step: 0,
		Data: make(map[string]any),
	}
	return m.nextVerifierStep()
}

func (m *model) nextVerifierStep() tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil {
		return nil
	}

	switch ws.Step {
	case 0:
		m.palette.InputValue = ""
		input := huh.NewText().
			Title("Verification Topic").
			Description("What topic or resource should I verify?").
			Placeholder("e.g. Docker run commands").
			Value(&m.palette.InputValue)

		m.palette.InputForm = huh.NewForm(huh.NewGroup(input)).
			WithWidth(80).
			WithHeight(8).
			WithShowHelp(false).
			WithShowErrors(false).
			WithTheme(huh.ThemeCatppuccin())

		m.palette.State = PaletteStateCollectingParams
		return m.palette.InputForm.Init()

	case 1:
		topic := ws.Data["topic"].(string)
		if strings.TrimSpace(topic) == "" {
			return func() tea.Msg {
				return aiAgentResultMsg{
					title:  "Doc Verifier",
					output: "No topic provided",
					err:    fmt.Errorf("no topic provided"),
				}
			}
		}

		m.palette.WizardState = nil
		m.palette.State = PaletteStateIdle
		return runVerifierAgent(topic)
	}

	return nil
}

func runVerifierAgent(topic string) tea.Cmd {
	cmd := &verifier.Cmd{Topic: topic}
	return tea.Exec(cmd, func(err error) tea.Msg {
		if err != nil {
			return commandDoneMsg{
				command: "verify",
				tool:    "verifier",
				success: false,
			}
		}
		return tea.BatchMsg{
			func() tea.Msg {
				return commandDoneMsg{
					command: "verify",
					tool:    "verifier",
					success: cmd.Success,
				}
			},
			func() tea.Msg {
				return agentInteractionMsg{
					interaction: cmd.Interaction,
				}
			},
		}
	})
}
