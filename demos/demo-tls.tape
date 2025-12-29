# Currier TLS/Certificate Settings Demo
# Shows: Ctrl+T to open, Tab to navigate fields, Enter to save

Output demos/demo-tls.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Open TLS settings with Ctrl+T
Ctrl+T
Sleep 600ms

# Type a certificate path
Type "/path/to/client.pem"
Sleep 400ms

# Tab to next field (key file)
Tab
Sleep 300ms
Type "/path/to/client-key.pem"
Sleep 400ms

# Tab to CA cert field
Tab
Sleep 300ms
Type "/path/to/ca.pem"
Sleep 400ms

# Tab to insecure toggle
Tab
Sleep 300ms

# Toggle insecure mode with space
Type " "
Sleep 300ms

# Save with Enter
Enter
Sleep 500ms

# Show settings saved
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
