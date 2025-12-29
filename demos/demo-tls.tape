# Currier TLS/Certificate Settings Demo
# Shows: Ctrl+T to open, Tab to navigate fields, Enter to save

Output demos/demo-tls.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Type "currier"
Enter
Sleep 2s

# Open TLS settings with Ctrl+T
Ctrl+T
Sleep 1s

# Type a certificate path
Type "/path/to/client.pem"
Sleep 600ms

# Tab to next field (key file)
Tab
Sleep 500ms
Type "/path/to/client-key.pem"
Sleep 600ms

# Tab to CA cert field
Tab
Sleep 500ms
Type "/path/to/ca.pem"
Sleep 600ms

# Tab to insecure toggle
Tab
Sleep 500ms

# Toggle insecure mode with space
Type " "
Sleep 500ms

# Save with Enter
Enter
Sleep 1s

# Quit
Type "q"
Sleep 500ms
