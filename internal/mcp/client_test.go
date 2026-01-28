package mcp

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestClientConnection(t *testing.T) {
	serverURL := GetServerURL()
	if os.Getenv("MCP_TEST_ENABLED") == "" {
		t.Skipf("MCP_TEST_ENABLED not set, skipping MCP client test (server URL: %s)", serverURL)
	}

	client, err := NewClient(serverURL)
	if err != nil {
		t.Fatalf("Failed to create MCP client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !client.IsConnected() {
		t.Error("Client should be connected")
	}

	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	t.Logf("Found %d tools:", len(tools))
	for _, tool := range tools {
		desc := tool.Description
		if len(desc) > 50 {
			desc = desc[:50] + "..."
		}
		t.Logf("  - %s: %s", tool.Name, desc)
	}

	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}

	name, sessionID := client.GetServerInfo()
	t.Logf("Server: %s, Session: %s", name, sessionID)
}

func TestClientNotConnected(t *testing.T) {
	client, err := NewClient("http://localhost:9999/mcp/")
	if err != nil {
		t.Fatalf("Failed to create MCP client: %v", err)
	}

	ctx := context.Background()

	_, err = client.ListTools(ctx)
	if err == nil {
		t.Error("Expected error when calling ListTools without connection")
	}

	_, err = client.CallTool(ctx, "test", nil)
	if err == nil {
		t.Error("Expected error when calling CallTool without connection")
	}
}
