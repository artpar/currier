# Currier Collection Runner Demo
# Shows: Create request with tests, save to collection, run collection, view test results

Output demos/demo-runner.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 25ms

Type "currier"
Enter
Sleep 1.5s

# Create a new request
Type "n"
Sleep 400ms
Type "https://httpbin.org/get"
Sleep 300ms
Escape
Sleep 200ms

# Go to Tests tab (5 times ])
Type "]"
Sleep 150ms
Type "]"
Sleep 150ms
Type "]"
Sleep 150ms
Type "]"
Sleep 150ms
Type "]"
Sleep 300ms

# Add test script
Type "e"
Sleep 200ms
Type@30ms 'pm.test("Status 200", function() {'
Enter
Type@30ms '  pm.expect(pm.response.status).toBe(200);'
Enter
Type@30ms '});'
Sleep 300ms
Escape
Sleep 300ms

# Save to collection
Type "s"
Sleep 500ms
Type "Test Collection"
Enter
Sleep 600ms

# Switch to Collections panel
Type "1"
Sleep 400ms

# Navigate to the collection
Type "j"
Sleep 200ms

# Expand to show requests
Type "l"
Sleep 400ms

# Go back up to collection root
Type "h"
Sleep 300ms

# Run the collection with Ctrl+R
Ctrl+R
Sleep 700ms

# Watch the runner execute
Sleep 1.5s
Sleep 1.5s

# Results displayed - show test pass/fail
Sleep 800ms

# Navigate results
Type "j"
Sleep 300ms

# Close the runner modal
Escape
Sleep 500ms

# Quit
Type "q"
Sleep 300ms
