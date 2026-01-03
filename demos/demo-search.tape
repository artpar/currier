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
Sleep 1.5s

# Switch to History
Type "H"
Sleep 600ms

# Show history items
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 300ms

# Start search
Type "/"
Sleep 400ms

# Type search query
Type "post"
Sleep 500ms

# Execute search
Enter
Sleep 600ms

# Navigate through filtered results
Type "j"
Sleep 300ms
Type "j"
Sleep 400ms

# Clear search with Escape
Escape
Sleep 600ms

# Search again
Type "/"
Sleep 300ms
Type "get"
Sleep 400ms
Enter
Sleep 600ms

# Navigate results
Type "j"
Sleep 200ms
Type "k"
Sleep 200ms

# Select to load
Enter
Sleep 600ms

# Show loaded request
Type "2"
Sleep 500ms

# Send to verify
Enter
Sleep 2s

# View response
Type "3"
Sleep 600ms

# Quit
Type "q"
Sleep 300ms
