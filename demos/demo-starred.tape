# Currier Starred/Favorites Demo
# Shows: Star requests with *, visual indicator

Output demos/demo-starred.gif

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

# Navigate to a request
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Star the request with *
Type "*"
Sleep 500ms

# Navigate to another request
Type "j"
Sleep 200ms

# Star it too
Type "*"
Sleep 500ms

# Navigate around to show starred indicators
Type "k"
Sleep 300ms
Type "k"
Sleep 300ms
Type "j"
Sleep 300ms

# Unstar a request
Type "*"
Sleep 500ms

# Show the remaining starred request
Type "j"
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
