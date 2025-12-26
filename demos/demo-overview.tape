# Currier Overview Demo
# Shows: Stacked layout, H/C switching, pane navigation

Output demos/demo-overview.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 2

Type "currier"
Enter
Sleep 1s

# Show stacked History/Collections
Sleep 500ms

# Switch to History
Type "H"
Sleep 400ms

# Navigate history
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "k"
Sleep 300ms

# Switch to Collections
Type "C"
Sleep 400ms

# Tab through panes
Type "Tab"
Sleep 300ms
Type "Tab"
Sleep 300ms
Type "Tab"
Sleep 300ms

# Use number keys
Type "1"
Sleep 300ms
Type "2"
Sleep 300ms
Type "3"
Sleep 300ms

# Show help briefly
Type "?"
Sleep 800ms
Type "?"
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
