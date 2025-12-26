# Currier History Demo
# Shows: Browse history, select and replay

Output demos/demo-history.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 2

Type "currier"
Enter
Sleep 1s

# Switch to History
Type "H"
Sleep 400ms

# Navigate through history items
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Select a history item
Enter
Sleep 500ms

# Show request loaded
Sleep 300ms

# Send the request again
Enter
Sleep 1.5s

# Go back to history
Type "1"
Sleep 200ms

# Use G to go to bottom
Type "G"
Sleep 300ms

# Use gg to go to top
Type "gg"
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
