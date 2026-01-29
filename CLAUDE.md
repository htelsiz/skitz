# Skitz

Terminal UI command center with MCP (Model Context Protocol) integration.

## Stack

- **Go 1.25+**
- **Charm**: BubbleTea, Lipgloss, Glamour, Huh
- **MCP-Go**: AI agent integration via Model Context Protocol

## Structure

```
internal/
├── app/        # TUI application (model, views, keyboard, wizards)
├── config/     # YAML configuration and history
├── mcp/        # MCP client for AI server communication
└── resources/  # Embedded markdown command references
```

## Commands

```bash
go build -o skitz          # Build
go test ./...              # Run tests
./skitz                    # Run (or ./skitz <resource>)
```

## Data

- Config: `~/.config/skitz/config.yaml`
- History: `~/.local/share/skitz/history.json`
- Resources: `~/.config/skitz/resources/*.md`

See [AGENTS.md](AGENTS.md) for architecture and code style.
