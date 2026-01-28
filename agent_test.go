package main

import (
	"testing"
)

// TestCheckAzureCLI verifies Azure CLI detection
// Note: checkAzureCLI is defined in deploy.go
func TestCheckAzureCLI(t *testing.T) {
	// This test checks that the function exists and runs
	// The actual result depends on whether az is installed
	result := checkAzureCLI()
	t.Logf("checkAzureCLI() = %v", result)
}
