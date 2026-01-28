package app

import (
	"testing"
)

// TestCheckAzureCLI verifies Azure CLI detection
func TestCheckAzureCLI(t *testing.T) {
	result := checkAzureCLI()
	t.Logf("checkAzureCLI() = %v", result)
}
