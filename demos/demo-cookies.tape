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
Sleep 2500ms

# First, let's make a request that SETS a cookie
Type "n"
Sleep 800ms
Type "https://httpbin.org/cookies/set/user_session/abc123xyz"
Sleep 1000ms

# Send request - this will set a cookie
Enter
Sleep 3000ms

# Focus response pane
Type "3"
Sleep 1000ms

# Switch to Cookies tab to SEE the cookie that was set
Type "]"
Sleep 1000ms
Type "]"
Sleep 1500ms

# Viewer can now see the cookie: user_session=abc123xyz
Sleep 2000ms

# Now let's verify cookies are automatically sent in subsequent requests
# Make a request to /cookies which shows what cookies the server received
Type "n"
Sleep 800ms
Type "https://httpbin.org/cookies"
Sleep 1000ms

# Send request
Enter
Sleep 3000ms

# View response body - should show our cookie was sent automatically
Type "3"
Sleep 1500ms

# Scroll to see the cookies in the response JSON
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1200ms

# The response shows: {"cookies": {"user_session": "abc123xyz"}}
Sleep 1500ms

# Now let's clear all cookies with Ctrl+K
Ctrl+K
Sleep 1500ms

# Send the same request again
Enter
Sleep 3000ms

# View response - cookies should now be empty
Type "3"
Sleep 1500ms

# Response now shows: {"cookies": {}}
Sleep 2000ms

# Quit
Type "q"
Sleep 500ms
