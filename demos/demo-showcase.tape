# Currier Showcase Demo for Reddit
# A complete workflow showing what makes Currier feature-complete:
# 1. Create and send a request
# 2. Vim keybindings (j/k/gg/G)
# 3. Environment switching
# 4. Add headers
# 5. Collection runner with test results
# 6. History view

Output demos/demo-showcase.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set TypingSpeed 25ms
Set Padding 10

# Start currier
Type "currier"
Enter
Sleep 1.5s

# ============================================
# PART 1: Create and send a GET request
# ============================================

# New request
Type "n"
Sleep 400ms

# Type URL
Type "https://httpbin.org/get"
Sleep 400ms

# Send with Alt+Enter (while editing)
Alt+Enter
Sleep 2.2s

# ============================================
# PART 2: Vim navigation in response
# ============================================

# Move to response pane
Type "3"
Sleep 400ms

# Scroll with j/k
Type "j"
Sleep 120ms
Type "j"
Sleep 120ms
Type "j"
Sleep 120ms
Type "j"
Sleep 120ms
Type "j"
Sleep 200ms

# Jump to bottom with G
Type "G"
Sleep 400ms

# Jump to top with gg
Type "gg"
Sleep 500ms

# ============================================
# PART 3: Make a POST with headers
# ============================================

# New request
Type "n"
Sleep 400ms
Type "https://httpbin.org/post"
Sleep 300ms
Escape
Sleep 200ms

# Change to POST
Type "m"
Sleep 400ms

# Go to Headers tab
Type "]"
Sleep 250ms

# Add a custom header
Type "a"
Sleep 300ms
Type "X-Custom-Header"
Tab
Type "currier-demo"
Sleep 300ms
Escape
Sleep 300ms

# Go to Body tab
Type "]"
Sleep 250ms
Type "]"
Sleep 300ms

# Set JSON body type
Type "t"
Sleep 400ms

# Edit body
Type "e"
Sleep 200ms
Type@40ms '{"tool": "currier", "feature": "complete"}'
Sleep 300ms
Escape
Sleep 300ms

# Send POST
Enter
Sleep 2.2s

# View response
Type "3"
Sleep 400ms
Type "j"
Sleep 100ms
Type "j"
Sleep 100ms
Type "j"
Sleep 100ms
Type "j"
Sleep 400ms

# ============================================
# PART 4: Environment switching
# ============================================

# Open environment switcher
Type "V"
Sleep 700ms

# Navigate environments
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "k"
Sleep 300ms

# Select environment
Enter
Sleep 600ms

# ============================================
# PART 5: Save to collection
# ============================================

# Save current request
Type "s"
Sleep 500ms
Type "Demo Collection"
Enter
Sleep 700ms

# ============================================
# PART 6: Collection Runner
# ============================================

# Go to collections pane
Type "1"
Sleep 400ms

# Navigate to the collection
Type "j"
Sleep 200ms

# Expand collection
Type "l"
Sleep 400ms

# Go back to collection level
Type "h"
Sleep 300ms

# Run collection with Ctrl+R
Ctrl+R
Sleep 800ms

# Watch runner execute
Sleep 1.5s
Sleep 1.5s

# View results
Sleep 800ms

# Navigate results
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms

# Close runner
Escape
Sleep 500ms

# ============================================
# PART 7: History view
# ============================================

# Switch to History
Type "H"
Sleep 600ms

# Navigate history with j/k
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms

# Show search works
Type "/"
Sleep 300ms
Type "post"
Sleep 500ms
Escape
Sleep 400ms

# Back to Collections
Type "C"
Sleep 500ms

# ============================================
# PART 8: Help overlay
# ============================================

# Show help
Type "?"
Sleep 1.8s

# Close help
Type "?"
Sleep 400ms

# Quit
Type "q"
Sleep 300ms
