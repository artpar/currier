# Currier Cookie Management Demo
# Shows: Automatic cookie capture, persistence, Ctrl+K to clear

Output demos/demo-cookies.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Type "currier"
Enter
Sleep 2s

# Create new request to httpbin cookies endpoint
Type "n"
Sleep 500ms

# Type URL that sets cookies
Type "https://httpbin.org/cookies/set/session/abc123"
Sleep 500ms

# Send request (sets cookie)
Enter
Sleep 3s

# Focus response panel and switch to Cookies tab
Type "3"
Sleep 300ms
Type "]"
Sleep 300ms
Type "]"
Sleep 1s

# Create another request to verify cookies are sent
Type "n"
Sleep 500ms

Type "https://httpbin.org/cookies"
Sleep 500ms

# Send request (cookies automatically included in response body)
Enter
Sleep 3s

# Clear cookies with Ctrl+K
Ctrl+K
Sleep 1s

# Send again to show cookies are cleared
Enter
Sleep 3s

# Quit
Type "q"
Sleep 500ms
