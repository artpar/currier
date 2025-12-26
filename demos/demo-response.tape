# Currier Response Demo
# Shows: View response, scroll, switch tabs, copy

Output demos/demo-response.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 2

Type "currier"
Enter
Sleep 1s

# Create and send request
Type "n"
Sleep 200ms
Type "https://httpbin.org/get"
Alt+Enter
Sleep 1.5s

# Focus response pane
Type "3"
Sleep 300ms

# Scroll down
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms

# Scroll up
Type "k"
Sleep 150ms
Type "k"
Sleep 200ms

# Go to bottom
Type "G"
Sleep 300ms

# Go to top
Type "gg"
Sleep 300ms

# Switch to Headers tab
Type "]"
Sleep 400ms

# Switch to Timing tab
Type "]"
Sleep 200ms
Type "]"
Sleep 400ms

# Back to Body
Type "["
Sleep 200ms
Type "["
Sleep 200ms
Type "["
Sleep 300ms

# Copy response
Type "y"
Sleep 500ms

# Quit
Type "q"
Sleep 300ms
