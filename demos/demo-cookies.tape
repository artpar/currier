# Currier Cookie Management Demo
# Shows: Complete cookie cycle - set cookie, VIEW cookie, verify auto-send, clear cookies

Output demos/demo-cookies.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 1.5s

# Make a request that SETS a cookie
Type "n"
Sleep 400ms
Type "https://httpbin.org/cookies/set/user_session/abc123xyz"
Sleep 500ms

# Send request
Enter
Sleep 2s

# Focus response pane
Type "3"
Sleep 400ms

# Switch to Cookies tab to SEE the cookie
Type "]"
Sleep 400ms
Type "]"
Sleep 800ms

# Now verify cookies are auto-sent
Type "n"
Sleep 400ms
Type "https://httpbin.org/cookies"
Sleep 500ms

# Send request
Enter
Sleep 2s

# View response - shows cookie was sent
Type "3"
Sleep 500ms

# Scroll to see cookies in response
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 600ms

# Clear all cookies with Ctrl+K
Ctrl+K
Sleep 600ms

# Send same request again
Enter
Sleep 2s

# View response - cookies now empty
Type "3"
Sleep 800ms

# Quit
Type "q"
Sleep 300ms
