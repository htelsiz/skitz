# Demo - Running commands from resources
# Run: vhs commands.tape

Output ../docs/commands.gif

Require skitz

Set Shell "bash"
Set FontSize 28
Set Width 3840
Set Height 2160
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 30

# Start skitz
Type "skitz"
Sleep 500ms
Enter
Sleep 2s

# Open first resource
Enter
Sleep 1500ms

# Navigate to a command
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# Execute command (Enter to run)
Enter
Sleep 2s

# Command execution dialog appears
# Cancel it
Escape
Sleep 1s

# Copy command to clipboard
Ctrl+Y
Sleep 1s

# Back to dashboard
Escape
Sleep 1s

# Quit
Type "q"
Sleep 500ms
