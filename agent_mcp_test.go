package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestReviewCodeWithBIA(t *testing.T) {
	if os.Getenv("MCP_TEST_ENABLED") == "" {
		t.Skip("MCP_TEST_ENABLED not set, skipping")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	code := `
def hello(name):
    print("Hello " + name)

hello("World")
`

	result, err := ReviewCodeWithBIA(ctx, code)
	if err != nil {
		t.Fatalf("Failed to review code: %v", err)
	}

	t.Logf("Review result:\n%s", result)

	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestGetAvailableMCPTools(t *testing.T) {
	if os.Getenv("MCP_TEST_ENABLED") == "" {
		t.Skip("MCP_TEST_ENABLED not set, skipping")
	}

	tools, err := GetAvailableMCPTools()
	if err != nil {
		t.Fatalf("Failed to get tools: %v", err)
	}

	t.Logf("Available tools: %v", tools)

	found := false
	for _, tool := range tools {
		if tool == "bia_junior_agent" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected bia_junior_agent tool to be available")
	}
}
