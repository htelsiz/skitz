# skitz

A personal command reference TUI built with the [Charm](https://charm.sh) stack. Browse, search, and execute commands across multiple tools (Docker, Git, Claude, Azure, Cursor, MCP) with rich markdown rendering and AI-powered code review via MCP.

## Highlights

- **Interactive TUI**: Dashboard with resource cards, drill-down navigation, command palette (Ctrl+K)
- **Markdown-powered**: Resources defined in markdown, rendered with Glamour styling
- **Command execution**: Run annotated commands with `^run` tags and input prompts
- **MCP integration**: Connect to Model Context Protocol servers for AI agent tools
- **BIA code review**: Built-in code review via MCP-connected BIA Junior Agent
- **Azure deployment**: Interactive wizard for deploying agents to Azure

---

## Quickstart

### Build & Install

```bash
cd ~/projects/skitz
go build -o skitz
ln -sf $(pwd)/skitz ~/.local/bin/skitz
```

### Run

```bash
skitz              # Interactive dashboard mode
skitz docker       # Open specific resource
skitz docker -i    # Browse detailed sections
```

---

## Configuration

Config file: `~/.config/skitz/config.yaml`

```yaml
version: 1

quick_actions:
  - key: "code_review"
    label: "BIA Code Review"
    action: "bia_code_review"

history:
  max_items: 100
  enabled: true

favorites:
  - "docker:containers:ps"
  - "git:basics:status"

mcp_servers:
  - name: "bldrspec-ai"
    url: "http://localhost:8001/mcp/"
    refresh_interval: 60
```

**Environment variable**: `BLDRSPEC_MCP_URL` overrides the default MCP server URL.

---

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

---

## Tools Overview

### Command Execution

Commands can be annotated for execution:

```markdown
`docker ps -a` list all containers ^run
`docker exec -it {{container}} bash` shell into container ^run:container
```

- `^run` — execute as-is with confirmation
- `^run:varname` — prompt for input before execution

### MCP Client

Connect to MCP servers for AI agent capabilities:

- Tool listing and invocation
- Server status monitoring (shown in dashboard)
- Multi-server support

### BIA Junior Agent

Code review via MCP (Ctrl+K → "BIA Code Review"):

- File path input or paste mode
- Markdown-rendered feedback
- Interaction history tracking

### Deploy Agent

Azure deployment wizard:

- Azure CLI integration detection
- Supports Claude, Cursor, or custom agents
- Deployment targets: ACI or Azure Pipelines

---

## Resource Files

Resources live in `~/.config/skitz/resources/`:

```
resources/
  docker.md         # compact reference
  docker-detail.md  # detailed sections
  git.md
  git-detail.md
  azure.md
  claude.md
  cursor.md
  mcp.md
```

### Compact Format

One-liner style, scannable at a glance:

```markdown
# Docker

`docker ps -a` list all containers ^run
`docker exec -it {{c}} bash` shell into container ^run:c
`docker logs -f {{c}}` follow logs ^run:c
```

### Detail Format

Organized by `## Section` headers for interactive browsing:

```markdown
# Docker Detail

## Containers

docker ps                    # running containers
docker ps -a                 # all containers

## Images

...
```

---

## Development

```bash
# Build
go build -o skitz

# Run tests
go test ./...

# Run with verbose output
go test -v ./...

# Integration tests (requires MCP server)
MCP_TEST_ENABLED=1 go test ./...

# Single test
go test -run TestFunctionName
```

### Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| BubbleTea | v1.3.10 | TUI framework (Elm-inspired) |
| Glamour | v0.8.0 | Markdown rendering |
| Lipgloss | v1.1.0 | Terminal styling |
| Huh | v0.8.0 | Interactive forms |
| MCP-Go | v0.43.2 | MCP client |
| Harmonica | v0.2.0 | Spring physics animations |

### Data Storage

- **Config**: `~/.config/skitz/config.yaml`
- **History**: `~/.local/share/skitz/history.json`
- **Agent History**: `~/.local/share/skitz/agent_history.json`

---

## MCP Client Configuration

Connect Claude Desktop or other MCP clients:

```json
{
  "mcpServers": {
    "bldrspec-ai": {
      "transport": {
        "type": "http",
        "url": "http://localhost:8001/mcp/"
      }
    }
  }
}
```

---

## Documentation

See the [docs/](./docs/) directory for detailed documentation:

- [MCP Client](./docs/mcp-client.md) - MCP integration and API reference
- [Roadmap](./docs/roadmap.md) - Planned features

**Completed:**
- Interactive navigation with breadcrumbs
- Fuzzy search within resources
- Command execution with confirmation
- MCP client integration
- BIA code review

**In Progress:**
- Copy to clipboard
- Recently used tracking
- Custom themes

---

## Contributing

PRs welcome. Run tests before submitting:

```bash
go test ./...
```

## License

MIT
