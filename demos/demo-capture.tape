# Currier Traffic Capture Demo
# Shows: Switch to capture mode, start proxy, capture HTTP traffic, filter, clear

Output demos/demo-capture.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

# Start background processes that will make HTTP requests after delay
Type "( sleep 6 && curl -s --proxy http://localhost:8080 http://httpbin.org/get > /dev/null && sleep 1 && curl -s --proxy http://localhost:8080 http://httpbin.org/headers > /dev/null && sleep 1 && curl -s --proxy http://localhost:8080 -X POST http://httpbin.org/post > /dev/null ) &"
Enter
Sleep 300ms

Type "currier"
Enter
Sleep 1.5s

# Switch to Capture mode (C cycles: History -> Collections -> Capture)
Type "C"
Sleep 400ms
Type "C"
Sleep 600ms

# Start the proxy
Type "p"
Sleep 1s

# Wait for requests to come through (background curl running)
Sleep 3s

# First request should appear
Sleep 2s

# More requests coming in
Sleep 2s

# Navigate through captures
Type "j"
Sleep 300ms
Type "j"
Sleep 500ms

# Go back up
Type "k"
Sleep 300ms
Type "k"
Sleep 500ms

# Cycle method filter to show GET only
Type "m"
Sleep 800ms

# Cycle to POST
Type "m"
Sleep 800ms

# Clear filter
Type "x"
Sleep 600ms

# Clear all captures
Type "X"
Sleep 800ms

# Stop the proxy
Type "p"
Sleep 600ms

# Switch back to History mode
Type "H"
Sleep 600ms

# Quit
Type "q"
Sleep 300ms
