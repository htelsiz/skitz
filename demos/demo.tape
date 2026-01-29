# Main demo - Overview of skitz features
# Run: vhs demo.tape

Output ../docs/demo.gif
Output ../docs/demo.mp4

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

# Navigate resources
Down
Sleep 500ms
Down
Sleep 500ms
Up
Sleep 500ms

# Open a resource
Enter
Sleep 2s

# Browse commands
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms

# Go back to dashboard
Escape
Sleep 1s

# Switch to Actions tab
Tab
Sleep 1s

# Back to resources
Tab
Sleep 1s

# Open command palette
Ctrl+K
Sleep 1500ms
Escape
Sleep 1s

# Quit
Type "q"
Sleep 1s
