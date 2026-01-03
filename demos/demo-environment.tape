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
Sleep 2500ms

# Create a request with environment variable
Type "n"
Sleep 800ms
Type "https://{{host}}/get"
Sleep 1200ms
Escape
Sleep 800ms

# Show the variable syntax in the URL
Sleep 1000ms

# Open environment switcher
Type "V"
Sleep 1500ms

# Navigate through available environments
Type "j"
Sleep 800ms
Type "j"
Sleep 800ms
Type "k"
Sleep 800ms

# Select an environment
Enter
Sleep 1500ms

# Show that the environment is now active (visible in status)
Sleep 1000ms

# The URL still shows {{host}} but will be interpolated on send
# Let's send the request to see interpolation in action
Enter
Sleep 3000ms

# View the response - the actual URL used is shown
Type "3"
Sleep 1500ms

# Scroll to see the resolved URL in response
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1200ms

# Switch to a different environment
Type "V"
Sleep 1500ms

Type "j"
Sleep 800ms
Enter
Sleep 1500ms

# Send again - different environment, different host
Enter
Sleep 3000ms

# View response to confirm different endpoint was hit
Type "3"
Sleep 1500ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 1200ms

# Quit
Type "q"
Sleep 500ms
