# Currier Search Demo
# Shows: Complete search cycle - search, filter results, select, load request

Output demos/demo-search.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 2500ms

# Switch to History (typically has more items to search)
Type "H"
Sleep 1500ms

# Show current history items
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 800ms

# Start search
Type "/"
Sleep 800ms

# Type search query
Type "post"
Sleep 1200ms

# Execute search - results are filtered
Enter
Sleep 1500ms

# Navigate through filtered results
Type "j"
Sleep 600ms
Type "j"
Sleep 800ms

# Clear search with Escape to show all items again
Escape
Sleep 1500ms

# Search again with different term
Type "/"
Sleep 600ms
Type "get"
Sleep 800ms
Enter
Sleep 1500ms

# Navigate filtered results
Type "j"
Sleep 500ms
Type "k"
Sleep 500ms

# Select a result to load it
Enter
Sleep 1500ms

# Show the loaded request
Type "2"
Sleep 1500ms

# Send to verify it works
Enter
Sleep 3000ms

# View response
Type "3"
Sleep 1500ms

# Quit
Type "q"
Sleep 500ms
