# Currier Environment Switcher Demo
# Shows: Complete cycle - switch environment, see variable interpolation in action

Output demos/demo-environment.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms

Type "currier"
Enter
Sleep 1.5s

# Create a request with environment variable
Type "n"
Sleep 400ms
Type "https://{{host}}/get"
Sleep 600ms
Escape
Sleep 400ms

# Open environment switcher
Type "V"
Sleep 700ms

# Navigate through environments
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms
Type "k"
Sleep 350ms

# Select an environment
Enter
Sleep 600ms

# Send request to see interpolation
Enter
Sleep 2s

# View the response
Type "3"
Sleep 500ms

# Scroll to see the resolved URL
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 500ms

# Switch to a different environment
Type "V"
Sleep 700ms
Type "j"
Sleep 350ms
Enter
Sleep 600ms

# Send again
Enter
Sleep 2s

# View response
Type "3"
Sleep 500ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 500ms

# Quit
Type "q"
Sleep 300ms
