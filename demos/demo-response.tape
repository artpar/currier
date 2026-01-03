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
Sleep 1.5s

# Create a request
Type "n"
Sleep 400ms
Type "https://httpbin.org/response-headers?X-Custom=demo&Server=currier"
Sleep 500ms

# Send request
Enter
Sleep 2s

# Focus on response pane
Type "3"
Sleep 500ms

# Scroll through the JSON response
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

# Switch to Headers tab
Type "]"
Sleep 600ms

# Scroll through headers
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 400ms

# Switch to Cookies tab
Type "]"
Sleep 600ms

# Back to Body tab
Type "["
Sleep 200ms
Type "["
Sleep 400ms

# Copy the response body
Type "y"
Sleep 600ms

# Make a request that sets cookies
Type "n"
Sleep 400ms
Type "https://httpbin.org/cookies/set/session/demo123"
Sleep 500ms

Enter
Sleep 2s

# Go to response and check Cookies tab
Type "3"
Sleep 400ms
Type "]"
Sleep 300ms
Type "]"
Sleep 800ms

# Quit
Type "q"
Sleep 300ms
