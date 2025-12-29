# Currier Environment Switcher Demo
# Shows: V to open, j/k navigate, Enter to select, Esc to cancel

Output demos/demo-environment.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Type "currier"
Enter
Sleep 2s

# Show the environment switcher
Type "V"
Sleep 1s

# Navigate environments
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "k"
Sleep 500ms

# Select an environment
Enter
Sleep 1s

# Open again to show selection
Type "V"
Sleep 1s

# Cancel with Escape
Escape
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
