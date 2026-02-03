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
| `main.go` | Entry point |
| `internal/app/model.go` | BubbleTea model; core state |
| `internal/app/views.go` | View rendering |
| `internal/app/keyboard.go` | Keyboard input handling |
| `internal/app/palette.go` | Command palette (Ctrl+K) |
| `internal/app/wizards.go` | Multi-step wizard flows |
| `internal/app/actions.go` | Quick actions |
| `internal/app/types.go` | Data types and metadata |
| `internal/app/styles.go` | UI styling |
| `internal/app/agent.go` | BIA agent integration |
| `internal/app/cloud_agent.go` | Docker/E2B agent runtime |
| `internal/app/deploy.go` | Azure deployment wizard |
| `internal/app/terminal.go` | Embedded terminal |
| `internal/config/config.go` | YAML configuration |
| `internal/mcp/client.go` | MCP client |

### Data Storage

- **Config**: `~/.config/skitz/config.yaml`
- **History**: `~/.local/share/skitz/history.json`
- **Agent History**: `~/.local/share/skitz/agent_history.json`
- **Resources**: `~/.config/skitz/resources/*.md`

## Essential Commands

**Build**: `go build -o skitz`
**Run**: `./skitz` or `./skitz <resource>`
**All tests**: `go test ./...`
**Verbose tests**: `go test -v ./...`
**Integration tests**: `MCP_TEST_ENABLED=1 go test ./...`
**Single test**: `go test -run TestFunctionName`

## Key Components

### MCP Client (`internal/mcp/client.go`)

HTTP-based client for Model Context Protocol servers:
- Connection management with context timeouts
- Tool listing and invocation
- Multi-server status monitoring

**Environment**: `SKITZ_MCP_URL` (default: `http://localhost:8001/mcp/`)

### BIA Agent (`internal/app/agent.go`)

Code review integration via MCP:
- Invokes `bia_junior_agent` tool for code reviews
- Supports file path input or paste mode
- Accessed via Command Palette (Ctrl+K)

### Cloud Agent (`internal/app/cloud_agent.go`)

Agent runtime environments:
- Docker container execution
- E2B cloud sandbox support

### Deploy Wizard (`internal/app/deploy.go`)

Azure deployment:
- Interactive configuration flow
- Azure CLI integration
- Container Instances (ACI) or Pipelines targets

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

## Go Style

### Formatting

- Run `gofmt` before committing
- Imports: stdlib, then external, then internal (blank lines between groups)

```go
import (
    "context"
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/htelsiz/skitz/internal/config"
)
```

### Naming

**Packages**: short, lowercase, no underscores, match directory name

```go
package config
package mcp
```

**Avoid repetition** - don't repeat package name in exports

```go
func Load() Config { ... }  // config.Load()
```

**Interfaces**: `-er` suffix for single-method; no redundant prefixes

```go
type Reader interface { Read(p []byte) (n int, err error) }
type Storage interface { ... }
```

**Locks**: named `lock`, never embedded

```go
type Cache struct {
    lock sync.Mutex
    data map[string]string
}
```

### Error Handling

**Wrap with context** using `%w`

```go
if err := db.Connect(ctx); err != nil {
    return fmt.Errorf("failed to connect to database: %w", err)
}
```

**Sentinel errors** for expected conditions

```go
var ErrNotFound = errors.New("not found")

if errors.Is(err, ErrNotFound) { ... }
```

### Functions

**Keep functions focused** - split by responsibility

```go
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKey(msg)
    case tickMsg:
        return m.handleTick(msg)
    }
    return m, nil
}
```

**Accept interfaces, return concrete types**

```go
func NewClient(r io.Reader) *Client { ... }
```

### Structs

**Composite literals** with field names

```go
cfg := Config{
    Name:    "default",
    Timeout: 30 * time.Second,
}
```

**Zero values** with `var`

```go
var buf bytes.Buffer
var coords Point
```

### Testing

**Table-driven tests**

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int
        wantErr bool
    }{
        {name: "valid", input: "42", want: 42},
        {name: "negative", input: "-1", want: -1},
        {name: "invalid", input: "abc", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
            }
        })
    }
}
```

**Test helpers** call `t.Helper()`

```go
func mustLoadFixture(t *testing.T, path string) []byte {
    t.Helper()
    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("failed to load fixture %s: %v", path, err)
    }
    return data
}
```

### Concurrency

**Channel direction** - specify when possible

```go
func producer(out chan<- int) { ... }
func consumer(in <-chan int) { ... }
```

**Goroutines must be cancellable**

```go
func watch(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case event := <-events:
                handle(event)
            }
        }
    }()
}
```

### Package Organization

- `internal/` for private packages
- Group related types in same file
- Split large packages by domain (`client.go`, `server.go`, `types.go`)
- Name packages by function, not `util`

### Patterns

- **Elm architecture**: BubbleTea's Update/View/Init pattern
- **Map-based lookups**: Metadata and action handlers

## Available Resources

Auto-discovered from `~/.config/skitz/resources/` and exposed in the dashboard:

- `azure` - Azure CLI commands
- `claude` - Claude AI assistant
- `cursor` - Cursor editor shortcuts
- `docker` - Docker commands
- `gcp` - Google Cloud CLI commands
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
