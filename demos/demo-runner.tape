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

# Focus on collections panel
Type "C"
Sleep 400ms

# Navigate to a collection
Type "j"
Sleep 200ms

# Run collection with Ctrl+R
Ctrl+R
Sleep 2s

# Close the runner modal with Enter
Enter
Sleep 500ms

# Quit
Type "q"
Sleep 300ms
