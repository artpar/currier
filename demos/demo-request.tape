# Currier Request Demo
# Shows: New request, typing URL, Alt+Enter to send

Output demos/demo-request.gif

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
Sleep 400ms

# Type URL
Type "https://httpbin.org/get"
Sleep 300ms

# Send with Alt+Enter (while still editing!)
Alt+Enter
Sleep 1.5s

# Show response loaded
Sleep 500ms

# Create another request with POST
Type "n"
Sleep 300ms
Type "https://httpbin.org/post"
Sleep 200ms
Escape
Sleep 200ms

# Change method (m cycles: GET → POST)
Type "m"
Sleep 300ms

# Send
Enter
Sleep 1.5s

# Quit
Type "q"
Sleep 300ms
