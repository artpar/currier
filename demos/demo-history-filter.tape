# Currier History Filtering Demo
# Shows: Filter history by method and status

Output demos/demo-history-filter.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Already in History view (default)
Sleep 300ms

# Show some history entries
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 300ms

# Filter by method - press 'm' to cycle
Type "m"
Sleep 400ms

# Header should show [GET]
Type "j"
Sleep 150ms
Type "j"
Sleep 300ms

# Cycle to POST
Type "m"
Sleep 400ms

# Now filter by status - press 's'
Type "s"
Sleep 400ms

# Header shows [POST,2xx]
Sleep 300ms

# Cycle to 4xx
Type "s"
Sleep 300ms
Type "s"
Sleep 400ms

# Clear all filters with 'x'
Type "x"
Sleep 500ms

# Filters cleared
Type "j"
Sleep 150ms
Type "j"
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
