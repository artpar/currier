# Currier Environment Switcher Demo
# Shows: Environment switching with variable preview, editing variables

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

# Open environment switcher - shows variables on the right
Type "V"
Sleep 800ms

# Navigate through environments - watch variables update
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "k"
Sleep 600ms

# Edit the selected environment
Type "e"
Sleep 700ms

# Navigate in the editor
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Edit a variable value
Type "e"
Sleep 200ms
Type "httpbin.org"
Sleep 400ms
Escape
Sleep 400ms

# Close editor
Escape
Sleep 500ms

# Select the environment
Enter
Sleep 600ms

# Send request to see interpolation
Enter
Sleep 2.2s

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

# Quit
Type "q"
Sleep 300ms
