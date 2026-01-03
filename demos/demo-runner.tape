# Currier Collection Runner Demo
# Shows: Complete cycle - run collection, watch progress, see test results

Output demos/demo-runner.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 1.5s

# Switch to Collections panel
Type "C"
Sleep 600ms

# Navigate to a collection
Type "j"
Sleep 300ms

# Expand to show requests
Type "l"
Sleep 500ms

# Show requests in collection
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 300ms

# Go back up to collection root
Type "h"
Sleep 400ms

# Run the collection
Ctrl+R
Sleep 700ms

# Watch the runner execute
Sleep 1s
Sleep 1s
Sleep 1s

# Results displayed
Sleep 800ms

# Scroll through results
Type "j"
Sleep 250ms
Type "j"
Sleep 250ms
Type "j"
Sleep 400ms

# Back to top
Type "gg"
Sleep 500ms

# Close the runner modal
Escape
Sleep 600ms

# Quit
Type "q"
Sleep 300ms
