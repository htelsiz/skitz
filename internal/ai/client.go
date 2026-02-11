package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/htelsiz/skitz/internal/config"
)

// Client handles AI provider API calls
type Client struct {
	provider   config.ProviderConfig
	httpClient *http.Client
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents an AI response
type Response struct {
	Content string
	Error   error
}

// NewClient creates a new AI client for the given provider
func NewClient(provider config.ProviderConfig) *Client {
	return &Client{
		provider: provider,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetDefaultClient returns a client for the default provider
func GetDefaultClient(cfg config.Config) (*Client, error) {
	if cfg.AI.DefaultProvider == "" {
		return nil, fmt.Errorf("no default provider configured")
	}

	for _, p := range cfg.AI.Providers {
		if p.Name == cfg.AI.DefaultProvider && p.Enabled {
			return NewClient(p), nil
		}
	}

	return nil, fmt.Errorf("default provider '%s' not found or disabled", cfg.AI.DefaultProvider)
}

// Ask sends a question to the AI with optional context
func (c *Client) Ask(question string, context string) Response {
	systemPrompt := `You are a helpful CLI assistant for skitz, a command center tool.
You help users understand and work with command-line tools.
Be concise and practical. When suggesting commands, format them in backticks.
If you suggest a runnable command, put it on its own line starting with $ like: $ command here`

	if context != "" {
		systemPrompt += "\n\nHere is the current resource content for context:\n" + context
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: question},
	}

	return c.chat(messages)
}

// EditResource asks the AI to edit a resource file based on an instruction
func (c *Client) EditResource(instruction string, currentContent string) Response {
	systemPrompt := `You are a resource file editor for skitz, a CLI command center tool.
You edit markdown resource files that contain command references.

Resource file format rules:
- Lines with ` + "`command here`" + ` description ^run are runnable commands
- Lines with ` + "`command {{var}}`" + ` description ^run:var prompt for the variable before running
- Regular markdown (headings, lists, text) for documentation
- ## headings create navigable sections

Given the current resource content and the user's edit instruction,
output ONLY the complete updated resource file content.
Do not wrap in code fences. Do not include any explanation, just the raw markdown content.
Preserve the existing format and annotations (^run, ^run:varname) unless told to change them.
When adding new runnable commands, include the ^run annotation.`

	userMsg := "Current resource content:\n```\n" + currentContent + "\n```\n\nEdit instruction: " + instruction

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}

	return c.chat(messages)
}

// GenerateCommand asks the AI to generate a specific command
func (c *Client) GenerateCommand(description string, context string) Response {
	systemPrompt := `You are a command generator for CLI tools.
Given a description of what the user wants to do, generate the appropriate command.
ONLY output the command itself, nothing else. No explanation, no backticks, just the raw command.
If you cannot generate a valid command, respond with "ERROR: " followed by a brief explanation.`

	if context != "" {
		systemPrompt += "\n\nHere are example commands from the current resource:\n" + context
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: description},
	}

	return c.chat(messages)
}

// DetectProviderType determines the provider type from API key format, URL, or name
func DetectProviderType(apiKey, baseURL, name string) string {
	// 1. Check API key format first (most reliable)
	if strings.HasPrefix(apiKey, "sk-ant-") {
		return "anthropic"
	}
	if strings.HasPrefix(apiKey, "sk-") && !strings.HasPrefix(apiKey, "sk-ant-") {
		return "openai"
	}

	// 2. Check base URL
	if strings.Contains(baseURL, "anthropic.com") {
		return "anthropic"
	}
	if strings.Contains(baseURL, "openai.com") {
		return "openai"
	}
	if strings.Contains(baseURL, "11434") || strings.Contains(baseURL, "ollama") {
		return "ollama"
	}

	// 3. Check name as fallback
	nameLower := strings.ToLower(name)
	if strings.Contains(nameLower, "anthropic") || strings.Contains(nameLower, "claude") {
		return "anthropic"
	}
	if strings.Contains(nameLower, "openai") || strings.Contains(nameLower, "gpt") {
		return "openai"
	}
	if strings.Contains(nameLower, "ollama") || strings.Contains(nameLower, "llama") {
		return "ollama"
	}

	// Default to openai-compatible
	return "openai"
}

func (c *Client) chat(messages []Message) Response {
	// Use explicit provider type if set, otherwise detect
	providerType := c.provider.ProviderType
	if providerType == "" {
		providerType = DetectProviderType(c.provider.APIKey, c.provider.BaseURL, c.provider.Name)
	}

	switch providerType {
	case "anthropic":
		return c.callAnthropic(messages)
	case "ollama":
		return c.callOllama(messages)
	default:
		return c.callOpenAI(messages)
	}
}

// TestConnection verifies the provider connection works
func (c *Client) TestConnection() error {
	// Send a minimal request to verify authentication
	messages := []Message{
		{Role: "user", Content: "Hi"},
	}

	resp := c.chat(messages)
	return resp.Error
}

// OpenAI API format
func (c *Client) callOpenAI(messages []Message) Response {
	baseURL := c.provider.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := c.provider.DefaultModel
	if model == "" {
		model = "gpt-4"
	}

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Response{Error: err}
	}

	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{Error: err}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.provider.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{Error: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{Error: err}
	}

	if resp.StatusCode != 200 {
		return Response{Error: fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))}
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return Response{Error: err}
	}

	if len(result.Choices) == 0 {
		return Response{Error: fmt.Errorf("no response from API")}
	}

	return Response{Content: result.Choices[0].Message.Content}
}

// Anthropic API format
func (c *Client) callAnthropic(messages []Message) Response {
	baseURL := c.provider.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	model := c.provider.DefaultModel
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	// Convert messages to Anthropic format
	var systemPrompt string
	var anthropicMessages []map[string]string

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			anthropicMessages = append(anthropicMessages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": 2048,
		"messages":   anthropicMessages,
	}
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Response{Error: err}
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return Response{Error: err}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{Error: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{Error: err}
	}

	if resp.StatusCode != 200 {
		return Response{Error: fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))}
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return Response{Error: err}
	}

	if len(result.Content) == 0 {
		return Response{Error: fmt.Errorf("no response from API")}
	}

	return Response{Content: result.Content[0].Text}
}

// Ollama API format
func (c *Client) callOllama(messages []Message) Response {
	baseURL := c.provider.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := c.provider.DefaultModel
	if model == "" {
		model = "llama3"
	}

	// Convert to Ollama format
	var ollamaMessages []map[string]string
	for _, msg := range messages {
		ollamaMessages = append(ollamaMessages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": ollamaMessages,
		"stream":   false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Response{Error: err}
	}

	req, err := http.NewRequest("POST", baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Response{Error: err}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{Error: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{Error: err}
	}

	if resp.StatusCode != 200 {
		return Response{Error: fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))}
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return Response{Error: err}
	}

	return Response{Content: result.Message.Content}
}
