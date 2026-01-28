package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yarlson/tap"
)

// BIAJuniorAgentResult represents the result from bia_junior_agent
type BIAJuniorAgentResult struct {
	Feedback string `json:"feedback"`
}

// ReviewCodeWithBIA sends code to the BIA Junior Agent for review via MCP
func ReviewCodeWithBIA(ctx context.Context, code string) (string, error) {
	client, err := GetMCPClient()
	if err != nil {
		return "", fmt.Errorf("failed to get MCP client: %w", err)
	}

	result, err := client.CallTool(ctx, "bia_junior_agent", map[string]any{
		"code": code,
	})
	if err != nil {
		return "", fmt.Errorf("failed to call bia_junior_agent: %w", err)
	}

	// Extract text content from result
	return extractTextFromResult(result)
}

// ReviewCodeWithBIAStream sends code to BIA and streams the response
// For now, this is non-streaming since MCP tool calls are request/response
func ReviewCodeWithBIAStream(ctx context.Context, code string, onChunk func(string)) error {
	response, err := ReviewCodeWithBIA(ctx, code)
	if err != nil {
		return err
	}

	// Simulate streaming by sending the whole response
	// TODO: Implement true streaming if MCP supports it
	onChunk(response)
	return nil
}

// extractTextFromResult extracts text content from MCP CallToolResult
func extractTextFromResult(result *mcp.CallToolResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("nil result")
	}

	// Check for structured content first
	if result.StructuredContent != nil {
		if m, ok := result.StructuredContent.(map[string]any); ok {
			if feedback, ok := m["feedback"].(string); ok {
				return feedback, nil
			}
			if text, ok := m["text"].(string); ok {
				return text, nil
			}
		}
	}

	// Fall back to text content
	var texts []string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			texts = append(texts, textContent.Text)
		}
	}

	if len(texts) == 0 {
		return "", fmt.Errorf("no text content in result")
	}

	return texts[0], nil
}

// CheckMCPServer checks if the MCP server is available
func CheckMCPServer() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := GetMCPClient()
	if err != nil {
		return false
	}

	return client.Ping(ctx) == nil
}

// GetAvailableMCPTools returns the list of available MCP tools
func GetAvailableMCPTools() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := GetMCPClient()
	if err != nil {
		return nil, err
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names, nil
}

// biaCodeReviewCmd implements tea.ExecCommand for BIA code review
type biaCodeReviewCmd struct {
	success     bool
	interaction AgentInteraction
}

func (c *biaCodeReviewCmd) Run() error {
	ctx := context.Background()

	fmt.Print("\033[H\033[2J")
	tap.Intro("🔍 BIA Code Review")

	// Check MCP server connection
	spinner := tap.NewSpinner(tap.SpinnerOptions{})
	spinner.Start("Connecting to MCP server...")

	if !CheckMCPServer() {
		spinner.Stop("", 0)
		tap.Box("MCP server not available.\nMake sure bldrspec-ai is running:\n  docker compose up", "Error", tap.BoxOptions{})
		waitForEnterMCP()
		return nil
	}
	spinner.Stop("Connected to MCP server", 1)

	// Restore terminal for input
	stty := exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	// Ask for input method
	fmt.Println()
	inputOptions := []tap.SelectOption[string]{
		{Value: "file", Label: "Enter file path", Hint: "Review a file from disk"},
		{Value: "paste", Label: "Paste code", Hint: "Paste code directly"},
	}
	inputType := tap.Select(ctx, tap.SelectOptions[string]{
		Message: "How would you like to provide code?",
		Options: inputOptions,
	})

	var code string
	reader := bufio.NewReader(os.Stdin)

	if inputType == "file" {
		// File path mode
		filePath := tap.Text(ctx, tap.TextOptions{
			Message:     "File path:",
			Placeholder: "e.g., ./main.py or /path/to/file.py",
		})

		if filePath == "" {
			tap.Cancel("No file path provided")
			return nil
		}

		// Expand ~ to home directory
		if strings.HasPrefix(filePath, "~/") {
			home, _ := os.UserHomeDir()
			filePath = home + filePath[1:]
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			tap.Box(fmt.Sprintf("Failed to read file: %v", err), "Error", tap.BoxOptions{})
			waitForEnterMCP()
			return nil
		}
		code = string(content)
	} else {
		// Restore terminal again after tap.Select (it uses raw mode)
		stty := exec.Command("stty", "sane")
		stty.Stdin = os.Stdin
		stty.Run()

		// Paste mode with double-enter to submit
		fmt.Println("\n📝 Paste your code below.")
		fmt.Println("   Press Enter twice on an empty line to submit.")
		fmt.Println()

		var lines []string
		emptyLineCount := 0

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}

			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				emptyLineCount++
				if emptyLineCount >= 2 {
					// Double empty line - submit
					break
				}
				lines = append(lines, line)
			} else {
				emptyLineCount = 0
				lines = append(lines, line)
			}
		}

		code = strings.TrimRight(strings.Join(lines, ""), "\n\t ")
	}

	if strings.TrimSpace(code) == "" {
		tap.Cancel("No code provided")
		return nil
	}

	lineCount := strings.Count(code, "\n") + 1
	fmt.Printf("\n📊 Reviewing %d lines of code...\n\n", lineCount)

	// Call BIA Junior Agent
	reviewCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	spinner2 := tap.NewSpinner(tap.SpinnerOptions{})
	spinner2.Start("Analyzing code...")

	feedback, err := ReviewCodeWithBIA(reviewCtx, code)
	spinner2.Stop("Analysis complete", 1)

	// Track the interaction
	c.interaction = AgentInteraction{
		Agent:     "BIA Junior",
		Action:    "Code Review",
		Timestamp: time.Now(),
	}

	// Truncate input for storage (first 100 chars)
	inputSummary := strings.TrimSpace(code)
	if len(inputSummary) > 100 {
		inputSummary = inputSummary[:100] + "..."
	}
	c.interaction.Input = inputSummary

	if err != nil {
		c.interaction.Success = false
		c.interaction.Output = err.Error()
		tap.Box(fmt.Sprintf("Review failed: %v", err), "Error", tap.BoxOptions{})
		waitForEnterMCP()
		return nil
	}

	// Truncate output for storage (first 200 chars)
	outputSummary := strings.TrimSpace(feedback)
	if len(outputSummary) > 200 {
		outputSummary = outputSummary[:200] + "..."
	}
	c.interaction.Output = outputSummary
	c.interaction.Success = true

	// Display feedback with glamour markdown rendering
	fmt.Println()
	rendered, err := renderMarkdown(feedback)
	if err != nil {
		fmt.Println(feedback) // Fallback to plain text
	} else {
		fmt.Println(rendered)
	}

	// Restore terminal before waiting for input
	stty = exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	// Wait for user to read before returning
	waitForEnterMCP()

	tap.Outro("Review complete")
	c.success = true
	return nil
}

func (c biaCodeReviewCmd) SetStdin(r io.Reader)  {}
func (c biaCodeReviewCmd) SetStdout(w io.Writer) {}
func (c biaCodeReviewCmd) SetStderr(w io.Writer) {}

// runBIACodeReview creates a tea.Cmd that runs the BIA code review
func runBIACodeReview() tea.Cmd {
	cmd := &biaCodeReviewCmd{}
	return tea.Exec(cmd, func(err error) tea.Msg {
		// Return batch of messages to update both command and agent history
		return tea.BatchMsg{
			func() tea.Msg {
				return commandDoneMsg{
					command: "bia-review",
					tool:    "skitz",
					success: cmd.success,
				}
			},
			func() tea.Msg {
				return agentInteractionMsg{
					interaction: cmd.interaction,
				}
			},
		}
	})
}

// waitForEnterMCP waits for user to press enter
func waitForEnterMCP() {
	fmt.Print("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// renderMarkdown renders markdown text using glamour with full terminal width
func renderMarkdown(text string) (string, error) {
	// Get terminal width
	width := 120 // default
	if w, _, err := getTerminalSize(); err == nil && w > 0 {
		width = w - 4 // leave some margin
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(customStyleJSON)),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return r.Render(text)
}

// getTerminalSize returns the terminal dimensions
func getTerminalSize() (int, int, error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	var rows, cols int
	fmt.Sscanf(string(out), "%d %d", &rows, &cols)
	return cols, rows, nil
}

// mcpToolCmd implements tea.ExecCommand for running MCP tools
type mcpToolCmd struct {
	serverName  string
	serverURL   string
	tool        mcp.Tool
	success     bool
	interaction AgentInteraction
}

func (c *mcpToolCmd) Run() error {
	ctx := context.Background()

	fmt.Print("\033[H\033[2J")
	tap.Intro(fmt.Sprintf("⚡ %s", c.tool.Name))

	// Show tool description
	if c.tool.Description != "" {
		fmt.Printf("\n%s\n", c.tool.Description)
	}

	// Restore terminal for input
	stty := exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	// Collect parameters if the tool has any
	args := make(map[string]any)

	if len(c.tool.InputSchema.Properties) > 0 {
		// Build required map for quick lookup
		required := make(map[string]bool)
		for _, r := range c.tool.InputSchema.Required {
			required[r] = true
		}

		// Sort parameter names for stable order
		var paramNames []string
		for paramName := range c.tool.InputSchema.Properties {
			paramNames = append(paramNames, paramName)
		}
		sort.Slice(paramNames, func(i, j int) bool {
			reqI := required[paramNames[i]]
			reqJ := required[paramNames[j]]
			if reqI != reqJ {
				return reqI
			}
			return paramNames[i] < paramNames[j]
		})

		fmt.Println("\n📝 Enter parameters:")
		for _, paramName := range paramNames {
			paramDef := c.tool.InputSchema.Properties[paramName]
			paramMap, ok := paramDef.(map[string]interface{})
			if !ok {
				continue
			}

			description := ""
			if desc, ok := paramMap["description"].(string); ok {
				description = desc
			}

			isRequired := required[paramName]
			label := paramName
			if isRequired {
				label = paramName + " *"
			}

			placeholder := description
			if placeholder == "" {
				placeholder = fmt.Sprintf("Enter %s", paramName)
			}

			value := tap.Text(ctx, tap.TextOptions{
				Message:     label,
				Placeholder: placeholder,
			})

			if value != "" {
				args[paramName] = value
			} else if isRequired {
				tap.Cancel(fmt.Sprintf("Required parameter '%s' not provided", paramName))
				waitForEnterMCP()
				return nil
			}
		}
	}

	// Track the interaction
	c.interaction = AgentInteraction{
		Agent:     c.serverName,
		Action:    c.tool.Name,
		Timestamp: time.Now(),
	}

	// Call the MCP tool
	fmt.Printf("\n🔄 Calling %s...\n\n", c.tool.Name)

	callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	spinner := tap.NewSpinner(tap.SpinnerOptions{})
	spinner.Start("Executing tool...")

	// Create a new client for this call
	client, err := NewMCPClient(c.serverURL)
	if err != nil {
		spinner.Stop("", 0)
		c.interaction.Success = false
		c.interaction.Output = err.Error()
		tap.Box(fmt.Sprintf("Failed to create client: %v", err), "Error", tap.BoxOptions{})
		waitForEnterMCP()
		return nil
	}

	if err := client.Connect(callCtx); err != nil {
		spinner.Stop("", 0)
		c.interaction.Success = false
		c.interaction.Output = err.Error()
		tap.Box(fmt.Sprintf("Failed to connect: %v", err), "Error", tap.BoxOptions{})
		waitForEnterMCP()
		return nil
	}
	defer client.Close()

	result, err := client.CallTool(callCtx, c.tool.Name, args)
	spinner.Stop("Complete", 1)

	if err != nil {
		c.interaction.Success = false
		c.interaction.Output = err.Error()
		tap.Box(fmt.Sprintf("Tool execution failed: %v", err), "Error", tap.BoxOptions{})
		waitForEnterMCP()
		return nil
	}

	// Extract and display result
	output, err := extractTextFromResult(result)
	if err != nil {
		c.interaction.Success = false
		c.interaction.Output = err.Error()
		tap.Box(fmt.Sprintf("Failed to parse result: %v", err), "Error", tap.BoxOptions{})
		waitForEnterMCP()
		return nil
	}

	// Truncate output for storage
	outputSummary := strings.TrimSpace(output)
	if len(outputSummary) > 200 {
		outputSummary = outputSummary[:200] + "..."
	}
	c.interaction.Output = outputSummary
	c.interaction.Success = true

	// Display result with glamour markdown rendering
	fmt.Println()
	rendered, err := renderMarkdown(output)
	if err != nil {
		fmt.Println(output) // Fallback to plain text
	} else {
		fmt.Println(rendered)
	}

	// Restore terminal before waiting for input
	stty = exec.Command("stty", "sane")
	stty.Stdin = os.Stdin
	stty.Run()

	waitForEnterMCP()
	tap.Outro("Tool execution complete")
	c.success = true
	return nil
}

func (c mcpToolCmd) SetStdin(r io.Reader)  {}
func (c mcpToolCmd) SetStdout(w io.Writer) {}
func (c mcpToolCmd) SetStderr(w io.Writer) {}

// runMCPTool creates a tea.Cmd that runs an MCP tool
func runMCPTool(serverName string, serverURL string, tool mcp.Tool) tea.Cmd {
	cmd := &mcpToolCmd{serverName: serverName, serverURL: serverURL, tool: tool}
	return tea.Exec(cmd, func(err error) tea.Msg {
		return tea.BatchMsg{
			func() tea.Msg {
				return commandDoneMsg{
					command: fmt.Sprintf("mcp:%s", tool.Name),
					tool:    serverName,
					success: cmd.success,
				}
			},
			func() tea.Msg {
				return agentInteractionMsg{
					interaction: cmd.interaction,
				}
			},
		}
	})
}
