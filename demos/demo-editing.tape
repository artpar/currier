# Currier Editing Demo
# Shows: Edit URL, switch tabs, add headers

Output demos/demo-editing.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Create new request
Type "n"
Sleep 300ms
Type "https://httpbin.org/get"
Enter
Sleep 300ms

# Edit URL
Type "e"
Sleep 300ms
Ctrl+U
Type "https://httpbin.org/headers"
Enter
Sleep 300ms

# Switch to Headers tab with ]
Type "]"
Sleep 300ms

# Add a header
Type "a"
Sleep 200ms
Type "X-Custom-Header"
Tab
Type "my-value"
Enter
Sleep 300ms

# Switch to Query tab
Type "]"
Sleep 300ms

# Switch to Body tab
Type "]"
Sleep 300ms

# Back to URL tab
Type "["
Sleep 200ms
Type "["
Sleep 200ms
Type "["
Sleep 300ms

# Send request
Enter
Sleep 1.5s

# Quit
Type "q"
Sleep 300ms
