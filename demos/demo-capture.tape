# Currier Traffic Capture Demo (HTTP & HTTPS)
# Shows: Start in capture mode, capture HTTP & HTTPS traffic, inspect captures

Output demos/demo-capture.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

# Start currier in capture mode - this starts proxy automatically
Type "currier --capture"
Enter
Sleep 2.5s

# The proxy is now running - get the port from the display
# We'll make requests in background after a delay

# Make HTTP request in background
Hide
Type "PORT=$(tmux capture-pane -p | grep -oE 'localhost:[0-9]+' | head -1 | cut -d: -f2)"
Enter
Sleep 200ms
Type "( sleep 2 && curl -s --proxy http://localhost:$PORT http://httpbin.org/get > /dev/null ) &"
Enter
Sleep 200ms
Show

Sleep 2.5s

# HTTP request should appear now
Sleep 1s

# Make HTTPS request - shows lock icon
Hide
Type "( curl -s --proxy http://localhost:$PORT -k https://httpbin.org/headers > /dev/null ) &"
Enter
Sleep 200ms
Show

Sleep 2s

# HTTPS request appears with lock icon
Sleep 1s

# Make another HTTPS POST request
Hide
Type "( curl -s --proxy http://localhost:$PORT -k -X POST -d 'test=value' https://httpbin.org/post > /dev/null ) &"
Enter
Sleep 200ms
Show

Sleep 2s

# Navigate to the HTTPS GET request (should have lock icon)
Type "j"
Sleep 400ms

# Select and inspect this capture
Type "Enter"
Sleep 1s

# Show the request details loaded in the panel
Sleep 1.5s

# Go back to sidebar
Type "1"
Sleep 400ms

# Filter by POST method
Type "m"
Sleep 600ms
Type "m"
Sleep 800ms

# Shows only POST requests
Sleep 1s

# Clear filter
Type "x"
Sleep 600ms

# Show all captures again
Sleep 1s

# Clear all captures
Type "X"
Sleep 800ms

# Show empty capture view
Sleep 1s

# Quit
Type "q"
Sleep 300ms
