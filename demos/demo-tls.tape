# Currier TLS/Certificate Settings Demo
# Shows: Complete cycle - configure client certs, CA cert, skip verify option

Output demos/demo-tls.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 1.5s

# Open TLS settings
Ctrl+T
Sleep 700ms

# Tab to client certificate field
Tab
Sleep 400ms

# Enter certificate path
Type "/path/to/client.pem"
Sleep 500ms

# Tab to client key field
Tab
Sleep 400ms
Type "/path/to/client-key.pem"
Sleep 500ms

# Tab to CA certificate field
Tab
Sleep 400ms
Type "/path/to/ca.pem"
Sleep 500ms

# Tab to Skip TLS Verification
Tab
Sleep 400ms

# Toggle skip verification
Space
Sleep 500ms

# Save settings
Enter
Sleep 600ms

# Create a request to a secure endpoint
Type "n"
Sleep 400ms
Type "https://secure-api.example.com/endpoint"
Sleep 500ms
Escape
Sleep 400ms

# Open TLS settings again to show persistence
Ctrl+T
Sleep 1s

# Close dialog
Escape
Sleep 400ms

# Quit
Type "q"
Sleep 300ms
