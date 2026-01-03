# Currier Editing Demo
# Shows: Complete editing cycle - URL, headers, query params, body, then send and verify

Output demos/demo-editing.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 2500ms

# Create new request
Type "n"
Sleep 800ms
Type "https://httpbin.org/anything"
Sleep 800ms
Escape
Sleep 600ms

# Switch to Headers tab
Type "]"
Sleep 1000ms

# Add a custom header
Type "a"
Sleep 600ms
Type "X-Custom-Header"
Tab
Type "my-custom-value"
Enter
Sleep 1200ms

# Add another header
Type "a"
Sleep 500ms
Type "X-Request-ID"
Tab
Type "demo-12345"
Enter
Sleep 1200ms

# Switch to Query tab
Type "]"
Sleep 1000ms

# Add query parameters
Type "a"
Sleep 600ms
Type "search"
Tab
Type "currier"
Enter
Sleep 1000ms

Type "a"
Sleep 500ms
Type "limit"
Tab
Type "10"
Enter
Sleep 1200ms

# Switch to Body tab
Type "]"
Sleep 1000ms

# Change body type to JSON
Type "t"
Sleep 800ms

# Edit body content
Type "e"
Sleep 500ms
Type@100ms '{"message": "Hello from Currier!", "demo": true}'
Sleep 1000ms
Escape
Sleep 600ms

# Change method to POST (body requires POST/PUT/PATCH)
Type "m"
Sleep 1000ms

# Go back to URL tab to see the full request
Type "["
Sleep 500ms
Type "["
Sleep 500ms
Type "["
Sleep 1200ms

# Send the request
Enter
Sleep 3000ms

# View the response - httpbin.org/anything echoes everything back
Type "3"
Sleep 1500ms

# Scroll to see our headers in the response
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 1500ms

# Switch to Headers tab in response to see response headers
Type "]"
Sleep 1500ms

# Back to body
Type "["
Sleep 1000ms

# Quit
Type "q"
Sleep 500ms
