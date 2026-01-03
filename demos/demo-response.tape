# Currier Response Demo
# Shows: Complete response exploration - body, headers, cookies tabs, scrolling, copying

Output demos/demo-response.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 2500ms

# Create a request that returns rich response data
Type "n"
Sleep 800ms
Type "https://httpbin.org/response-headers?X-Custom=demo&Server=currier"
Sleep 1000ms

# Send request
Enter
Sleep 3000ms

# Focus on response pane
Type "3"
Sleep 1500ms

# We're on Body tab - scroll through the JSON response
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 800ms

# Jump to bottom
Type "G"
Sleep 1200ms

# Jump to top
Type "gg"
Sleep 1200ms

# Switch to Headers tab to see response headers
Type "]"
Sleep 1500ms

# Scroll through headers
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1000ms

# Switch to Cookies tab
Type "]"
Sleep 1500ms

# Back to Body tab
Type "["
Sleep 500ms
Type "["
Sleep 1000ms

# Copy the response body
Type "y"
Sleep 1500ms

# Now make a request that sets cookies so we can see them
Type "n"
Sleep 800ms
Type "https://httpbin.org/cookies/set/session/demo123"
Sleep 1000ms

Enter
Sleep 3000ms

# Go to response and check Cookies tab
Type "3"
Sleep 1000ms
Type "]"
Sleep 800ms
Type "]"
Sleep 1500ms

# Now we can see the cookie that was set
Sleep 1500ms

# Quit
Type "q"
Sleep 500ms
