# Currier Overview Demo
# Shows: Full layout, all three panes, navigation, help system

Output demos/demo-overview.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 1.5s

# Navigate within collections - expand a collection
Type "j"
Sleep 300ms
Type "l"
Sleep 500ms

# Show requests inside the collection
Type "j"
Sleep 250ms
Type "j"
Sleep 250ms

# Select a request to load it
Enter
Sleep 800ms

# Move to Request pane (pane 2)
Type "2"
Sleep 600ms

# Show the request tabs - URL, Headers, Query, Body
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "["
Sleep 200ms
Type "["
Sleep 200ms
Type "["
Sleep 400ms

# Move to Response pane (pane 3)
Type "3"
Sleep 600ms

# Show response tabs - Body, Headers, Cookies
Type "]"
Sleep 400ms
Type "]"
Sleep 400ms
Type "["
Sleep 200ms
Type "["
Sleep 400ms

# Switch to History view
Type "H"
Sleep 600ms

# Navigate history
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 300ms

# Back to Collections
Type "C"
Sleep 600ms

# Show help overlay
Type "?"
Sleep 1.5s

# Close help
Type "?"
Sleep 400ms

# Quit
Type "q"
Sleep 300ms
