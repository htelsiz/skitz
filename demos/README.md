# Demo Recordings

Terminal recordings using [VHS](https://github.com/charmbracelet/vhs).

## Requirements

```bash
brew install vhs ffmpeg ttyd
```

## Generate Demos

```bash
# Generate all demos
cd demos
vhs demo.tape      # Main overview
vhs commands.tape  # Command execution
vhs ai.tape        # AI features
vhs manage.tape    # Resource management

# Or generate all at once
for tape in *.tape; do vhs "$tape"; done
```

## Output

Demos output to `docs/`:
- `demo.gif` / `demo.mp4` - Main overview
- `commands.gif` - Running commands
- `ai.gif` - AI features
- `manage.gif` - Resource management

## Customization

Edit tape files to adjust:
- `Set FontSize` - Terminal font size
- `Set Width/Height` - Output dimensions
- `Set Theme` - Color theme (Catppuccin Mocha, Dracula, etc.)
- `Sleep` durations - Timing between actions

## Available Themes

```
Catppuccin Mocha, Catppuccin Latte, Dracula,
GitHub Dark, Gruvbox, Nord, One Dark, Tokyo Night
```
