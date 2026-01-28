# Skitz TUI - Development Context

## Project Overview
Skitz is a terminal UI application for browsing command reference documentation stored as markdown files in `~/.config/skitz/resources/`. Each resource (docker, git, claude) has:
- `{name}.md` - Quick reference (compact command list)
- `{name}-detail.md` - Detailed documentation with `## Section` headers

## Current State
The app has a two-pane layout:
- **Left pane (20%)**: Resource list (docker, git, claude)
- **Right pane (80%)**: Section tabs at top + rendered markdown content below

### What Works
- Resource selection with `↑↓` in left pane
- Section tabs in right pane with `←→` navigation
- Number keys `1-9` to jump to section tabs
- `tab` to switch focus between panes
- Content scrolling with `↑↓` when right pane focused
- Status bar showing breadcrumb and context-aware hints
- Glamour markdown rendering

### Recent Fixes
1. **Tab styling fixed** - Implemented proper lipgloss tab borders with:
   - Active tabs have open bottom border (connects to content)
   - Inactive tabs have closed bottom border with `┴` corners
   - Gap filler extends the bottom border line across remaining width

### Minor Issues
1. **Tabs are truncated** - Long section names show as "Starting S..", "Slash Comm.." etc. (by design for narrow widths)

## Next Steps / Future Enhancements
1. **Command execution** - Press `x` on a command to execute it (was in earlier versions)
2. **Search/filter** - Filter sections or commands with `/`
3. **Command palette** - `ctrl+p` for quick actions (was implemented, then removed for simplicity)
4. **Copy to clipboard** - Copy commands with `y`

## File Structure
```
cli/skitz/
├── main.go          # All application code (~500 lines)
├── go.mod           # Dependencies: bubbletea, glamour, lipgloss
├── go.sum
├── skitz            # Built binary
└── CLAUDE_CONTEXT.md # This file
```

## Dependencies
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/glamour` - Markdown rendering
- `github.com/charmbracelet/lipgloss` - Styling and layout

## Key Design Decisions
1. **No bubbles/list** - We removed the bubbles list component because it was causing hangs. Using simple custom list rendering instead.
2. **No caching** - Glamour renders on each view; simple and stateless
3. **Single file** - All code in main.go for simplicity
4. **Value receivers on model** - Bubbletea pattern; Update returns new model

## Resources Location
```go
resourcesDir = filepath.Join(home, ".config", "skitz", "resources")
```

## Color Palette
```go
primary   = lipgloss.Color("205") // Pink - selected items, active tabs
secondary = lipgloss.Color("39")  // Cyan - focused borders, titles
subtle    = lipgloss.Color("241") // Gray - muted text, inactive
dark      = lipgloss.Color("236") // Dark bg for status bar
white     = lipgloss.Color("255") // Normal text
```

## Navigation Model
```
Focus 0 (Left pane - Resources):
  ↑↓     Navigate resources
  Enter  Focus right pane
  Tab    Switch to right pane

Focus 1 (Right pane - Sections):
  ←→     Switch section tabs
  ↑↓     Scroll content
  1-9    Jump to section tab
  Esc    Focus left pane
  Tab    Switch to left pane
```

## Build & Run
```bash
cd ~/projects/skitz
go build -o skitz .
./skitz           # Start with resource list
./skitz docker    # Start with docker selected
```
