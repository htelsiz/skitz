package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestMCPClientConnection(t *testing.T) {
	// Skip if MCP server is not available
	serverURL := getMCPServerURL()
	if os.Getenv("MCP_TEST_ENABLED") == "" {
		t.Skipf("MCP_TEST_ENABLED not set, skipping MCP client test (server URL: %s)", serverURL)
	}

	client, err := NewMCPClient(serverURL)
	if err != nil {
		t.Fatalf("Failed to create MCP client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test connection
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !client.IsConnected() {
		t.Error("Client should be connected")
	}

	// Test ping
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Test list tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	t.Logf("Found %d tools:", len(tools))
	for _, tool := range tools {
		t.Logf("  - %s: %s", tool.Name, tool.Description[:min(50, len(tool.Description))]+"...")
	}

	// Verify at least one tool exists
	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}

	// Test server info
	name, sessionID := client.GetServerInfo()
	t.Logf("Server: %s, Session: %s", name, sessionID)
}

func TestMCPClientNotConnected(t *testing.T) {
	client, err := NewMCPClient("http://localhost:9999/mcp/")
	if err != nil {
		t.Fatalf("Failed to create MCP client: %v", err)
	}

	ctx := context.Background()

	// Should fail when not connected
	_, err = client.ListTools(ctx)
	if err == nil {
		t.Error("Expected error when calling ListTools without connection")
	}

	_, err = client.CallTool(ctx, "test", nil)
	if err == nil {
		t.Error("Expected error when calling CallTool without connection")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
