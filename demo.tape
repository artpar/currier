# Currier TUI Demo
# Run with: vhs demo.tape

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"

Type "currier"
Enter
Sleep 2s

# Show the main interface
Sleep 1s

# Navigate to History
Type "H"
Sleep 1s

# Navigate through history items
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms

# Select a history item
Enter
Sleep 1s

# Switch to Collections
Type "1"
Sleep 500ms
Type "C"
Sleep 1s

# Create a new request
Type "n"
Sleep 1s

# Type a URL
Type "https://httpbin.org/get"
Sleep 500ms
Enter
Sleep 1s

# Send the request
Enter
Sleep 3s

# Show response tabs
Type "]"
Sleep 500ms
Type "]"
Sleep 500ms

# Show help
Type "?"
Sleep 2s
Type "?"
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
