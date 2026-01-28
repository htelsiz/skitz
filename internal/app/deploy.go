package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yarlson/tap"
)

// AgentType represents the type of agent to deploy
type AgentType string

const (
	AgentClaude AgentType = "claude"
	AgentCursor AgentType = "cursor"
	AgentCustom AgentType = "custom"
)

// DeployMethod represents how to deploy the agent
type DeployMethod string

const (
	DeployACI      DeployMethod = "aci"
	DeployPipeline DeployMethod = "pipeline"
)

// DeployConfig holds the configuration for agent deployment
type DeployConfig struct {
	AgentType     AgentType
	DeployMethod  DeployMethod
	AgentName     string
	ResourceGroup string
	Location      string
	Prompt        string
	AIAccount     string
	AIEndpoint    string
	AIDeployment  string
	AIModel       string
}

// AzureAIAccount represents an Azure AI Services account
type AzureAIAccount struct {
	Name          string
	ResourceGroup string
	Location      string
	Endpoint      string
	Kind          string
}

// AzureAIDeployment represents a model deployment
type AzureAIDeployment struct {
	Name     string
	Model    string
	Version  string
	SKU      string
	Capacity int
}

// deployAgentCmd implements tea.ExecCommand for interactive deployment
type deployAgentCmd struct {
	success bool
}

func (c *deployAgentCmd) Run() error {
	ctx := context.Background()

	fmt.Print("\033[H\033[2J")

	tap.Intro("🚀 Deploy Agent")

	if !checkAzureCLI() {
		tap.Box("Azure CLI required. Install from:\nhttps://docs.microsoft.com/en-us/cli/azure/install-azure-cli", "Error", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	spinner := tap.NewSpinner(tap.SpinnerOptions{})
	spinner.Start("Loading Azure AI accounts...")
	accounts := getAzureAIAccounts()
	spinner.Stop("", 0)

	if len(accounts) == 0 {
		tap.Box("No Azure AI accounts found.\nCreate one at: https://ai.azure.com", "No AI Accounts", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	accountOptions := make([]tap.SelectOption[string], len(accounts))
	accountMap := make(map[string]AzureAIAccount)
	for i, acc := range accounts {
		accountOptions[i] = tap.SelectOption[string]{
			Value: acc.Name,
			Label: acc.Name,
			Hint:  fmt.Sprintf("%s (%s)", acc.Kind, acc.Location),
		}
		accountMap[acc.Name] = acc
	}

	selectedAccount := tap.Select(ctx, tap.SelectOptions[string]{
		Message: "Select Azure AI account:",
		Options: accountOptions,
	})
	if selectedAccount == "" {
		tap.Cancel("Cancelled")
		return nil
	}
	account := accountMap[selectedAccount]

	spinner.Start("Loading model deployments...")
	deployments := getAzureAIDeployments(account.ResourceGroup, account.Name)
	spinner.Stop("", 0)

	if len(deployments) == 0 {
		tap.Box("No model deployments found in this account.\nDeploy a model at: https://ai.azure.com", "No Deployments", tap.BoxOptions{})
		waitForEnter()
		return nil
	}

	deploymentOptions := make([]tap.SelectOption[string], len(deployments))
	deploymentMap := make(map[string]AzureAIDeployment)
	for i, dep := range deployments {
		hint := dep.Model
		if dep.Version != "" {
			hint += " v" + dep.Version
		}
		deploymentOptions[i] = tap.SelectOption[string]{
			Value: dep.Name,
			Label: dep.Name,
			Hint:  hint,
		}
		deploymentMap[dep.Name] = dep
	}

	selectedDeployment := tap.Select(ctx, tap.SelectOptions[string]{
		Message: "Select model deployment:",
		Options: deploymentOptions,
	})
	if selectedDeployment == "" {
		tap.Cancel("Cancelled")
		return nil
	}
	deployment := deploymentMap[selectedDeployment]

	deployOptions := []tap.SelectOption[DeployMethod]{
		{Value: DeployACI, Label: "Azure Container Instance", Hint: "Run once in a container"},
		{Value: DeployPipeline, Label: "Azure Pipeline", Hint: "Run as CI/CD pipeline"},
	}

	deployMethod := tap.Select(ctx, tap.SelectOptions[DeployMethod]{
		Message: "How to run:",
		Options: deployOptions,
	})
	if deployMethod == "" {
		tap.Cancel("Cancelled")
		return nil
	}

	prompt := tap.Text(ctx, tap.TextOptions{
		Message:     "Task for the agent:",
		Placeholder: "Review this PR and suggest improvements...",
	})
	if prompt == "" {
		tap.Cancel("Cancelled")
		return nil
	}

	agentType := AgentCustom
	modelLower := strings.ToLower(deployment.Model)
	if strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "openai") {
		agentType = AgentCursor
	} else if strings.Contains(modelLower, "claude") {
		agentType = AgentClaude
	}

	dconfig := DeployConfig{
		AgentType:     agentType,
		DeployMethod:  deployMethod,
		AgentName:     fmt.Sprintf("agent-%d", time.Now().Unix()),
		ResourceGroup: account.ResourceGroup,
		Location:      account.Location,
		Prompt:        prompt,
		AIAccount:     account.Name,
		AIEndpoint:    account.Endpoint,
		AIDeployment:  deployment.Name,
		AIModel:       deployment.Model,
	}

	summaryText := fmt.Sprintf(`AI Account:  %s
Model:       %s (%s)
Method:      %s
Task:        %s`,
		dconfig.AIAccount,
		dconfig.AIDeployment,
		dconfig.AIModel,
		dconfig.DeployMethod,
		truncateStr(dconfig.Prompt, 35),
	)
	tap.Box(summaryText, "Deployment Summary", tap.BoxOptions{})

	confirmed := tap.Confirm(ctx, tap.ConfirmOptions{
		Message: "Deploy now?",
	})
	if !confirmed {
		tap.Cancel("Cancelled")
		return nil
	}

	spinner.Start("Getting API key...")
	apiKey := getAzureAIKey(dconfig.ResourceGroup, dconfig.AIAccount)
	if apiKey == "" {
		spinner.Stop("Failed to get API key", 1)
		waitForEnter()
		return nil
	}
	spinner.Stop("Ready", 0)

	switch dconfig.DeployMethod {
	case DeployACI:
		spinner.Start("Deploying container...")

		image := "python:3.11-slim"
		envVars := []string{
			fmt.Sprintf("AZURE_OPENAI_ENDPOINT=%s", dconfig.AIEndpoint),
			fmt.Sprintf("AZURE_OPENAI_API_KEY=%s", apiKey),
			fmt.Sprintf("AZURE_OPENAI_DEPLOYMENT=%s", dconfig.AIDeployment),
			fmt.Sprintf("AGENT_PROMPT=%s", dconfig.Prompt),
		}

		script := generateAzureAgentScript(dconfig)

		args := []string{
			"container", "create",
			"--resource-group", dconfig.ResourceGroup,
			"--name", dconfig.AgentName,
			"--image", image,
			"--restart-policy", "Never",
			"--location", dconfig.Location,
		}

		for _, env := range envVars {
			args = append(args, "--environment-variables", env)
		}

		args = append(args, "--command-line", script)

		aciCmd := exec.Command("az", args...)
		output, err := aciCmd.CombinedOutput()
		if err != nil {
			spinner.Stop("Deployment failed: "+string(output), 1)
			return nil
		}
		spinner.Stop("Container deployed!", 0)

		showLogs := tap.Confirm(ctx, tap.ConfirmOptions{
			Message: "Stream container logs?",
		})
		if showLogs {
			fmt.Println("\n--- Container Logs ---")
			logsCmd := exec.Command("az", "container", "logs",
				"--resource-group", dconfig.ResourceGroup,
				"--name", dconfig.AgentName,
				"--follow",
			)
			logsCmd.Stdout = os.Stdout
			logsCmd.Stderr = os.Stderr
			logsCmd.Run()
		}

	case DeployPipeline:
		if !checkAzureDevOpsCLI() {
			spinner.Stop("Azure DevOps CLI required", 1)
			tap.Box("Install with: az extension add --name azure-devops", "Setup Required", tap.BoxOptions{})
			waitForEnter()
			return nil
		}

		orgURL := tap.Text(ctx, tap.TextOptions{
			Message:     "Azure DevOps Org URL:",
			Placeholder: "https://dev.azure.com/myorg",
		})
		project := tap.Text(ctx, tap.TextOptions{
			Message:     "Project name:",
			Placeholder: "MyProject",
		})

		spinner.Start("Creating pipeline run...")

		tmpYAML := fmt.Sprintf(`trigger: none
pool:
  vmImage: ubuntu-latest
steps:
- script: |
    pip install openai
    python3 -c "
    from openai import AzureOpenAI
    client = AzureOpenAI(
        azure_endpoint='%s',
        api_key='%s',
        api_version='2024-02-15-preview'
    )
    response = client.chat.completions.create(
        model='%s',
        messages=[{'role': 'user', 'content': '''%s'''}]
    )
    print(response.choices[0].message.content)
    "
  displayName: 'Run AI Agent'
`, dconfig.AIEndpoint, apiKey, dconfig.AIDeployment, dconfig.Prompt)

		tmpFile := filepath.Join(os.TempDir(), "agent-pipeline.yml")
		os.WriteFile(tmpFile, []byte(tmpYAML), 0644)

		runCmd := exec.Command("az", "pipelines", "run",
			"--org", orgURL,
			"--project", project,
			"--name", dconfig.AgentName,
		)
		output, err := runCmd.CombinedOutput()
		if err != nil {
			spinner.Stop("Pipeline failed: "+string(output), 1)
		} else {
			spinner.Stop("Pipeline started!", 0)
		}
	}

	tap.Outro("🎉 Agent deployed!")

	fmt.Print("\nPress Enter to return to skitz...")
	fmt.Scanln()

	c.success = true
	return nil
}

func (c deployAgentCmd) SetStdin(r io.Reader)  {}
func (c deployAgentCmd) SetStdout(w io.Writer) {}
func (c deployAgentCmd) SetStderr(w io.Writer) {}

// checkAzureCLI checks if Azure CLI is installed
func checkAzureCLI() bool {
	cmd := exec.Command("az", "--version")
	return cmd.Run() == nil
}

// checkAzureDevOpsCLI checks if the Azure DevOps CLI extension is installed
func checkAzureDevOpsCLI() bool {
	cmd := exec.Command("az", "extension", "show", "--name", "azure-devops")
	return cmd.Run() == nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func waitForEnter() {
	fmt.Print("\nPress Enter to continue...")
	fmt.Scanln()
}

func getAzureAIAccounts() []AzureAIAccount {
	cmd := exec.Command("az", "cognitiveservices", "account", "list",
		"--query", "[?kind=='OpenAI' || kind=='CognitiveServices'].{name:name, resourceGroup:resourceGroup, location:location, endpoint:properties.endpoint, kind:kind}",
		"-o", "json",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var accounts []AzureAIAccount
	lines := strings.TrimSpace(string(output))
	if lines == "" || lines == "[]" {
		return nil
	}

	type jsonAccount struct {
		Name          string `json:"name"`
		ResourceGroup string `json:"resourceGroup"`
		Location      string `json:"location"`
		Endpoint      string `json:"endpoint"`
		Kind          string `json:"kind"`
	}
	var jsonAccounts []jsonAccount

	if err := parseJSON(output, &jsonAccounts); err != nil {
		return nil
	}

	for _, ja := range jsonAccounts {
		accounts = append(accounts, AzureAIAccount{
			Name:          ja.Name,
			ResourceGroup: ja.ResourceGroup,
			Location:      ja.Location,
			Endpoint:      ja.Endpoint,
			Kind:          ja.Kind,
		})
	}

	return accounts
}

func getAzureAIDeployments(resourceGroup, accountName string) []AzureAIDeployment {
	cmd := exec.Command("az", "cognitiveservices", "account", "deployment", "list",
		"--resource-group", resourceGroup,
		"--name", accountName,
		"--query", "[].{name:name, model:properties.model.name, version:properties.model.version, sku:sku.name, capacity:sku.capacity}",
		"-o", "json",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	type jsonDeployment struct {
		Name     string `json:"name"`
		Model    string `json:"model"`
		Version  string `json:"version"`
		SKU      string `json:"sku"`
		Capacity int    `json:"capacity"`
	}
	var jsonDeployments []jsonDeployment

	if err := parseJSON(output, &jsonDeployments); err != nil {
		return nil
	}

	var deployments []AzureAIDeployment
	for _, jd := range jsonDeployments {
		deployments = append(deployments, AzureAIDeployment{
			Name:     jd.Name,
			Model:    jd.Model,
			Version:  jd.Version,
			SKU:      jd.SKU,
			Capacity: jd.Capacity,
		})
	}

	return deployments
}

func getAzureAIKey(resourceGroup, accountName string) string {
	cmd := exec.Command("az", "cognitiveservices", "account", "keys", "list",
		"--resource-group", resourceGroup,
		"--name", accountName,
		"--query", "key1",
		"-o", "tsv",
	)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func generateAzureAgentScript(dconfig DeployConfig) string {
	return fmt.Sprintf(`/bin/sh -c 'pip install openai && python3 -c "
from openai import AzureOpenAI
import os

client = AzureOpenAI(
    azure_endpoint=os.environ[\"AZURE_OPENAI_ENDPOINT\"],
    api_key=os.environ[\"AZURE_OPENAI_API_KEY\"],
    api_version=\"2024-02-15-preview\"
)

response = client.chat.completions.create(
    model=os.environ[\"AZURE_OPENAI_DEPLOYMENT\"],
    messages=[{\"role\": \"user\", \"content\": \"\"\"%s\"\"\"}]
)

print(response.choices[0].message.content)
"'`, strings.ReplaceAll(dconfig.Prompt, `"`, `\"`))
}

func runDeployAgent() tea.Cmd {
	dc := &deployAgentCmd{}
	return tea.Exec(dc, func(err error) tea.Msg {
		return commandDoneMsg{
			command: "deploy-agent",
			tool:    "skitz",
			success: dc.success,
		}
	})
}

func deployToACIFromPalette(dconfig DeployConfig, apiKey string) (string, error) {
	image := "python:3.11-slim"
	envVars := []string{
		fmt.Sprintf("AZURE_OPENAI_ENDPOINT=%s", dconfig.AIEndpoint),
		fmt.Sprintf("AZURE_OPENAI_API_KEY=%s", apiKey),
		fmt.Sprintf("AZURE_OPENAI_DEPLOYMENT=%s", dconfig.AIDeployment),
		fmt.Sprintf("AGENT_PROMPT=%s", dconfig.Prompt),
	}

	script := generateAzureAgentScript(dconfig)

	args := []string{
		"container", "create",
		"--resource-group", dconfig.ResourceGroup,
		"--name", dconfig.AgentName,
		"--image", image,
		"--restart-policy", "Never",
		"--location", dconfig.Location,
	}

	for _, env := range envVars {
		args = append(args, "--environment-variables", env)
	}

	args = append(args, "--command-line", script)

	aciCmd := exec.Command("az", args...)
	output, err := aciCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("deployment failed: %s", string(output))
	}

	result := fmt.Sprintf("Container '%s' deployed successfully!\n\nResource Group: %s\nLocation: %s\nModel: %s\n\nTo view logs:\naz container logs --resource-group %s --name %s --follow",
		dconfig.AgentName,
		dconfig.ResourceGroup,
		dconfig.Location,
		dconfig.AIModel,
		dconfig.ResourceGroup,
		dconfig.AgentName,
	)

	return result, nil
}

func deployToPipelineFromPalette(dconfig DeployConfig, apiKey string) (string, error) {
	if !checkAzureDevOpsCLI() {
		return "", fmt.Errorf("Azure DevOps CLI extension is required.\n\nInstall with: az extension add --name azure-devops")
	}

	pipelineYAML := fmt.Sprintf(`trigger: none
pool:
  vmImage: ubuntu-latest
steps:
- script: |
    pip install openai
    python3 -c "
    from openai import AzureOpenAI
    client = AzureOpenAI(
        azure_endpoint='%s',
        api_key='%s',
        api_version='2024-02-15-preview'
    )
    response = client.chat.completions.create(
        model='%s',
        messages=[{'role': 'user', 'content': '''%s'''}]
    )
    print(response.choices[0].message.content)
    "
  displayName: 'Run AI Agent'
`, dconfig.AIEndpoint, apiKey, dconfig.AIDeployment, strings.ReplaceAll(dconfig.Prompt, "'", "'\\''"))

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("agent-pipeline-%s.yml", dconfig.AgentName))
	if err := os.WriteFile(tmpFile, []byte(pipelineYAML), 0644); err != nil {
		return "", fmt.Errorf("failed to create pipeline YAML: %v", err)
	}

	result := fmt.Sprintf("Pipeline YAML created: %s\n\nTo deploy this pipeline:\n\n1. Push this YAML to your Azure DevOps repository\n2. Create a new pipeline in Azure DevOps using this YAML\n3. Set up the required service connection for Azure\n\nPipeline configuration:\n- Model: %s\n- Deployment: %s\n- Task: %s",
		tmpFile,
		dconfig.AIModel,
		dconfig.AIDeployment,
		truncateStr(dconfig.Prompt, 60),
	)

	return result, nil
}

func azureAIAccountsTableCommand() string {
	return `az cognitiveservices account list --query "[?kind=='OpenAI' || kind=='CognitiveServices'].{name:name, resourceGroup:resourceGroup, location:location, kind:kind}" -o table`
}

func azureAIDeploymentsTableCommand(resourceGroup, accountName string) string {
	return fmt.Sprintf(`az cognitiveservices account deployment list --resource-group %s --name %s --query "[].{name:name, model:properties.model.name, version:properties.model.version, sku:sku.name, capacity:sku.capacity}" -o table`,
		shellEscape(resourceGroup),
		shellEscape(accountName),
	)
}

func shellEscape(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
