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
Sleep 1.5s

# Open proxy settings
Type "P"
Sleep 700ms

# Type proxy URL
Type "http://localhost:8080"
Sleep 500ms

# Save
Enter
Sleep 600ms

# Create a request
Type "n"
Sleep 400ms
Type "https://httpbin.org/get"
Sleep 500ms

# Send - goes through proxy
Enter
Sleep 2s

# View response
Type "3"
Sleep 500ms

# Scroll
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 500ms

# Open proxy settings to clear
Type "P"
Sleep 700ms

# Clear the proxy
Ctrl+U
Sleep 400ms

# Save empty
Enter
Sleep 600ms

# Quit
Type "q"
Sleep 300ms
