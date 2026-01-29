# Demo - Resource management (edit, delete)
# Run: vhs manage.tape

Output ../docs/manage.gif

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

# Navigate to a resource
Down
Sleep 500ms

# Press 'e' to edit (opens $EDITOR)
# Note: This will open editor, cancel for demo
Type "e"
Sleep 2s

# Editor opens - quit without changes
Type ":q"
Enter
Sleep 1s

# Press 'd' to delete (shows confirmation)
Type "d"
Sleep 1500ms

# Cancel delete
Escape
Sleep 1s

# Switch to Actions tab
Tab
Sleep 1s

# Select "Add Resource"
Enter
Sleep 2s

# Cancel
Escape
Sleep 1s

# Quit
Type "q"
Sleep 500ms
