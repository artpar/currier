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

# Show the main interface with stacked History/Collections
Sleep 1s

# Focus History section
Type "H"
Sleep 1s

# Navigate through history items
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms

# Select a history item
Enter
Sleep 1s

# Focus Collections section
Type "1"
Sleep 400ms
Type "C"
Sleep 1s

# Create a new request
Type "n"
Sleep 800ms

# Type a URL
Type "https://httpbin.org/get"
Sleep 400ms
Enter
Sleep 800ms

# Send the request
Enter
Sleep 3s

# Show response tabs
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms

# Show help
Type "?"
Sleep 2s
Type "?"
Sleep 400ms

# Quit
Type "q"
Sleep 500ms
