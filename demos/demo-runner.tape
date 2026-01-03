# Currier Collection Runner Demo
# Shows: Complete cycle - run collection, watch progress, see test results

Output demos/demo-runner.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 2500ms

# Switch to Collections panel
Type "C"
Sleep 1500ms

# Navigate to a collection
Type "j"
Sleep 600ms

# Expand the collection to show requests
Type "l"
Sleep 1200ms

# Show requests in collection
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 800ms

# Go back up to collection root
Type "h"
Sleep 800ms

# Run the collection with Ctrl+R
Ctrl+R
Sleep 1500ms

# Watch the runner execute requests
# Progress indicator shows current request
Sleep 1500ms
Sleep 1500ms
Sleep 1500ms

# Runner completes - results are displayed
Sleep 2000ms

# Results show:
# - Each request with pass/fail status
# - Response times
# - Test assertion results
# - Summary: X/Y passed, total time

# Scroll through results if needed
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "j"
Sleep 1000ms

# Back to top
Type "gg"
Sleep 1000ms

# View detailed results
Sleep 1500ms

# Close the runner modal
Escape
Sleep 1500ms

# Quit
Type "q"
Sleep 500ms
