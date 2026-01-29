package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// CloudProvider represents a supported cloud provider
type CloudProvider string

const (
	CloudAzure CloudProvider = "azure"
	CloudAWS   CloudProvider = "aws"
	CloudGCP   CloudProvider = "gcp"
)

// CloudAction represents an action type for cloud agents
type CloudAction string

const (
	CloudActionRunAgent    CloudAction = "run_agent"
	CloudActionExecuteTask CloudAction = "execute_task"
	CloudActionListAgents  CloudAction = "list_agents"
	CloudActionStopAgent   CloudAction = "stop_agent"
)

// CloudAgentConfig holds configuration for cloud agent actions
type CloudAgentConfig struct {
	Provider      CloudProvider
	Action        CloudAction
	AgentName     string
	ResourceGroup string
	Region        string
	Prompt        string
	Model         string
	Timeout       int
}

// CloudAgentWizard holds state for the Cloud Agent wizard
type CloudAgentWizard struct {
	Step      int
	Data      map[string]any
	InputForm *huh.Form
}

// CloudResource represents a cloud resource (agent, container, etc.)
type CloudResource struct {
	Name          string
	Provider      CloudProvider
	ResourceGroup string
	Region        string
	Status        string
	CreatedAt     string
}

// cloudAgentWizardMsg messages for wizard state updates
type cloudAgentResourcesMsg struct {
	resources []CloudResource
	err       error
}

type cloudAgentResultMsg struct {
	title  string
	output string
	err    error
}

func (m *model) startCloudAgentWizard() tea.Cmd {
	m.palette.WizardState = &wizardState{
		Type: "cloud_agent",
		Step: 0,
		Data: make(map[string]any),
	}
	m.term.staticOutput = ""
	m.term.staticTitle = ""
	return m.nextCloudAgentStep()
}

func (m *model) nextCloudAgentStep() tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil {
		return nil
	}

	switch ws.Step {
	case 0:
		// Step 0: Select cloud provider
		var provider string
		select_ := huh.NewSelect[string]().
			Title("Select Cloud Provider").
			Description("Choose your cloud platform").
			Options(
				huh.NewOption("Azure (ACI, Azure AI)", string(CloudAzure)),
				huh.NewOption("AWS (Lambda, Bedrock)", string(CloudAWS)),
				huh.NewOption("GCP (Cloud Run, Vertex AI)", string(CloudGCP)),
			).
			Value(&provider)

		m.palette.InputForm = huh.NewForm(huh.NewGroup(select_)).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(false).
			WithTheme(huh.ThemeCatppuccin())

		m.palette.State = PaletteStateCollectingParams
		ws.Data["selected_provider"] = &provider

		return m.palette.InputForm.Init()

	case 1:
		// Step 1: Select action type
		provider := *ws.Data["selected_provider"].(*string)
		ws.Data["provider"] = provider

		var action string
		actionOptions := getProviderActions(CloudProvider(provider))

		select_ := huh.NewSelect[string]().
			Title("Select Action").
			Description(fmt.Sprintf("Available actions for %s", provider)).
			Options(actionOptions...).
			Value(&action)

		m.palette.InputForm = huh.NewForm(huh.NewGroup(select_)).
			WithWidth(80).
			WithShowHelp(false).
			WithShowErrors(false).
			WithTheme(huh.ThemeCatppuccin())

		m.palette.State = PaletteStateCollectingParams
		ws.Data["selected_action"] = &action

		return m.palette.InputForm.Init()

	case 2:
		// Step 2: Configure action parameters based on selected action
		action := *ws.Data["selected_action"].(*string)
		ws.Data["action"] = action
		provider := CloudProvider(ws.Data["provider"].(string))

		return m.buildCloudAgentParameterForm(provider, CloudAction(action))

	case 3:
		// Step 3: Execute the action
		m.palette.State = PaletteStateExecuting
		m.palette.LoadingText = "Executing cloud action..."
		m.palette.InputForm = nil

		provider := CloudProvider(ws.Data["provider"].(string))
		action := CloudAction(ws.Data["action"].(string))

		return m.executeCloudAction(provider, action)
	}

	return nil
}

func getProviderActions(provider CloudProvider) []huh.Option[string] {
	baseActions := []huh.Option[string]{
		huh.NewOption("Run Agent - Deploy and run an AI agent", string(CloudActionRunAgent)),
		huh.NewOption("Execute Task - Run a one-time task", string(CloudActionExecuteTask)),
		huh.NewOption("List Agents - View running agents", string(CloudActionListAgents)),
		huh.NewOption("Stop Agent - Terminate a running agent", string(CloudActionStopAgent)),
	}

	switch provider {
	case CloudAzure:
		return baseActions
	case CloudAWS:
		return baseActions
	case CloudGCP:
		return baseActions
	default:
		return baseActions
	}
}

func (m *model) buildCloudAgentParameterForm(provider CloudProvider, action CloudAction) tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil {
		return nil
	}

	var fields []huh.Field

	switch action {
	case CloudActionRunAgent:
		var agentName, prompt, model string
		var region string

		fields = append(fields,
			huh.NewInput().
				Title("Agent Name").
				Description("Name for your agent instance").
				Placeholder("my-agent").
				Value(&agentName),
		)
		ws.Data["agent_name"] = &agentName

		fields = append(fields,
			huh.NewSelect[string]().
				Title("Region").
				Description("Deployment region").
				Options(getProviderRegions(provider)...).
				Value(&region),
		)
		ws.Data["region"] = &region

		fields = append(fields,
			huh.NewSelect[string]().
				Title("Model").
				Description("AI model to use").
				Options(getProviderModels(provider)...).
				Value(&model),
		)
		ws.Data["model"] = &model

		fields = append(fields,
			huh.NewText().
				Title("Task Prompt").
				Description("Instructions for the agent").
				Placeholder("Analyze the data and provide insights...").
				CharLimit(2000).
				Value(&prompt),
		)
		ws.Data["prompt"] = &prompt

	case CloudActionExecuteTask:
		var taskCmd, region string

		fields = append(fields,
			huh.NewSelect[string]().
				Title("Region").
				Options(getProviderRegions(provider)...).
				Value(&region),
		)
		ws.Data["region"] = &region

		fields = append(fields,
			huh.NewText().
				Title("Task Command").
				Description("Command or script to execute").
				Placeholder("python script.py --analyze").
				CharLimit(5000).
				Value(&taskCmd),
		)
		ws.Data["task_cmd"] = &taskCmd

	case CloudActionListAgents:
		var region string
		fields = append(fields,
			huh.NewSelect[string]().
				Title("Region").
				Description("Filter by region (optional)").
				Options(append([]huh.Option[string]{huh.NewOption("All Regions", "all")}, getProviderRegions(provider)...)...).
				Value(&region),
		)
		ws.Data["region"] = &region

	case CloudActionStopAgent:
		var agentName string
		fields = append(fields,
			huh.NewInput().
				Title("Agent Name").
				Description("Name of the agent to stop").
				Placeholder("agent-1234567890").
				Value(&agentName),
		)
		ws.Data["agent_name"] = &agentName
	}

	if len(fields) == 0 {
		ws.Step++
		return m.nextCloudAgentStep()
	}

	m.palette.InputForm = huh.NewForm(huh.NewGroup(fields...)).
		WithWidth(80).
		WithShowHelp(true).
		WithShowErrors(true).
		WithTheme(huh.ThemeCatppuccin())

	m.palette.State = PaletteStateCollectingParams
	return m.palette.InputForm.Init()
}

func getProviderRegions(provider CloudProvider) []huh.Option[string] {
	switch provider {
	case CloudAzure:
		return []huh.Option[string]{
			huh.NewOption("East US", "eastus"),
			huh.NewOption("West US 2", "westus2"),
			huh.NewOption("West Europe", "westeurope"),
			huh.NewOption("North Europe", "northeurope"),
			huh.NewOption("Southeast Asia", "southeastasia"),
		}
	case CloudAWS:
		return []huh.Option[string]{
			huh.NewOption("US East (N. Virginia)", "us-east-1"),
			huh.NewOption("US West (Oregon)", "us-west-2"),
			huh.NewOption("EU (Ireland)", "eu-west-1"),
			huh.NewOption("EU (Frankfurt)", "eu-central-1"),
			huh.NewOption("Asia Pacific (Singapore)", "ap-southeast-1"),
		}
	case CloudGCP:
		return []huh.Option[string]{
			huh.NewOption("US Central (Iowa)", "us-central1"),
			huh.NewOption("US East (S. Carolina)", "us-east1"),
			huh.NewOption("Europe West (Belgium)", "europe-west1"),
			huh.NewOption("Asia East (Taiwan)", "asia-east1"),
		}
	default:
		return []huh.Option[string]{}
	}
}

func getProviderModels(provider CloudProvider) []huh.Option[string] {
	switch provider {
	case CloudAzure:
		return []huh.Option[string]{
			huh.NewOption("GPT-4o", "gpt-4o"),
			huh.NewOption("GPT-4o Mini", "gpt-4o-mini"),
			huh.NewOption("GPT-4 Turbo", "gpt-4-turbo"),
		}
	case CloudAWS:
		return []huh.Option[string]{
			huh.NewOption("Claude 3.5 Sonnet", "anthropic.claude-3-5-sonnet-20240620-v1:0"),
			huh.NewOption("Claude 3 Haiku", "anthropic.claude-3-haiku-20240307-v1:0"),
			huh.NewOption("Titan Text Express", "amazon.titan-text-express-v1"),
		}
	case CloudGCP:
		return []huh.Option[string]{
			huh.NewOption("Gemini 1.5 Pro", "gemini-1.5-pro"),
			huh.NewOption("Gemini 1.5 Flash", "gemini-1.5-flash"),
			huh.NewOption("PaLM 2", "text-bison"),
		}
	default:
		return []huh.Option[string]{}
	}
}

func (m *model) executeCloudAction(provider CloudProvider, action CloudAction) tea.Cmd {
	return func() tea.Msg {
		ws := m.palette.WizardState
		if ws == nil {
			return cloudAgentResultMsg{
				title:  "Cloud Agent",
				output: "Error: Wizard state not found",
				err:    fmt.Errorf("wizard state is nil"),
			}
		}

		switch action {
		case CloudActionRunAgent:
			return m.executeRunAgent(provider, ws)
		case CloudActionExecuteTask:
			return m.executeTask(provider, ws)
		case CloudActionListAgents:
			return m.listAgents(provider, ws)
		case CloudActionStopAgent:
			return m.stopAgent(provider, ws)
		default:
			return cloudAgentResultMsg{
				title:  "Cloud Agent",
				output: "Unknown action",
				err:    fmt.Errorf("unknown action: %s", action),
			}
		}
	}
}

func (m *model) executeRunAgent(provider CloudProvider, ws *wizardState) tea.Msg {
	agentName := getStringFromData(ws.Data, "agent_name")
	region := getStringFromData(ws.Data, "region")
	model := getStringFromData(ws.Data, "model")
	prompt := getStringFromData(ws.Data, "prompt")

	if agentName == "" {
		agentName = fmt.Sprintf("agent-%d", time.Now().Unix())
	}

	switch provider {
	case CloudAzure:
		return runAzureAgent(agentName, region, model, prompt)
	case CloudAWS:
		return runAWSAgent(agentName, region, model, prompt)
	case CloudGCP:
		return runGCPAgent(agentName, region, model, prompt)
	default:
		return cloudAgentResultMsg{
			title:  "Cloud Agent",
			output: fmt.Sprintf("Provider %s not yet supported", provider),
			err:    fmt.Errorf("unsupported provider"),
		}
	}
}

func (m *model) executeTask(provider CloudProvider, ws *wizardState) tea.Msg {
	region := getStringFromData(ws.Data, "region")
	taskCmd := getStringFromData(ws.Data, "task_cmd")

	if taskCmd == "" {
		return cloudAgentResultMsg{
			title:  "Cloud Agent",
			output: "No task command provided",
			err:    fmt.Errorf("empty task command"),
		}
	}

	switch provider {
	case CloudAzure:
		return executeAzureTask(region, taskCmd)
	case CloudAWS:
		return executeAWSTask(region, taskCmd)
	case CloudGCP:
		return executeGCPTask(region, taskCmd)
	default:
		return cloudAgentResultMsg{
			title:  "Cloud Agent",
			output: fmt.Sprintf("Provider %s not yet supported", provider),
			err:    fmt.Errorf("unsupported provider"),
		}
	}
}

func (m *model) listAgents(provider CloudProvider, ws *wizardState) tea.Msg {
	region := getStringFromData(ws.Data, "region")

	switch provider {
	case CloudAzure:
		return listAzureAgents(region)
	case CloudAWS:
		return listAWSAgents(region)
	case CloudGCP:
		return listGCPAgents(region)
	default:
		return cloudAgentResultMsg{
			title:  "Cloud Agent",
			output: fmt.Sprintf("Provider %s not yet supported", provider),
			err:    fmt.Errorf("unsupported provider"),
		}
	}
}

func (m *model) stopAgent(provider CloudProvider, ws *wizardState) tea.Msg {
	agentName := getStringFromData(ws.Data, "agent_name")

	if agentName == "" {
		return cloudAgentResultMsg{
			title:  "Cloud Agent",
			output: "No agent name provided",
			err:    fmt.Errorf("empty agent name"),
		}
	}

	switch provider {
	case CloudAzure:
		return stopAzureAgent(agentName)
	case CloudAWS:
		return stopAWSAgent(agentName)
	case CloudGCP:
		return stopGCPAgent(agentName)
	default:
		return cloudAgentResultMsg{
			title:  "Cloud Agent",
			output: fmt.Sprintf("Provider %s not yet supported", provider),
			err:    fmt.Errorf("unsupported provider"),
		}
	}
}

func getStringFromData(data map[string]any, key string) string {
	if ptr, ok := data[key].(*string); ok && ptr != nil {
		return *ptr
	}
	if s, ok := data[key].(string); ok {
		return s
	}
	return ""
}

// Azure implementations
func runAzureAgent(name, region, model, prompt string) tea.Msg {
	if !checkAzureCLI() {
		return cloudAgentResultMsg{
			title:  "Azure Agent",
			output: "Azure CLI is required but not installed.\n\nInstall from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli",
			err:    fmt.Errorf("az cli not found"),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	containerName := name
	image := "python:3.11-slim"

	script := fmt.Sprintf(`pip install openai && python3 -c "
import os
from openai import AzureOpenAI

client = AzureOpenAI(
    azure_endpoint=os.environ.get('AZURE_OPENAI_ENDPOINT', ''),
    api_key=os.environ.get('AZURE_OPENAI_API_KEY', ''),
    api_version='2024-02-15-preview'
)

response = client.chat.completions.create(
    model='%s',
    messages=[{'role': 'user', 'content': '''%s'''}]
)
print(response.choices[0].message.content)
"`, model, strings.ReplaceAll(prompt, "'", "\\'"))

	cmd := exec.CommandContext(ctx, "az", "container", "create",
		"--resource-group", "skitz-agents",
		"--name", containerName,
		"--image", image,
		"--restart-policy", "Never",
		"--location", region,
		"--command-line", fmt.Sprintf("/bin/sh -c '%s'", script),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "ResourceGroupNotFound") {
			return cloudAgentResultMsg{
				title: "Azure Agent",
				output: fmt.Sprintf(`Resource group 'skitz-agents' not found.

Create it first with:
%s

Then try again.`, "az group create --name skitz-agents --location "+region),
				err: err,
			}
		}
		return cloudAgentResultMsg{
			title:  "Azure Agent",
			output: fmt.Sprintf("Failed to create container:\n%s", string(output)),
			err:    err,
		}
	}

	return cloudAgentResultMsg{
		title: "Azure Agent - Deployed!",
		output: fmt.Sprintf(`Agent '%s' deployed successfully!

**Region:** %s
**Model:** %s
**Container:** %s

View logs:
%s

Stop agent:
%s`,
			name, region, model, containerName,
			"`az container logs --resource-group skitz-agents --name "+containerName+" --follow`",
			"`az container delete --resource-group skitz-agents --name "+containerName+" -y`",
		),
		err: nil,
	}
}

func listAzureAgents(region string) tea.Msg {
	if !checkAzureCLI() {
		return cloudAgentResultMsg{
			title:  "Azure Agents",
			output: "Azure CLI is required but not installed.",
			err:    fmt.Errorf("az cli not found"),
		}
	}

	args := []string{"container", "list", "--resource-group", "skitz-agents", "-o", "json"}
	cmd := exec.Command("az", args...)
	output, err := cmd.Output()
	if err != nil {
		return cloudAgentResultMsg{
			title:  "Azure Agents",
			output: "Failed to list agents. Resource group may not exist.",
			err:    err,
		}
	}

	var containers []struct {
		Name              string `json:"name"`
		Location          string `json:"location"`
		ProvisioningState string `json:"provisioningState"`
	}
	if err := json.Unmarshal(output, &containers); err != nil {
		return cloudAgentResultMsg{
			title:  "Azure Agents",
			output: "Failed to parse agent list",
			err:    err,
		}
	}

	if len(containers) == 0 {
		return cloudAgentResultMsg{
			title:  "Azure Agents",
			output: "No agents running in resource group 'skitz-agents'",
			err:    nil,
		}
	}

	var sb strings.Builder
	sb.WriteString("| Name | Region | Status |\n")
	sb.WriteString("|------|--------|--------|\n")
	for _, c := range containers {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", c.Name, c.Location, c.ProvisioningState))
	}

	return cloudAgentResultMsg{
		title:  fmt.Sprintf("Azure Agents (%d)", len(containers)),
		output: sb.String(),
		err:    nil,
	}
}

func stopAzureAgent(name string) tea.Msg {
	if !checkAzureCLI() {
		return cloudAgentResultMsg{
			title:  "Stop Azure Agent",
			output: "Azure CLI is required but not installed.",
			err:    fmt.Errorf("az cli not found"),
		}
	}

	cmd := exec.Command("az", "container", "delete",
		"--resource-group", "skitz-agents",
		"--name", name,
		"--yes",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return cloudAgentResultMsg{
			title:  "Stop Azure Agent",
			output: fmt.Sprintf("Failed to stop agent:\n%s", string(output)),
			err:    err,
		}
	}

	return cloudAgentResultMsg{
		title:  "Agent Stopped",
		output: fmt.Sprintf("Agent '%s' has been stopped and deleted.", name),
		err:    nil,
	}
}

func executeAzureTask(region, taskCmd string) tea.Msg {
	return cloudAgentResultMsg{
		title: "Azure Task",
		output: fmt.Sprintf(`Task execution would run in Azure Container Instances.

**Region:** %s
**Command:** %s

To execute manually:
%s`,
			region, taskCmd,
			"`az container create --resource-group skitz-agents --name task-$(date +%%s) --image python:3.11-slim --command-line \""+taskCmd+"\" --restart-policy Never`",
		),
		err: nil,
	}
}

// AWS implementations
func runAWSAgent(name, region, model, prompt string) tea.Msg {
	if !checkAWSCLI() {
		return cloudAgentResultMsg{
			title:  "AWS Agent",
			output: "AWS CLI is required but not installed.\n\nInstall from: https://aws.amazon.com/cli/",
			err:    fmt.Errorf("aws cli not found"),
		}
	}

	return cloudAgentResultMsg{
		title: "AWS Agent",
		output: fmt.Sprintf(`AWS Bedrock agent deployment prepared.

**Agent:** %s
**Region:** %s
**Model:** %s

To deploy using AWS Lambda + Bedrock:
%s

Or use ECS for longer-running tasks:
%s`,
			name, region, model,
			"`aws lambda create-function --function-name "+name+" --runtime python3.11 --handler lambda.handler --role arn:aws:iam::ACCOUNT:role/lambda-bedrock-role`",
			"`aws ecs run-task --cluster agents --task-definition bedrock-agent`",
		),
		err: nil,
	}
}

func listAWSAgents(region string) tea.Msg {
	if !checkAWSCLI() {
		return cloudAgentResultMsg{
			title:  "AWS Agents",
			output: "AWS CLI is required but not installed.",
			err:    fmt.Errorf("aws cli not found"),
		}
	}

	args := []string{"lambda", "list-functions", "--query", "Functions[?starts_with(FunctionName, 'agent-')]", "--output", "json"}
	if region != "all" && region != "" {
		args = append(args, "--region", region)
	}

	cmd := exec.Command("aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return cloudAgentResultMsg{
			title:  "AWS Agents",
			output: "Failed to list Lambda functions. Check your AWS credentials.",
			err:    err,
		}
	}

	var functions []struct {
		FunctionName string `json:"FunctionName"`
		Runtime      string `json:"Runtime"`
		LastModified string `json:"LastModified"`
	}
	if err := json.Unmarshal(output, &functions); err != nil || len(functions) == 0 {
		return cloudAgentResultMsg{
			title:  "AWS Agents",
			output: "No agent functions found (looking for functions starting with 'agent-')",
			err:    nil,
		}
	}

	var sb strings.Builder
	sb.WriteString("| Function | Runtime | Last Modified |\n")
	sb.WriteString("|----------|---------|---------------|\n")
	for _, f := range functions {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", f.FunctionName, f.Runtime, f.LastModified))
	}

	return cloudAgentResultMsg{
		title:  fmt.Sprintf("AWS Agents (%d)", len(functions)),
		output: sb.String(),
		err:    nil,
	}
}

func stopAWSAgent(name string) tea.Msg {
	return cloudAgentResultMsg{
		title: "Stop AWS Agent",
		output: fmt.Sprintf(`To delete Lambda function '%s':

%s

To stop an ECS task:
%s`,
			name,
			"`aws lambda delete-function --function-name "+name+"`",
			"`aws ecs stop-task --cluster agents --task <task-arn>`",
		),
		err: nil,
	}
}

func executeAWSTask(region, taskCmd string) tea.Msg {
	return cloudAgentResultMsg{
		title: "AWS Task",
		output: fmt.Sprintf(`AWS task execution prepared.

**Region:** %s
**Command:** %s

Execute via Lambda:
%s`,
			region, taskCmd,
			"`aws lambda invoke --function-name task-runner --payload '{\"cmd\":\""+taskCmd+"\"}' response.json`",
		),
		err: nil,
	}
}

// GCP implementations
func runGCPAgent(name, region, model, prompt string) tea.Msg {
	if !checkGCloudCLI() {
		return cloudAgentResultMsg{
			title:  "GCP Agent",
			output: "Google Cloud SDK is required but not installed.\n\nInstall from: https://cloud.google.com/sdk/docs/install",
			err:    fmt.Errorf("gcloud cli not found"),
		}
	}

	return cloudAgentResultMsg{
		title: "GCP Agent",
		output: fmt.Sprintf(`GCP Vertex AI agent deployment prepared.

**Agent:** %s
**Region:** %s
**Model:** %s

To deploy using Cloud Run:
%s

To invoke Vertex AI directly:
%s`,
			name, region, model,
			"`gcloud run deploy "+name+" --image python:3.11-slim --region "+region+" --allow-unauthenticated`",
			"`gcloud ai models predict --model="+model+" --region="+region+"`",
		),
		err: nil,
	}
}

func listGCPAgents(region string) tea.Msg {
	if !checkGCloudCLI() {
		return cloudAgentResultMsg{
			title:  "GCP Agents",
			output: "Google Cloud SDK is required but not installed.",
			err:    fmt.Errorf("gcloud cli not found"),
		}
	}

	args := []string{"run", "services", "list", "--format=json"}
	if region != "all" && region != "" {
		args = append(args, "--region="+region)
	}

	cmd := exec.Command("gcloud", args...)
	output, err := cmd.Output()
	if err != nil {
		return cloudAgentResultMsg{
			title:  "GCP Agents",
			output: "Failed to list Cloud Run services. Check your GCP credentials.",
			err:    err,
		}
	}

	var services []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			URL string `json:"url"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &services); err != nil || len(services) == 0 {
		return cloudAgentResultMsg{
			title:  "GCP Agents",
			output: "No Cloud Run services found",
			err:    nil,
		}
	}

	var sb strings.Builder
	sb.WriteString("| Service | URL |\n")
	sb.WriteString("|---------|-----|\n")
	for _, s := range services {
		sb.WriteString(fmt.Sprintf("| %s | %s |\n", s.Metadata.Name, s.Status.URL))
	}

	return cloudAgentResultMsg{
		title:  fmt.Sprintf("GCP Services (%d)", len(services)),
		output: sb.String(),
		err:    nil,
	}
}

func stopGCPAgent(name string) tea.Msg {
	return cloudAgentResultMsg{
		title: "Stop GCP Agent",
		output: fmt.Sprintf(`To delete Cloud Run service '%s':

%s`,
			name,
			"`gcloud run services delete "+name+" --quiet`",
		),
		err: nil,
	}
}

func executeGCPTask(region, taskCmd string) tea.Msg {
	return cloudAgentResultMsg{
		title: "GCP Task",
		output: fmt.Sprintf(`GCP task execution prepared.

**Region:** %s
**Command:** %s

Execute via Cloud Run Jobs:
%s`,
			region, taskCmd,
			"`gcloud run jobs create task-$(date +%%s) --image python:3.11-slim --command \""+taskCmd+"\" --region "+region+" --execute-now`",
		),
		err: nil,
	}
}

// CLI check functions
func checkAWSCLI() bool {
	cmd := exec.Command("aws", "--version")
	return cmd.Run() == nil
}

func checkGCloudCLI() bool {
	cmd := exec.Command("gcloud", "--version")
	return cmd.Run() == nil
}

// handleCloudAgentWizardSubmit processes form submission
func (m *model) handleCloudAgentWizardSubmit() tea.Cmd {
	ws := m.palette.WizardState
	if ws == nil || ws.Type != "cloud_agent" {
		return nil
	}

	ws.Step++
	return m.nextCloudAgentStep()
}
