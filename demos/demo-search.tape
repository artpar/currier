# Currier Search Demo
# Shows: Search in collections/history

Output demos/demo-search.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Switch to History (has items to search)
Type "H"
Sleep 400ms

# Start search
Type "/"
Sleep 300ms

# Type search query
Type "post"
Sleep 300ms

# Execute search
Enter
Sleep 400ms

# Clear search with Escape
Escape
Sleep 300ms

# Search again
Type "/"
Sleep 200ms
Type "get"
Enter
Sleep 400ms

# Navigate results
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Select result
Enter
Sleep 500ms

# Quit
Type "q"
Sleep 300ms
