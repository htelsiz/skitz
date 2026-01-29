# skitz

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> A terminal command center with AI assistance and MCP integration, built with [Charm](https://charm.sh).

<img src="docs/demo.gif" alt="skitz demo" width="700">

## Quick Start

```bash
curl -fsSL https://raw.githubusercontent.com/htelsiz/skitz/main/install.sh | bash
```

Then run `skitz`.

<details>
<summary>Manual install</summary>

```bash
git clone https://github.com/htelsiz/skitz.git
cd skitz
go build -o skitz .
sudo mv skitz /usr/local/bin/
```

</details>

## Features

| Feature | Description |
|---------|-------------|
| **Dashboard** | Tabbed interface with Resources and Actions |
| **Command Execution** | Run annotated commands with `^run` tags |
| **AI Integration** | Ask AI, generate commands (Anthropic, OpenAI, Ollama) |
| **Resource Management** | Add, edit, delete markdown command references |
| **Command Palette** | Quick access via `Ctrl+K` |
| **MCP Support** | Connect to [Model Context Protocol](https://modelcontextprotocol.io/) servers |

## Resources

Resources are markdown files in `~/.config/skitz/resources/` that define your command references:

```markdown
# Docker

`docker ps -a` list containers ^run
`docker exec -it {{c}} bash` shell into container ^run:c
```

- `^run` marks a command as executable
- `^run:varname` prompts for `{{varname}}` before running

Press `Enter` on any `^run` command to execute it directly from the TUI.

## Configuration

Config location: `~/.config/skitz/config.yaml`

```yaml
ai:
  default_provider: "anthropic"
  providers:
    - name: "anthropic"
      provider_type: "anthropic"  # or: openai, ollama, openai-compatible
      api_key: "sk-ant-..."
      enabled: true

mcp:
  enabled: true
  servers:
    - name: "local"
      url: "http://localhost:8001/mcp/"
```

Configure providers interactively via **Actions > Configure Providers**.

<details>
<summary>Keyboard shortcuts</summary>

### Dashboard

| Key | Action |
|-----|--------|
| `Tab` | Switch Resources/Actions |
| `e` | Edit resource in `$EDITOR` |
| `d` | Delete resource |
| `Enter` | Open/execute |
| `Ctrl+K` | Command palette |

### Resource View

| Key | Action |
|-----|--------|
| `a` | Ask AI |
| `Ctrl+G` | Generate command |
| `Ctrl+Y` | Copy to clipboard |
| `Enter` | Run command |

### Navigation

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Move up/down |
| `h/l` or `←/→` | Switch sections |
| `g/G` | Jump to top/bottom |
| `Ctrl+D/U` | Page down/up |
| `q` or `Esc` | Back/quit |

</details>

## Development

```bash
go build -o skitz .
go test ./...
```

**Stack:** BubbleTea, Lipgloss, Glamour, Huh, MCP-Go

<details>
<summary>Generate demo GIFs</summary>

```bash
brew install vhs ffmpeg ttyd
cd demos && vhs demo.tape
```

See [demos/](./demos/) for all tape files.

</details>

## License

MIT
