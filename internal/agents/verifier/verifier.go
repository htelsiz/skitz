package verifier

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yarlson/tap"

	"github.com/htelsiz/skitz/internal/config"
)

const verifierScript = `import asyncio
import subprocess
import os
from fast_agent import FastAgent

fast = FastAgent("Documentation Verifier")

@fast.tool
def run_in_sandbox(command: str) -> str:
    """Runs a shell command in a secure sandbox (Alpine Docker container).
    
    Args:
        command: The shell command to execute.
        
    Returns:
        The command output (stdout + stderr) or error message.
    """
    try:
        # Run in a disposable alpine container
        # We mount nothing to keep it secure and isolated
        print(f"\n[Sandbox] Running: {command}")
        result = subprocess.run(
            ["docker", "run", "--rm", "alpine", "sh", "-c", command],
            capture_output=True,
            text=True,
            timeout=30
        )
        
        output = result.stdout
        if result.stderr:
            output += "\nSTDERR:\n" + result.stderr
            
        return f"Exit Code: {result.returncode}\nOutput:\n{output}"
    except Exception as e:
        return f"Error executing command: {str(e)}"

@fast.agent(
    "verifier",
    """You are a Documentation Verifier. Your goal is to research a topic, extract commands, and verify them.
    
    1. Search for documentation on the user's topic.
    2. Extract key commands mentioned in the documentation.
    3. Verify these commands by running them in the sandbox.
    4. Report your findings:
       - Which commands worked?
       - Which commands failed and why?
       - Are the documentation examples accurate?
       
    When running commands, assume a clean Alpine Linux environment.
    If a command requires packages not in Alpine base, try to install them (e.g., apk add ...).
    """,
    servers=["fetch"],
    tools=[run_in_sandbox],
)
async def main():
    async with fast.run() as agent:
        await agent.verifier.interactive()

if __name__ == "__main__":
    asyncio.run(main())
`

const verifierConfigYAML = `# Fast-Agent Verifier Configuration
default_model: sonnet

logger:
  progress_display: true
  
mcp:
  servers:
    fetch:
      command: uvx
      args: ["mcp-server-fetch"]
`

// Cmd implements tea.ExecCommand for the verifier agent
type Cmd struct {
	Topic       string
	Success     bool
	Interaction config.AgentInteraction
}

// Run executes the agent process
func (c *Cmd) Run() error {
	fmt.Print("\033[H\033[2J")
	tap.Intro("🕵️ Documentation Verifier")

	if !checkFastAgent() {
		tap.Box("fast-agent is not installed.\n\nInstall with:\n  uv tool install fast-agent-mcp\n\nOr:\n  pip install fast-agent-mcp", "Setup Required", tap.BoxOptions{})
		waitForEnter()
		return nil
	}
	
	if _, err := exec.LookPath("docker"); err != nil {
		tap.Box("Docker is not installed or not in PATH.\nRequired for sandbox execution.", "Setup Required", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	stty := exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	verifierDir := filepath.Join(config.ConfigDir, "agents", "verifier")
	if err := setupVerifierEnvironment(verifierDir); err != nil {
		tap.Box(fmt.Sprintf("Failed to setup verifier environment: %v", err), "Error", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	fmt.Println()
	fmt.Println("📝 Enter a topic to research and verify.")
	fmt.Println("   The agent will find docs and test commands in a sandbox.")
	fmt.Println()

	var topic string
	if c.Topic != "" {
		topic = c.Topic
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Topic: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			tap.Box(fmt.Sprintf("Failed to read input: %v", err), "Error", tap.BoxOptions{})
			waitForEnter()
			return nil
		}
		topic = strings.TrimSpace(input)
	}

	if topic == "" {
		tap.Cancel("No topic provided")
		waitForEnter()
		return nil
	}

	c.Topic = topic
	c.Interaction = config.AgentInteraction{
		Agent:     "Doc Verifier",
		Action:    "Verify",
		Input:     topic,
		Timestamp: time.Now(),
	}

	fmt.Println()
	fmt.Printf("🔍 Researching and Verifying: %s\n\n", topic)

	cmd := exec.Command("uv", "run", "verifier.py", "--agent", "verifier", "--message", topic)
	cmd.Dir = verifierDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		c.Success = false
		c.Interaction.Success = false
		c.Interaction.Output = fmt.Sprintf("Error: %v", err)
		// Don't return error here, just mark as failed interaction
		// returning error would cause tea.Exec to fail without returning messages
	} else {
		c.Success = true
		c.Interaction.Success = true
		c.Interaction.Output = "Completed successfully"
	}
	
	duration := time.Since(start)
	c.Interaction.Duration = duration.Milliseconds()

	stty = exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	waitForEnter()
	tap.Outro("Verification complete")
	return nil
}

func (c *Cmd) SetStdin(r io.Reader)  {}
func (c *Cmd) SetStdout(w io.Writer) {}
func (c *Cmd) SetStderr(w io.Writer) {}

func checkFastAgent() bool {
	cmd := exec.Command("fast-agent", "--version")
	return cmd.Run() == nil
}

func setupVerifierEnvironment(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	scriptPath := filepath.Join(dir, "verifier.py")
	if err := os.WriteFile(scriptPath, []byte(verifierScript), 0644); err != nil {
		return fmt.Errorf("failed to write script: %w", err)
	}

	configPath := filepath.Join(dir, "fastagent.config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(verifierConfigYAML), 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	return nil
}

func waitForEnter() {
	fmt.Print("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
