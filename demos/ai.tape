# Demo - AI features (Ask AI and Generate Commands)
# Run: vhs ai.tape

Output ../docs/ai.gif

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

# Open a resource
Enter
Sleep 1500ms

# Press 'a' to Ask AI
Type "a"
Sleep 1500ms

# Type a question
Type "How do I list running containers?"
Sleep 1s

# Submit the question
Enter
Sleep 3s

# Close AI panel
Escape
Sleep 1s

# Generate command with Ctrl+G
Ctrl+G
Sleep 1500ms

# Type what we want
Type "show disk usage sorted by size"
Sleep 1s

# Submit
Enter
Sleep 3s

# Close panel
Escape
Sleep 1s

# Back to dashboard
Escape
Sleep 1s

# Quit
Type "q"
Sleep 500ms
