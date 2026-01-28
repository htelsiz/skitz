# Embedded Terminal Pane MVP

## Goal
Run commands in an embedded terminal pane within skitz, allowing hotkey (F1) to toggle focus back to skitz without killing the subprocess.

## Architecture

```
┌─────────────────────────────────────────┐
│  SKITZ TUI                              │
├─────────────────────────────────────────┤
│  Command Table (existing)               │
├─────────────────────────────────────────┤
│  ┌───────────────────────────────────┐  │
│  │  Embedded Terminal (vterm)        │  │
│  │  $ claude                         │  │
│  │  > hello                          │  │
│  └───────────────────────────────────┘  │
├─────────────────────────────────────────┤
│  [F1] toggle focus  │ status...        │
└─────────────────────────────────────────┘
```

## Dependencies
```
go get github.com/aaronjanse/3mux/vterm
go get github.com/aaronjanse/3mux/ecma48
go get github.com/creack/pty
```

## Files to Modify

| File | Changes |
|------|---------|
| `model.go` | Add `EmbeddedTerm` struct, new message types, Update() handlers |
| `exec.go` | New `executeEmbedded()` using PTY + vterm instead of `tea.Exec()` |
| `views.go` | Render terminal pane in `renderResourceView()` |

## Implementation

### 1. Add Terminal State (model.go)

```go
type EmbeddedTerm struct {
    active    bool
    focused   bool           // F1 toggles this
    vt        *vterm.VTerm   // Terminal emulator
    pty       *os.File       // PTY master
    cmd       *exec.Cmd      // Running process
    done      chan error     // Process exit signal
}

// Add to model struct:
term EmbeddedTerm

// New message types:
type termOutputMsg struct{}      // Signals vterm has new content
type termExitMsg struct{ err error }
```

### 2. Execute via Embedded Terminal (exec.go)

```go
func (m *model) executeEmbedded(cmd command) tea.Cmd {
    return func() tea.Msg {
        // 1. Create PTY
        c := exec.Command("sh", "-c", cmd.cmd)
        ptmx, _ := pty.Start(c)

        // 2. Initialize vterm (sized to pane dimensions)
        vt := vterm.NewVTerm()
        vt.Reshape(width, height)

        // 3. Pipe PTY output to vterm
        go func() {
            buf := make([]byte, 4096)
            for {
                n, err := ptmx.Read(buf)
                if err != nil { break }
                vt.ProcessStream(buf[:n])
                // Signal redraw needed
            }
        }()

        // 4. Store state
        m.term = EmbeddedTerm{
            active: true, focused: true,
            vt: vt, pty: ptmx, cmd: c,
        }

        return termOutputMsg{}
    }
}
```

### 3. Keyboard Routing (model.go Update)

```go
case tea.KeyMsg:
    // F1 toggles terminal focus
    if keyStr == "f1" && m.term.active {
        m.term.focused = !m.term.focused
        return m, nil
    }

    // When terminal focused, send keys to PTY
    if m.term.active && m.term.focused {
        m.term.pty.Write([]byte(keyStr))
        return m, nil
    }

    // Otherwise, normal skitz key handling...
```

### 4. Render Terminal Pane (views.go)

```go
func (m model) renderTerminalPane() string {
    if !m.term.active {
        return ""
    }

    // Get vterm screen as [][]vterm.Char
    screen := m.term.vt.Screen

    // Convert to string with ANSI styling
    var lines []string
    for _, row := range screen {
        var line strings.Builder
        for _, ch := range row {
            // Apply ch.Style (fg, bg, bold, etc.)
            line.WriteString(styledChar(ch))
        }
        lines = append(lines, line.String())
    }

    // Add border and focus indicator
    border := lipgloss.Color("240")
    if m.term.focused {
        border = lipgloss.Color("99") // Purple when focused
    }

    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(border).
        Render(strings.Join(lines, "\n"))
}
```

### 5. Integrate in renderResourceView()

After the command table, add:
```go
if m.term.active {
    termPane := m.renderTerminalPane()
    content = lipgloss.JoinVertical(lipgloss.Left,
        commandTableView,
        termPane,
    )
}
```

## Key Behaviors

- **F1**: Toggle focus between skitz and terminal
- **When focused on terminal**: All keys go to subprocess
- **When focused on skitz**: Navigate commands, sections normally
- **Visual indicator**: Purple border = terminal focused, gray = skitz focused
- **Process exit**: Terminal pane stays visible with output, shows exit status

## Verification

1. Build: `go build -o skitz`
2. Run: `./skitz`
3. Navigate to a resource, select `claude` command, press Enter
4. Verify terminal pane appears with claude running
5. Press F1 - verify focus returns to skitz (can navigate)
6. Press F1 again - verify focus returns to terminal
7. Type in terminal - verify input goes to claude
8. Exit claude normally - verify return to skitz with output preserved
