# Currier Cookie Management Demo
# Shows: Automatic cookie capture, persistence, Ctrl+K to clear

Output demos/demo-cookies.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Create new request to httpbin cookies endpoint
Type "n"
Sleep 300ms

# Type URL that sets cookies
Type "https://httpbin.org/cookies/set/session/abc123"
Sleep 300ms

# Send request (sets cookie)
Enter
Sleep 2s

# Create another request to see cookies sent back
Type "n"
Sleep 300ms

Type "https://httpbin.org/cookies"
Sleep 300ms

# Send request (cookies automatically included)
Enter
Sleep 2s

# Clear cookies with Ctrl+K
Ctrl+K
Sleep 500ms

# Send again to show cookies are cleared
Enter
Sleep 2s

# Quit
Type "q"
Sleep 300ms
