# skitz

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> A personal command reference TUI with AI-powered assistance, built with [Charm](https://charm.sh).

<img src="docs/demo.gif" alt="skitz demo" width="700">

---

## ⚡️ Quick Start

```bash
git clone https://github.com/htelsiz/skitz.git && cd skitz && ./install.sh
```

Then run:

```bash
skitz
```

---

## 📖 Features

| Feature | Description |
|---------|-------------|
| **Dashboard** | Tabbed interface with Resources and Actions |
| **Command Execution** | Run annotated commands with `^run` tags |
| **AI Integration** | Ask AI, generate commands (OpenAI, Anthropic, Ollama) |
| **Resource Management** | Add, edit (`e`), delete (`d`) resources |
| **Command Palette** | Quick access to tools and actions (`Ctrl+K`) |
| **MCP Support** | Model Context Protocol server integration |

---

## ⌨️ Keyboard Shortcuts

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

---

## ⚙️ Configuration

Config: `~/.config/skitz/config.yaml`

```yaml
ai:
  default_provider: "anthropic"
  providers:
    - name: "anthropic"
      provider_type: "anthropic"
      api_key: "sk-ant-..."
      enabled: true

history:
  enabled: true
  max_items: 100

mcp:
  enabled: true
  servers:
    - name: "local"
      url: "http://localhost:8001/mcp/"
```

> 💡 Configure providers via **Actions → Configure Providers** with connection testing.

---

## 📝 Resources

<img src="docs/commands.gif" alt="Command execution demo" width="700">

Resources are markdown files in `~/.config/skitz/resources/`:

```markdown
# Docker

`docker ps -a` list containers ^run
`docker exec -it {{c}} bash` shell ^run:c
```

- `^run` — executable command
- `^run:varname` — prompts for `{{varname}}`

<img src="docs/manage.gif" alt="Resource management demo" width="700">

---

## 🤖 AI Providers

<img src="docs/ai.gif" alt="AI features demo" width="700">

| Provider | Type | API Key Format |
|----------|------|----------------|
| Anthropic | `anthropic` | `sk-ant-...` |
| OpenAI | `openai` | `sk-...` |
| Ollama | `ollama` | (none, local) |
| Custom | `openai-compatible` | varies |

---

## 🛠️ Development

```bash
go build -o skitz ./cmd/skitz/
go test ./...
```

**Stack:** BubbleTea, Lipgloss, Huh, Harmonica

### Generate Demos

```bash
brew install vhs ffmpeg ttyd
cd demos && vhs demo.tape
```

See [demos/](./demos/) for all available recordings.

---

## 📄 License

MIT
