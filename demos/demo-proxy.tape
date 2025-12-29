# Currier Proxy Settings Demo
# Shows: P to open, configure proxy URL, Enter to save, Esc to cancel

Output demos/demo-proxy.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Open proxy settings with P
Type "P"
Sleep 600ms

# Type a proxy URL
Type "http://localhost:8080"
Sleep 500ms

# Save with Enter
Enter
Sleep 500ms

# Open again to show saved value
Type "P"
Sleep 600ms

# Cancel with Escape
Escape
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
