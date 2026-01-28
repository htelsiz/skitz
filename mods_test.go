package main

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestModsInstalled checks if mods is installed
func TestModsInstalled(t *testing.T) {
	cmd := exec.Command("which", "mods")
	err := cmd.Run()
	if err != nil {
		t.Skip("mods not installed, skipping test. Install with: brew install charmbracelet/tap/mods")
	}
}

// TestModsBasicCall tests that mods can be called and returns output
func TestModsBasicCall(t *testing.T) {
	cmd := exec.Command("which", "mods")
	if err := cmd.Run(); err != nil {
		t.Skip("mods not installed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, "mods")
	cmd.Stdin = strings.NewReader("Say hello in JSON format: {\"message\": \"your response here\"}")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mods call failed: %v\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		t.Fatal("mods returned empty output")
	}

	t.Logf("mods output: %s", result)
}

// TestModsJSONResponse tests that mods can return valid JSON
func TestModsJSONResponse(t *testing.T) {
	cmd := exec.Command("which", "mods")
	if err := cmd.Run(); err != nil {
		t.Skip("mods not installed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	prompt := `Return ONLY a JSON object with these fields:
{
  "name": "test",
  "count": 42,
  "active": true
}
Do not include any other text, just the JSON.`

	cmd = exec.CommandContext(ctx, "mods")
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mods call failed: %v\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	t.Logf("mods output: %s", result)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Failed to parse mods output as JSON: %v\nOutput: %s", err, result)
	}

	if parsed["name"] == nil {
		t.Error("Expected 'name' field in response")
	}
}

// TestModsToolParameterExtraction simulates the actual use case
func TestModsToolParameterExtraction(t *testing.T) {
	cmd := exec.Command("which", "mods")
	if err := cmd.Run(); err != nil {
		t.Skip("mods not installed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	prompt := `You are helping execute an MCP tool. Based on the user's request, determine the appropriate parameter values.

Tool: datadog_query
Description: Fetch logs from Datadog API

Parameters Schema:
- service (required): string - Service name to query
- status (required): string - Log status (ERROR, WARN, INFO)
- hours_ago: number - Hours ago to start search (default: 1)

User Request: Find all errors from the frontend service in the last 2 hours

Respond with ONLY a JSON object containing the parameter values. Example: {"param1": "value1", "param2": 123}
Make reasonable assumptions for any missing information. Return only valid JSON, no additional text.`

	cmd = exec.CommandContext(ctx, "mods")
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mods call failed: %v\nOutput: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	t.Logf("mods output: %s", result)

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(result), &params); err != nil {
		t.Fatalf("Failed to parse mods output as JSON: %v\nOutput: %s", err, result)
	}

	// Check that expected parameters are present
	if params["service"] == nil {
		t.Error("Expected 'service' parameter")
	}
	if params["status"] == nil {
		t.Error("Expected 'status' parameter")
	}

	t.Logf("Extracted parameters: %+v", params)
}
