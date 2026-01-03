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
Sleep 1.5s

# Switch to History panel
Type "H"
Sleep 600ms

# Navigate through history
Type "j"
Sleep 250ms
Type "j"
Sleep 250ms
Type "j"
Sleep 300ms

# Jump to bottom/top
Type "G"
Sleep 400ms
Type "gg"
Sleep 400ms

# Navigate to a request
Type "j"
Sleep 200ms
Type "j"
Sleep 300ms

# Select to load it
Enter
Sleep 600ms

# Show the loaded request
Type "2"
Sleep 500ms

# Switch through request tabs
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "["
Sleep 200ms
Type "["
Sleep 400ms

# Replay - send it again
Enter
Sleep 2s

# View the fresh response
Type "3"
Sleep 500ms

# Scroll through response
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 400ms

# Go back to history - new entry at top
Type "H"
Sleep 500ms
Type "gg"
Sleep 500ms

# Quit
Type "q"
Sleep 300ms
