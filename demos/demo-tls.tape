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
Sleep 2500ms

# Open TLS settings with Ctrl+T
Ctrl+T
Sleep 1500ms

# The TLS dialog is now open showing:
# - Client Certificate path
# - Client Key path
# - CA Certificate path
# - Skip TLS Verification checkbox

# Tab to client certificate field
Tab
Sleep 800ms

# Enter a certificate path
Type "/path/to/client.pem"
Sleep 1200ms

# Tab to client key field
Tab
Sleep 800ms

Type "/path/to/client-key.pem"
Sleep 1200ms

# Tab to CA certificate field
Tab
Sleep 800ms

Type "/path/to/ca.pem"
Sleep 1200ms

# Tab to Skip TLS Verification
Tab
Sleep 800ms

# Toggle skip verification (for self-signed certs)
Space
Sleep 1200ms

# Save settings with Enter
Enter
Sleep 1500ms

# Settings are saved
Sleep 1000ms

# Create a request to a secure endpoint
Type "n"
Sleep 800ms
Type "https://secure-api.example.com/endpoint"
Sleep 1000ms
Escape
Sleep 800ms

# The request would use our TLS settings when sent
Sleep 1500ms

# Open TLS settings again to show they're saved
Ctrl+T
Sleep 2000ms

# Close dialog
Escape
Sleep 1000ms

# Quit
Type "q"
Sleep 500ms
