# Command Execution Audit

This documents every command execution path in `cli/skitz`, how it runs, and where behavior differs.

## Summary
- There are multiple execution paths: embedded terminal (PTY + vterm), full terminal control via `tea.Exec`, direct `exec.Command`, and MCP tool calls (no shell).
- Only the embedded terminal path uses the user's shell (`$SHELL`); several other paths hardcode `sh -c` or call binaries directly.
- Command validity is not centrally enforced; some paths use string templates, others build argv lists.
- Testing coverage is partial; no unified unit/integration tests exist for command execution paths.

## Execution Paths

### Embedded Terminal (resource commands + palette)
- Entry: `runCommand(CommandSpec{Mode: CommandEmbedded})` → `executeEmbedded()` in `cli/skitz/exec.go`
- Shell: **uses user shell** (`$SHELL`, fallback `/bin/sh`)
- Behavior: runs via PTY, output rendered with vterm, integrated focus toggle
- Source:
  - `runCommand()` in `cli/skitz/command_runner.go`
  - `executeEmbedded()` in `cli/skitz/exec.go`

### Full Terminal Control (interactive commands)
- Entry: `runCommand(CommandSpec{Mode: CommandInteractive})` → `executeInteractive()` → `tea.Exec`
- Shell: **hardcoded `sh -c`** in `interactiveCmd.Run()`
- Behavior: clears screen, runs command with full terminal control, waits for Enter
- Source:
  - `runCommand()` in `cli/skitz/command_runner.go`
  - `interactiveCmd.Run()` in `cli/skitz/exec.go`
  - `executeInteractive()` in `cli/skitz/exec.go`

### Direct Shell Execution (non-embedded)
- Entry: `executeCommandDirect()` (used by actions)
- Shell: **hardcoded `sh -c`**
- Behavior: direct execution, no embedded output or PTY
- Source:
  - `executeCommandDirect()` in `cli/skitz/exec.go`
  - Quick action handler uses `tea.Exec` and `interactiveCmd` in `cli/skitz/actions.go`

### Azure CLI Calls (deploy flow)
- Entry: `deployAgentCmd.Run()` and helper functions in `cli/skitz/deploy.go`
- Shell: **no shell**; `exec.Command("az", ...)`
- Behavior: direct binary execution, uses `CombinedOutput` / `Output`
- Source:
  - `deployAgentCmd.Run()` in `cli/skitz/deploy.go`
  - `checkAzureCLI()`, `getAzureAIAccounts()`, `getAzureAIDeployments()`, `getAzureAIKey()` in `cli/skitz/deploy.go`

### MCP Tool Calls (BIA + MCP tools)
- Entry: MCP client calls; no shell execution
- Behavior: request/response to MCP server
- Source:
  - `ReviewCodeWithBIA()` in `cli/skitz/agent_mcp.go`
  - `executeMCPToolWithArgs()` in `cli/skitz/palette.go`

### TUI/TTY Helpers (BIA/MCP)
- Uses `exec.Command("stty", ...)` to restore terminal state
- Source:
  - `cli/skitz/agent_mcp.go`

### External Editor Launch (edit action)
- Entry: `actionEditFile()` uses `exec.Command(editor, filePath)`
- Shell: **no shell**; direct exec of editor binary
- Source:
  - `cli/skitz/actions.go`

## Differences and Inconsistencies

### Shell Context
- Uses user shell:
  - `executeEmbedded()` (PTY/vterm)
- Hardcoded `sh -c`:
  - `interactiveCmd.Run()` (tea.Exec path)
  - `executeCommandDirect()`
- Direct binary execution:
  - Azure CLI in deploy flow
  - `stty` calls in BIA/MCP
  - `actionEditFile()` editor launch

### Output Handling
- Embedded terminal output rendered via vterm in both resource view and palette.
- `tea.Exec` paths render in the real terminal (not embedded).
- Direct `exec.Command` paths use raw stdout/stderr or `CombinedOutput`.

### Command Validation
- Markdown `^run` commands are passed through with minimal validation.
- Deploy flow uses specific `az` argv lists; validity depends on Azure CLI availability.
- Palette now uses embedded terminal for list commands but does not verify command availability beyond error text.

## Testing Status (Current)
- No centralized test suite for command execution paths.
- Existing tests include:
  - `TestCheckAzureCLI` in `cli/skitz/agent_test.go` (only checks function runs)
- No integration tests verifying:
  - Shell selection (`$SHELL`) for interactive/embedded paths
  - Command rendering in palette/resource views
  - Deploy flow commands
  - MCP tool execution paths

## Open Questions / Decisions Needed
- Should all command execution paths use `$SHELL` consistently?
- Should Azure CLI calls remain direct exec or be funneled through the embedded terminal path?
- Should command validity be enforced at parse time (`^run`) or at execution time?
- What is the acceptable strategy for integration tests that require external CLIs (Azure, Docker, etc.)?

## Files Referenced
- `cli/skitz/exec.go`
- `cli/skitz/actions.go`
- `cli/skitz/deploy.go`
- `cli/skitz/palette.go`
- `cli/skitz/agent_mcp.go`
