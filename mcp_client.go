package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClient wraps the mcp-go client for bldrspec-ai server
type MCPClient struct {
	client    *client.Client
	serverURL string
	connected bool
}

// Default MCP server URL
const defaultMCPServerURL = "http://localhost:8001/mcp/"

// getMCPServerURL returns the MCP server URL from env or default
func getMCPServerURL() string {
	if url := os.Getenv("BLDRSPEC_MCP_URL"); url != "" {
		return url
	}
	return defaultMCPServerURL
}

// GetDefaultMCPServerURL returns the default MCP server URL
func GetDefaultMCPServerURL() string {
	return getMCPServerURL()
}

func buildInitializeRequest() mcp.InitializeRequest {
	return mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			ClientInfo: mcp.Implementation{
				Name:    "skitz",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}
}

// NewMCPClient creates a new MCP client for the given server URL
func NewMCPClient(serverURL string) (*MCPClient, error) {
	if serverURL == "" {
		serverURL = getMCPServerURL()
	}

	c, err := client.NewStreamableHttpClient(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	return &MCPClient{
		client:    c,
		serverURL: serverURL,
		connected: false,
	}, nil
}

// Connect initializes the MCP connection
func (m *MCPClient) Connect(ctx context.Context) error {
	if m.connected {
		return nil
	}

	// Start the transport connection
	if err := m.client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP client: %w", err)
	}

	_, err := m.client.Initialize(ctx, buildInitializeRequest())
	if err != nil {
		m.client.Close()
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	m.connected = true
	return nil
}

// Close closes the MCP connection
func (m *MCPClient) Close() error {
	if !m.connected {
		return nil
	}
	m.connected = false
	return m.client.Close()
}

// IsConnected returns whether the client is connected
func (m *MCPClient) IsConnected() bool {
	return m.connected
}

// ListTools returns the available tools from the MCP server
func (m *MCPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if !m.connected {
		return nil, fmt.Errorf("MCP client not connected")
	}

	result, err := m.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return result.Tools, nil
}

// CallTool calls an MCP tool with the given arguments and returns the result
func (m *MCPClient) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	if !m.connected {
		return nil, fmt.Errorf("MCP client not connected")
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}

	result, err := m.client.CallTool(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", name, err)
	}

	if result.IsError {
		// Extract error message from content
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				return nil, fmt.Errorf("tool error: %s", textContent.Text)
			}
		}
		return nil, fmt.Errorf("tool %s returned an error", name)
	}

	return result, nil
}

// CallToolString calls a tool and returns the text content as a string
func (m *MCPClient) CallToolString(ctx context.Context, name string, args map[string]any) (string, error) {
	result, err := m.CallTool(ctx, name, args)
	if err != nil {
		return "", err
	}

	// Extract text from content
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			return textContent.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in tool result")
}

// CallToolJSON calls a tool and unmarshals the result into the given target
func (m *MCPClient) CallToolJSON(ctx context.Context, name string, args map[string]any, target any) error {
	result, err := m.CallTool(ctx, name, args)
	if err != nil {
		return err
	}

	// Try structured content first
	if result.StructuredContent != nil {
		data, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return fmt.Errorf("failed to marshal structured content: %w", err)
		}
		return json.Unmarshal(data, target)
	}

	// Fall back to text content (assuming JSON)
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			return json.Unmarshal([]byte(textContent.Text), target)
		}
	}

	return fmt.Errorf("no content in tool result")
}

// Ping checks if the MCP server is reachable
func (m *MCPClient) Ping(ctx context.Context) error {
	if !m.connected {
		return fmt.Errorf("MCP client not connected")
	}
	return m.client.Ping(ctx)
}

// GetServerInfo returns information about the connected MCP server
func (m *MCPClient) GetServerInfo() (name string, sessionID string) {
	if !m.connected {
		return "", ""
	}
	return "bldrspec-ai", m.client.GetSessionId()
}

// FetchMCPTools connects to an MCP server and returns the available tools.
// This is a standalone function (like FetchMCPServerStatus) that creates its own connection.
func FetchMCPTools(ctx context.Context, url string) ([]mcp.Tool, error) {
	if url == "" {
		return nil, fmt.Errorf("missing server URL")
	}

	c, err := client.NewStreamableHttpClient(url)
	if err != nil {
		return nil, fmt.Errorf("client init: %w", err)
	}
	defer c.Close()

	if err := c.Start(ctx); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if _, err := c.Initialize(ctx, buildInitializeRequest()); err != nil {
		return nil, fmt.Errorf("init: %w", err)
	}

	result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	return result.Tools, nil
}

// FetchMCPServerStatus connects to the given MCP server and returns status data.
func FetchMCPServerStatus(ctx context.Context, name string, url string) MCPServerStatus {
	status := MCPServerStatus{
		Name:        name,
		URL:         url,
		Connected:   false,
		LastUpdated: time.Now(),
	}

	if url == "" {
		status.Error = "missing server URL"
		return status
	}

	c, err := client.NewStreamableHttpClient(url)
	if err != nil {
		status.Error = fmt.Sprintf("client init: %v", err)
		return status
	}
	defer c.Close()

	if err := c.Start(ctx); err != nil {
		status.Error = fmt.Sprintf("connect: %v", err)
		return status
	}

	if _, err := c.Initialize(ctx, buildInitializeRequest()); err != nil {
		status.Error = fmt.Sprintf("init: %v", err)
		return status
	}

	status.Connected = true
	tools, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		status.ToolsError = fmt.Sprintf("tools: %v", err)
	} else {
		status.Tools = make([]string, len(tools.Tools))
		for i, tool := range tools.Tools {
			status.Tools[i] = tool.Name
		}
	}

	prompts, err := c.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		status.PromptsError = fmt.Sprintf("prompts: %v", err)
	} else {
		status.Prompts = make([]string, len(prompts.Prompts))
		for i, prompt := range prompts.Prompts {
			status.Prompts[i] = prompt.Name
		}
	}

	resources, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		status.ResourcesError = fmt.Sprintf("resources: %v", err)
	} else {
		status.Resources = make([]string, len(resources.Resources))
		for i, resource := range resources.Resources {
			name := resource.Name
			if name == "" {
				name = resource.URI
			}
			status.Resources[i] = name
		}
	}

	templates, err := c.ListResourceTemplates(ctx, mcp.ListResourceTemplatesRequest{})
	if err != nil {
		status.ResourceTemplatesError = fmt.Sprintf("templates: %v", err)
	} else {
		status.ResourceTemplates = make([]string, len(templates.ResourceTemplates))
		for i, tmpl := range templates.ResourceTemplates {
			status.ResourceTemplates[i] = tmpl.Name
		}
	}

	return status
}

// Global MCP client instance (lazy initialized)
var globalMCPClient *MCPClient

// GetMCPClient returns the global MCP client, creating it if necessary
func GetMCPClient() (*MCPClient, error) {
	if globalMCPClient != nil && globalMCPClient.IsConnected() {
		return globalMCPClient, nil
	}

	var err error
	globalMCPClient, err = NewMCPClient("")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := globalMCPClient.Connect(ctx); err != nil {
		return nil, err
	}

	return globalMCPClient, nil
}

// CloseMCPClient closes the global MCP client
func CloseMCPClient() {
	if globalMCPClient != nil {
		globalMCPClient.Close()
		globalMCPClient = nil
	}
}
