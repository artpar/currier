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
Sleep 3s

# Show the three-pane layout - Collections/History on left, Request in middle, Response on right
# Start in Collections pane (pane 1)
Sleep 1s

# Navigate within collections - expand a collection
Type "j"
Sleep 600ms
Type "l"
Sleep 1s

# Show requests inside the collection
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms

# Select a request to load it
Enter
Sleep 1500ms

# Move to Request pane (pane 2) - show the loaded request
Type "2"
Sleep 1500ms

# Show the request tabs - URL, Headers, Query, Body
Type "]"
Sleep 1000ms
Type "]"
Sleep 1000ms
Type "]"
Sleep 1000ms
Type "["
Sleep 500ms
Type "["
Sleep 500ms
Type "["
Sleep 1000ms

# Move to Response pane (pane 3)
Type "3"
Sleep 1500ms

# Show response tabs - Body, Headers, Cookies
Type "]"
Sleep 1000ms
Type "]"
Sleep 1000ms
Type "["
Sleep 500ms
Type "["
Sleep 1000ms

# Switch to History view
Type "H"
Sleep 1500ms

# Navigate history
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 800ms

# Back to Collections
Type "C"
Sleep 1500ms

# Show help overlay
Type "?"
Sleep 3000ms

# Close help
Type "?"
Sleep 1000ms

# Quit
Type "q"
Sleep 500ms
