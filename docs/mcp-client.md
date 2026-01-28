# MCP Client Documentation

The skitz CLI includes a full-featured MCP (Model Context Protocol) client that enables integration with AI agent servers and tools.

## Overview

| Aspect | Details |
|--------|---------|
| Protocol Version | `2024-11-05` |
| Transport | HTTP (StreamableHttpClient) |
| Client Identity | `skitz` v1.0.0 |
| Library | `github.com/mark3labs/mcp-go` v0.43.2 |

## Architecture

```
┌─────────────────────────────────────────────────┐
│           User Interface (views.go)             │
│  Dashboard sidebar • Command palette            │
└────────────────────┬────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────┐
│        Application Layer (agent_mcp.go)         │
│  ReviewCodeWithBIA • CheckMCPServer             │
│  GetAvailableMCPTools                           │
└────────────────────┬────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────┐
│        MCP Client (mcp_client.go)               │
│  MCPClient struct • Tool invocation             │
│  Server discovery • Connection management       │
└────────────────────┬────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────┘
│        mcp-go Library (mark3labs)               │
│  HTTP transport • Protocol implementation       │
└─────────────────────────────────────────────────┘
```

---

## Configuration

### Config File

Location: `~/.config/skitz/config.yaml`

```yaml
mcp:
  enabled: true
  refresh_seconds: 60
  servers:
    - name: "bldrspec-ai"
      url: "http://localhost:8001/mcp/"
```

### Environment Variable

```bash
export BLDRSPEC_MCP_URL="http://localhost:8001/mcp/"
```

The environment variable overrides the config file for the default server URL.

### Config Types

```go
type MCPConfig struct {
    Enabled        bool              `yaml:"enabled"`
    RefreshSeconds int               `yaml:"refresh_seconds"`
    Servers        []MCPServerConfig `yaml:"servers"`
}

type MCPServerConfig struct {
    Name string `yaml:"name"`
    URL  string `yaml:"url"`
}
```

---

## Core Client API

### MCPClient Struct

```go
type MCPClient struct {
    client    *client.Client  // mcp-go client
    serverURL string          // Server URL
    connected bool            // Connection state
}
```

### Creating a Client

```go
// Create with explicit URL
client, err := NewMCPClient("http://localhost:8001/mcp/")

// Create with default URL (from env or config)
client, err := NewMCPClient("")

// Use the global singleton (recommended)
client, err := GetMCPClient()
```

### Connection Lifecycle

```go
// Connect to server
ctx := context.Background()
err := client.Connect(ctx)

// Check connection status
if client.IsConnected() {
    // ready to use
}

// Close connection
err := client.Close()

// Close global client
CloseMCPClient()
```

### Timeouts

| Operation | Timeout |
|-----------|---------|
| Initial connection | 10 seconds |
| Status check (ping) | 5 seconds |
| Tool execution (code review) | 120 seconds |

---

## Tool Operations

### List Available Tools

```go
tools, err := client.ListTools(ctx)
for _, tool := range tools {
    fmt.Printf("Tool: %s\n", tool.Name)
    fmt.Printf("  Description: %s\n", tool.Description)
    fmt.Printf("  Schema: %v\n", tool.InputSchema)
}
```

### Call a Tool

Three methods for different result handling:

```go
// Raw result (CallToolResult)
result, err := client.CallTool(ctx, "tool_name", map[string]any{
    "arg1": "value1",
    "arg2": 123,
})

// Extract text content
text, err := client.CallToolString(ctx, "tool_name", args)

// Parse JSON into struct
var response MyStruct
err := client.CallToolJSON(ctx, "tool_name", args, &response)
```

### Result Handling

The client handles both structured content and text content:

```go
// CallTool automatically extracts error messages
if result.IsError {
    // Error text extracted from content
}

// CallToolJSON tries structured content first, then text
if result.StructuredContent != nil {
    // Use structured content
} else {
    // Fall back to JSON in text content
}
```

---

## Server Discovery

### Fetch Server Status

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

status := FetchMCPServerStatus(ctx, "bldrspec-ai", "http://localhost:8001/mcp/")

fmt.Printf("Connected: %v\n", status.Connected)
fmt.Printf("Tools: %v\n", status.Tools)
fmt.Printf("Prompts: %v\n", status.Prompts)
fmt.Printf("Resources: %v\n", status.Resources)
fmt.Printf("Templates: %v\n", status.ResourceTemplates)
```

### MCPServerStatus Struct

```go
type MCPServerStatus struct {
    Name                   string
    URL                    string
    Connected              bool
    Tools                  []string  // Available tool names
    Prompts                []string  // Available prompts
    Resources              []string  // Available resources
    ResourceTemplates      []string  // Resource templates
    Error                  string    // Connection error
    ToolsError             string    // Error listing tools
    PromptsError           string    // Error listing prompts
    ResourcesError         string    // Error listing resources
    ResourceTemplatesError string    // Error listing templates
    LastUpdated            time.Time
}
```

### Health Check

```go
if err := client.Ping(ctx); err != nil {
    // Server unreachable
}
```

### Server Info

```go
name, sessionID := client.GetServerInfo()
```

---

## BIA Code Review Integration

The primary use case for the MCP client is the BIA Junior Agent code review.

### Usage

Via command palette: `Ctrl+K` → "BIA Code Review"

Or programmatically:

```go
ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
defer cancel()

feedback, err := ReviewCodeWithBIA(ctx, codeString)
if err != nil {
    log.Fatal(err)
}
fmt.Println(feedback)
```

### Helper Functions

```go
// Check if MCP server is available
if CheckMCPServer() {
    // Server is reachable
}

// List available tools
tools, err := GetAvailableMCPTools()
```

### Interaction Tracking

Agent interactions are tracked and stored:

```go
type AgentInteraction struct {
    Agent     string    `json:"agent"`      // "BIA Junior"
    Action    string    `json:"action"`     // "Code Review"
    Input     string    `json:"input"`      // Truncated to 100 chars
    Output    string    `json:"output"`     // Truncated to 200 chars
    Timestamp time.Time `json:"timestamp"`
    Success   bool      `json:"success"`
}
```

Storage: `~/.local/share/skitz/agent_history.json`

---

## Dashboard Integration

The MCP server status is displayed in the dashboard sidebar.

### Status Display

- Connection indicator: `✓` connected / `✗` disconnected
- Tool count and names
- Prompt count
- Resource count
- Template count
- Last updated timestamp
- Error messages (if any)

### Auto-Refresh

Status refreshes automatically based on `refresh_seconds` config (default: 60s).

The refresh cycle:

```
Init() → fetchMCPStatusCmd() → mcpStatusMsg → scheduleMCPRefreshCmd()
                                                      ↓
                                              mcpRefreshTickMsg
                                                      ↓
                                              fetchMCPStatusCmd() → ...
```

---

## Error Handling

### Connection Errors

```go
client, err := GetMCPClient()
if err != nil {
    // "failed to create MCP client: ..."
    // "failed to start MCP client: ..."
    // "failed to initialize MCP client: ..."
}
```

### Tool Call Errors

```go
result, err := client.CallTool(ctx, "tool_name", args)
if err != nil {
    // "MCP client not connected"
    // "failed to call tool X: ..."
    // "tool error: <message from server>"
    // "tool X returned an error"
}
```

### Result Extraction Errors

```go
text, err := client.CallToolString(ctx, name, args)
if err != nil {
    // "no text content in tool result"
}

err := client.CallToolJSON(ctx, name, args, &target)
if err != nil {
    // "failed to marshal structured content: ..."
    // "no content in tool result"
}
```

---

## Testing

### Unit Tests

```bash
go test -run TestMCPClient ./...
```

### Integration Tests

Requires a running MCP server:

```bash
MCP_TEST_ENABLED=1 go test -run TestMCPIntegration ./...
```

### Test File

`mcp_client_test.go` includes tests for:
- Client creation
- Connection management
- Tool listing

---

## Files Reference

| File | Purpose |
|------|---------|
| `mcp_client.go` | Core MCP client wrapper (318 lines) |
| `agent_mcp.go` | BIA integration & tool usage (362 lines) |
| `config.go` | MCPConfig struct & loading |
| `types.go` | MCPServerStatus definition |
| `model.go` | Tea event loop, status refresh |
| `views.go` | Dashboard MCP status display |
| `mcp_client_test.go` | Client tests |

---

## Extending the Client

### Adding a New Tool Integration

1. Create a wrapper function in `agent_mcp.go`:

```go
func CallMyTool(ctx context.Context, input string) (string, error) {
    client, err := GetMCPClient()
    if err != nil {
        return "", err
    }

    return client.CallToolString(ctx, "my_tool", map[string]any{
        "input": input,
    })
}
```

2. Add to command palette in `commands.go`
3. Create a tea.Cmd wrapper if needed for TUI integration

### Adding a New Server

Update config:

```yaml
mcp:
  servers:
    - name: "bldrspec-ai"
      url: "http://localhost:8001/mcp/"
    - name: "another-server"
      url: "http://localhost:9000/mcp/"
```

The dashboard will show status for all configured servers.

---

## Limitations

| Limitation | Notes |
|------------|-------|
| No streaming | Tool calls are request/response only |
| HTTP only | SSE transport not implemented |
| Single active client | Global singleton pattern |
| Tool schema unused | UI doesn't auto-generate input forms |

See [ROADMAP.md](./ROADMAP.md) for planned improvements.
