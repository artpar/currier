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
Sleep 2500ms

# Create new request
Type "n"
Sleep 800ms

# Type URL (fast typing)
Type "https://httpbin.org/get"
Sleep 1200ms

# Send request with Alt+Enter while still in edit mode
Alt+Enter
Sleep 3000ms

# Move to response pane to examine the response
Type "3"
Sleep 1500ms

# Scroll through the JSON response
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 800ms

# Jump to bottom of response
Type "G"
Sleep 1200ms

# Jump back to top
Type "gg"
Sleep 1200ms

# Now let's make a POST request
Type "n"
Sleep 800ms

Type "https://httpbin.org/post"
Sleep 800ms
Escape
Sleep 500ms

# Change method to POST
Type "m"
Sleep 1000ms

# Switch to Body tab to add request body
Type "]"
Sleep 600ms
Type "]"
Sleep 600ms
Type "]"
Sleep 1000ms

# Change body type to JSON
Type "t"
Sleep 800ms

# Edit the body
Type "e"
Sleep 500ms
Type@100ms '{"name": "currier", "type": "api-client"}'
Sleep 1200ms
Escape
Sleep 800ms

# Send the POST request
Enter
Sleep 3000ms

# View the response - see our JSON echoed back
Type "3"
Sleep 1500ms

# Scroll to see the posted data in response
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

# Save this request to a collection
Type "s"
Sleep 1200ms

# Type collection name
Type "My API Tests"
Sleep 800ms
Enter
Sleep 1500ms

# Show it saved - switch to collections
Type "C"
Sleep 1500ms

# Quit
Type "q"
Sleep 500ms
