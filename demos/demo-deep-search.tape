# Currier Deep Search Demo
# Shows: Search within body, headers, URL

Output demos/demo-deep-search.gif

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

# Start search with /
Type "/"
Sleep 300ms

# Type a search term that matches body content
Type "authorization"
Sleep 400ms

# Show results (requests with Authorization header)
Enter
Sleep 500ms

# Clear and search for URL pattern
Type "/"
Sleep 200ms
Type "api.example"
Sleep 400ms
Enter
Sleep 500ms

# Switch to History and search there
Type "H"
Sleep 300ms

# Search in history
Type "/"
Sleep 200ms
Type "json"
Sleep 400ms
Enter
Sleep 500ms

# Quit
Type "q"
Sleep 300ms
