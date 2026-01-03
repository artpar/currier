# Currier Request Demo
# Shows: Complete cycle - create request, send, view response, save to collection

Output demos/demo-request.gif

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

# Type URL
Type "https://httpbin.org/get"
Sleep 500ms

# Send request with Alt+Enter
Alt+Enter
Sleep 2s

# Move to response pane
Type "3"
Sleep 500ms

# Scroll through the JSON response
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 300ms

# Jump to bottom/top
Type "G"
Sleep 400ms
Type "gg"
Sleep 400ms

# Now make a POST request
Type "n"
Sleep 400ms
Type "https://httpbin.org/post"
Sleep 400ms
Escape
Sleep 200ms

# Change method to POST
Type "m"
Sleep 400ms

# Switch to Body tab
Type "]"
Sleep 300ms
Type "]"
Sleep 300ms
Type "]"
Sleep 400ms

# Change body type to JSON
Type "t"
Sleep 400ms

# Edit the body
Type "e"
Sleep 200ms
Type@50ms '{"name": "currier", "type": "api-client"}'
Sleep 400ms
Escape
Sleep 300ms

# Send the POST request
Enter
Sleep 2s

# View the response
Type "3"
Sleep 500ms

# Scroll to see the posted data
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 500ms

# Save this request to a collection
Type "s"
Sleep 500ms
Type "My API Tests"
Sleep 300ms
Enter
Sleep 600ms

# Show it saved - switch to collections
Type "C"
Sleep 600ms

# Quit
Type "q"
Sleep 300ms
