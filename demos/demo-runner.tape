# Currier Collection Runner Demo
# Shows: Ctrl+R to run collection, progress display, results summary

Output demos/demo-runner.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Switch to Collections panel
Type "C"
Sleep 400ms

# Expand the first collection
Type "l"
Sleep 300ms

# Run collection with Ctrl+R
Ctrl+R
Sleep 3s

# Close the runner modal with Escape
Escape
Sleep 500ms

# Quit
Type "q"
Sleep 300ms
