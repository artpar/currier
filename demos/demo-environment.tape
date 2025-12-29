# Currier Environment Switcher Demo
# Shows: V to open, j/k navigate, Enter to select, Esc to cancel

Output demos/demo-environment.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1

Type "currier"
Enter
Sleep 1s

# Show the environment switcher
Type "V"
Sleep 600ms

# Navigate environments
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "k"
Sleep 300ms

# Select an environment
Enter
Sleep 500ms

# Open again to show selection
Type "V"
Sleep 600ms

# Cancel with Escape
Escape
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
