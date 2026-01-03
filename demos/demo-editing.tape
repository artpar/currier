# Currier Editing Demo
# Shows: Complete editing cycle - URL, headers, query params, body, send and verify

Output demos/demo-editing.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 1.5s

# Create new request
Type "n"
Sleep 400ms
Type "https://httpbin.org/anything"
Sleep 400ms
Escape
Sleep 300ms

# Switch to Headers tab
Type "]"
Sleep 400ms

# Add headers
Type "a"
Sleep 300ms
Type "X-Custom-Header"
Tab
Type "my-custom-value"
Enter
Sleep 500ms

Type "a"
Sleep 200ms
Type "X-Request-ID"
Tab
Type "demo-12345"
Enter
Sleep 500ms

# Switch to Query tab
Type "]"
Sleep 400ms

# Add query parameters
Type "a"
Sleep 300ms
Type "search"
Tab
Type "currier"
Enter
Sleep 400ms

Type "a"
Sleep 200ms
Type "limit"
Tab
Type "10"
Enter
Sleep 500ms

# Switch to Body tab
Type "]"
Sleep 400ms

# Change body type to JSON
Type "t"
Sleep 400ms

# Edit body
Type "e"
Sleep 200ms
Type@50ms '{"message": "Hello from Currier!", "demo": true}'
Sleep 400ms
Escape
Sleep 300ms

# Change method to POST
Type "m"
Sleep 400ms

# Go back to URL tab
Type "["
Sleep 200ms
Type "["
Sleep 200ms
Type "["
Sleep 500ms

# Send the request
Enter
Sleep 2s

# View response - httpbin echoes everything back
Type "3"
Sleep 500ms

# Scroll to see our data
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 500ms

# Switch to Headers tab in response
Type "]"
Sleep 600ms

# Back to body
Type "["
Sleep 400ms

# Quit
Type "q"
Sleep 300ms
