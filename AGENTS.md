# AGENTS.md

## Architecture

Skitz is a terminal UI command reference system built with the **Charm** stack in Go:

- **BubbleTea** (v1.3.10) - Elm-inspired TUI framework for state management
- **Glamour** (v0.8.0) - Markdown rendering with custom styling
- **Lipgloss** (v1.1.0) - Terminal styling and layout
- **Huh** (v0.8.0) - Interactive forms and prompts
- **MCP-Go** (v0.43.2) - Model Context Protocol client for AI agent integration

### Core Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point; initializes app and loads resources |
| `model.go` | BubbleTea model; core state management |
| `views.go` | View rendering (dashboard and detail views) |
| `actions.go` | Quick actions and built-in action handlers |
| `palette.go` | Command palette implementation (Ctrl+K) |
| `config.go` | YAML-based configuration management |
| `types.go` | Core data types and metadata definitions |
| `exec.go` | Command execution with confirmation flow |
| `mcp_client.go` | MCP client for AI server communication |
| `agent_mcp.go` | BIA Junior Agent code review integration |
| `deploy.go` | Azure deployment wizard |
| `styles.go` | UI styling and color schemes |
| `notifications.go` | Toast notification system |

### Data Storage

- **Config**: `~/.config/skitz/config.yaml`
- **History**: `~/.local/share/skitz/history.json`
- **Agent History**: `~/.local/share/skitz/agent_history.json`
- **Resources**: `~/dotfiles/resources/*.md`

## Essential Commands

**Build**: `go build -o skitz`
**Run**: `./skitz` or `./skitz <resource>`
**All tests**: `go test ./...`
**Verbose tests**: `go test -v ./...`
**Integration tests**: `MCP_TEST_ENABLED=1 go test ./...`
**Single test**: `go test -run TestFunctionName`

## Key Components

### MCP Client (`mcp_client.go`)

HTTP-based client for Model Context Protocol servers:
- Connection management with context timeouts (5s default)
- Tool listing and invocation
- Server info and capabilities retrieval
- Multi-server status monitoring

**Environment**: `BLDRSPEC_MCP_URL` (default: `http://localhost:8001/mcp/`)

### BIA Junior Agent (`agent_mcp.go`)

Code review integration via MCP:
- Invokes `bia_junior_agent` tool for code reviews
- Supports file path input or paste mode
- Renders feedback using Glamour markdown
- Accessed via Command Palette (Ctrl+K) → "BIA Code Review"

### Deploy Agent (`deploy.go`)

Azure deployment wizard:
- Interactive configuration flow
- Azure CLI integration detection
- Supports Claude, Cursor, or custom agents
- Deployment targets: Azure Container Instances (ACI) or Azure Pipelines

### Command Execution (`exec.go`)

- Interactive execution with confirmation
- Parameter substitution via `{{INPUT}}` placeholders
- History tracking with persistence
- Annotated commands with `^run` and `^run:varname` tags

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q` / `Esc` | Exit or back to dashboard |
| `Enter` | Select/execute item |
| `↑/↓` or `k/j` | Navigate up/down |
| `←/→` or `h/l` | Switch sections |
| `Tab` / `Shift+Tab` | Cycle through sections |
| `1-9` | Jump to section by number |
| `Ctrl+K` | Open command palette |
| `Ctrl+D` / `Ctrl+U` | Page down/up |
| `g` / `G` | Go to top/bottom |

## Code Style Guidelines

- **Go version**: 1.25+
- **Package**: Single `main` package with clear file separation
- **Naming**: Camel case; exported functions start uppercase
- **Error handling**: Check errors with `if err != nil`; graceful degradation preferred
- **Concurrency**: Use goroutines with channels for async operations
- **Testing**: Tests in same package with `_test.go` suffix
- **Env gating**: Use environment variables for integration tests (`MCP_TEST_ENABLED`)

### Patterns Used

- **Builders**: Complex object construction (config, styles)
- **Map-based lookups**: Metadata and action handlers
- **Elm architecture**: BubbleTea's Update/View/Init pattern
- **Spring physics**: Harmonica for quote animations

## Available Resources

Auto-discovered from `~/dotfiles/resources/` and exposed in the dashboard:

- `azure` - Azure CLI commands
- `claude` - Claude AI assistant
- `cursor` - Cursor editor shortcuts
- `docker` - Docker commands
- `git` - Git commands
- `mcp` - Model Context Protocol reference

Each resource has:
- Compact reference (`{resource}.md`)
- Detailed sections (`{resource}-detail.md`)

## External Integration Reference

- **MCP (Model Context Protocol)**: https://modelcontextprotocol.io/
- **BubbleTea**: https://github.com/charmbracelet/bubbletea
- **Glamour**: https://github.com/charmbracelet/glamour
- **MCP-Go**: https://github.com/mark3labs/mcp-go
