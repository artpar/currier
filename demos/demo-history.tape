# Currier History Demo
# Shows: Complete cycle - browse history, load request, replay, see response

Output demos/demo-history.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 2500ms

# Switch to History panel
Type "H"
Sleep 1500ms

# Show history items - navigate through them
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms

# Jump to bottom of history
Type "G"
Sleep 1200ms

# Jump to top
Type "gg"
Sleep 1200ms

# Navigate to an interesting request
Type "j"
Sleep 500ms
Type "j"
Sleep 800ms

# Select this history item to load it
Enter
Sleep 1500ms

# Show the loaded request in the request pane
Type "2"
Sleep 1500ms

# The request is now loaded - we can see URL, method, headers
# Switch through request tabs to see full details
Type "]"
Sleep 1000ms
Type "]"
Sleep 1000ms
Type "["
Sleep 500ms
Type "["
Sleep 1000ms

# Replay this request - send it again
Enter
Sleep 3000ms

# View the fresh response
Type "3"
Sleep 1500ms

# Scroll through the response
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1200ms

# Go back to history to show it recorded the replay
Type "H"
Sleep 1500ms

# New entry at top
Type "gg"
Sleep 1200ms

# Quit
Type "q"
Sleep 500ms
