# Currier Proxy Settings Demo
# Shows: Complete cycle - configure proxy, send request through proxy, verify

Output demos/demo-proxy.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 2500ms

# Open proxy settings with P
Type "P"
Sleep 1500ms

# The proxy dialog is now open
# Type a proxy URL
Type "http://localhost:8080"
Sleep 1200ms

# Save with Enter
Enter
Sleep 1500ms

# Proxy is now configured - visible in status bar
Sleep 1000ms

# Create a request to test
Type "n"
Sleep 800ms
Type "https://httpbin.org/get"
Sleep 1000ms

# Send request - it will go through the proxy
Enter
Sleep 3000ms

# View response
Type "3"
Sleep 1500ms

# Scroll to see request details
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1200ms

# Open proxy settings again to clear
Type "P"
Sleep 1500ms

# Clear the proxy
Ctrl+U
Sleep 800ms

# Save empty (disables proxy)
Enter
Sleep 1500ms

# Proxy is now disabled
Sleep 1000ms

# Quit
Type "q"
Sleep 500ms
