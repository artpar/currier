# Currier Bulk Select Demo
# Shows: Multi-select with Space, visual mode, bulk operations

Output demos/demo-bulk-select.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Switch to Collections
Type "C"
Sleep 300ms

# Expand a collection
Type "l"
Sleep 300ms

# Navigate down to requests
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Toggle selection with Space
Type " "
Sleep 300ms

# Move down and select more
Type "j"
Sleep 200ms
Type " "
Sleep 300ms

Type "j"
Sleep 200ms
Type " "
Sleep 300ms

# Show selection count in header
Sleep 500ms

# Enter visual mode with v
Type "v"
Sleep 300ms

# Extend selection down
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Exit visual mode
Escape
Sleep 300ms

# Copy selected as cURL with c
Type "c"
Sleep 500ms

# Clear selection with Escape
Escape
Sleep 300ms

# Select all with Ctrl+a
Ctrl+a
Sleep 500ms

# Clear again
Escape
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
