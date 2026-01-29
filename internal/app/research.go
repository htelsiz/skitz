package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yarlson/tap"

	"github.com/htelsiz/skitz/internal/config"
)

const researcherScript = `import asyncio
from fast_agent import FastAgent

fast = FastAgent("Deep Researcher")

@fast.agent(
    "web_searcher",
    """You are a thorough web researcher. Given a topic:
    1. Search for relevant, authoritative sources
    2. Extract key facts and insights
    3. Note any conflicting information
    4. Cite your sources""",
    servers=["fetch"],
)
@fast.agent(
    "quality_assurance",
    """You are a research quality evaluator. Review research for:
    - Accuracy and factual correctness
    - Completeness of coverage
    - Source credibility
    - Balanced perspectives
    
    Rate as EXCELLENT, GOOD, FAIR, or POOR with specific feedback.""",
)
@fast.evaluator_optimizer(
    name="researcher",
    generator="web_searcher",
    evaluator="quality_assurance",
    min_rating="GOOD",
    max_refinements=3,
)
async def main():
    async with fast.run() as agent:
        await agent.researcher.interactive()

if __name__ == "__main__":
    asyncio.run(main())
`

const researcherConfigYAML = `# Fast-Agent Research Configuration
default_model: sonnet

logger:
  progress_display: true
  
mcp:
  servers:
    fetch:
      command: uvx
      args: ["mcp-server-fetch"]
`

type deepResearchCmd struct {
	topic       string
	success     bool
	interaction config.AgentInteraction
}

func (c *deepResearchCmd) Run() error {
	fmt.Print("\033[H\033[2J")
	tap.Intro("🔬 Deep Research with Fast-Agent")

	if !checkFastAgent() {
		tap.Box("fast-agent is not installed.\n\nInstall with:\n  uv tool install fast-agent-mcp\n\nOr:\n  pip install fast-agent-mcp", "Setup Required", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	stty := exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	researchDir := filepath.Join(config.ConfigDir, "research")
	if err := setupResearchEnvironment(researchDir); err != nil {
		tap.Box(fmt.Sprintf("Failed to setup research environment: %v", err), "Error", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	fmt.Println()
	fmt.Println("📝 Enter your research topic or question.")
	fmt.Println("   The AI will search the web and synthesize findings.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Research topic: ")
	topic, err := reader.ReadString('\n')
	if err != nil {
		tap.Box(fmt.Sprintf("Failed to read input: %v", err), "Error", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	topic = strings.TrimSpace(topic)
	if topic == "" {
		tap.Cancel("No topic provided")
		waitForEnter()
		return nil
	}

	c.topic = topic

	fmt.Println()
	fmt.Printf("🔍 Researching: %s\n\n", topic)

	cmd := exec.Command("uv", "run", "researcher.py", "--agent", "researcher", "--message", topic)
	cmd.Dir = researchDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		c.success = false
		return nil
	}

	stty = exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	waitForEnter()
	tap.Outro("Research complete")
	c.success = true
	return nil
}

func (c deepResearchCmd) SetStdin(r io.Reader)  {}
func (c deepResearchCmd) SetStdout(w io.Writer) {}
func (c deepResearchCmd) SetStderr(w io.Writer) {}

type interactiveResearchCmd struct {
	success     bool
	interaction config.AgentInteraction
}

func (c *interactiveResearchCmd) Run() error {
	fmt.Print("\033[H\033[2J")
	tap.Intro("🔬 Interactive Research Session")

	if !checkFastAgent() {
		tap.Box("fast-agent is not installed.\n\nInstall with:\n  uv tool install fast-agent-mcp\n\nOr:\n  pip install fast-agent-mcp", "Setup Required", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	stty := exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	researchDir := filepath.Join(config.ConfigDir, "research")
	if err := setupResearchEnvironment(researchDir); err != nil {
		tap.Box(fmt.Sprintf("Failed to setup research environment: %v", err), "Error", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	fmt.Println()
	fmt.Println("Starting interactive research session...")
	fmt.Println("Type your questions and the AI will research them.")
	fmt.Println("Type 'exit' or press Ctrl+C to quit.")
	fmt.Println()

	cmd := exec.Command("uv", "run", "researcher.py", "--agent", "researcher")
	cmd.Dir = researchDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		c.success = false
		return nil
	}

	stty = exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	tap.Outro("Research session ended")
	c.success = true
	return nil
}

func (c interactiveResearchCmd) SetStdin(r io.Reader)  {}
func (c interactiveResearchCmd) SetStdout(w io.Writer) {}
func (c interactiveResearchCmd) SetStderr(w io.Writer) {}

func checkFastAgent() bool {
	cmd := exec.Command("fast-agent", "--version")
	return cmd.Run() == nil
}

func setupResearchEnvironment(researchDir string) error {
	if err := os.MkdirAll(researchDir, 0755); err != nil {
		return fmt.Errorf("failed to create research directory: %w", err)
	}

	scriptPath := filepath.Join(researchDir, "researcher.py")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		if err := os.WriteFile(scriptPath, []byte(researcherScript), 0644); err != nil {
			return fmt.Errorf("failed to write researcher script: %w", err)
		}
	}

	configPath := filepath.Join(researchDir, "fastagent.config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(researcherConfigYAML), 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	return nil
}

func runDeepResearch() tea.Cmd {
	cmd := &deepResearchCmd{}
	return tea.Exec(cmd, func(err error) tea.Msg {
		return tea.BatchMsg{
			func() tea.Msg {
				return commandDoneMsg{
					command: "deep-research",
					tool:    "fast-agent",
					success: cmd.success,
				}
			},
			func() tea.Msg {
				return agentInteractionMsg{
					interaction: config.AgentInteraction{
						Agent:   "Fast-Agent Researcher",
						Action:  "Deep Research",
						Input:   cmd.topic,
						Success: cmd.success,
					},
				}
			},
		}
	})
}

func runInteractiveResearch() tea.Cmd {
	cmd := &interactiveResearchCmd{}
	return tea.Exec(cmd, func(err error) tea.Msg {
		return tea.BatchMsg{
			func() tea.Msg {
				return commandDoneMsg{
					command: "interactive-research",
					tool:    "fast-agent",
					success: cmd.success,
				}
			},
			func() tea.Msg {
				return agentInteractionMsg{
					interaction: config.AgentInteraction{
						Agent:   "Fast-Agent Researcher",
						Action:  "Interactive Research",
						Success: cmd.success,
					},
				}
			},
		}
	})
}

