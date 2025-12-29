# Currier Environment Variable Editing Demo
# Shows: Edit environment variables with the 'e' key

Output demos/demo-env-editing.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Type "currier"
Enter
Sleep 2s

# Open environment switcher with 'E'
Type "E"
Sleep 1s

# Navigate to an environment
Type "j"
Sleep 300ms

# Press 'e' to edit the selected environment
Type "e"
Sleep 1s

# Now in the environment editor
# Navigate down to see existing variables
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Press 'a' to add a new variable
Type "a"
Sleep 500ms

# Type the variable key
Type "api_key"
Sleep 500ms

# Press Tab to move to value input
Tab
Sleep 300ms

# Type the variable value
Type "sk-12345-secret-key"
Sleep 500ms

# Press Enter to confirm
Enter
Sleep 500ms

# Add another variable
Type "a"
Sleep 500ms
Type "timeout"
Tab
Sleep 300ms
Type "30000"
Enter
Sleep 500ms

# Navigate to a variable and edit it with 'e'
Type "j"
Sleep 300ms
Type "e"
Sleep 500ms

# Modify the value
Type "_modified"
Tab
Sleep 300ms
Enter
Sleep 500ms

# Press Escape to save and close
Escape
Sleep 1s

# Show the environment is now active with edited variables
Sleep 1s

# Quit
Type "q"
Sleep 500ms
