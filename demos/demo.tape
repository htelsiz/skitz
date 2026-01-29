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

# Navigate to Docker resource
Down
Sleep 500ms
Down
Sleep 500ms

# Open Docker resource
Enter
Sleep 2s

# Browse commands
Down
Sleep 400ms
Down
Sleep 400ms

# Run command - show running containers
Enter
Sleep 2s

# Command executes and shows output
Sleep 3s

# Go back to dashboard
Escape
Sleep 1s
Escape
Sleep 1s

# Navigate to Git resource
Down
Sleep 500ms

# Open Git resource
Enter
Sleep 2s

# Select git log command
Down
Sleep 400ms

# Run git log
Enter
Sleep 2s

# Show output
Sleep 3s

# Go back to dashboard
Escape
Sleep 1s
Escape
Sleep 1s

# Switch to Actions tab
Tab
Sleep 1s

# Navigate to Configure Providers
Down
Sleep 400ms
Down
Sleep 400ms

# Open preferences
Enter
Sleep 2s

# Show preferences UI
Sleep 2s

# Exit preferences
Escape
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
