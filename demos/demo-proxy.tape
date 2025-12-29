# Currier Proxy Settings Demo
# Shows: P to open, configure proxy URL, Enter to save, Esc to cancel

Output demos/demo-proxy.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Type "currier"
Enter
Sleep 2s

# Open proxy settings with P
Type "P"
Sleep 1s

# Type a proxy URL
Type "http://localhost:8080"
Sleep 800ms

# Save with Enter
Enter
Sleep 1s

# Open again to show saved value
Type "P"
Sleep 1s

# Cancel with Escape
Escape
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
