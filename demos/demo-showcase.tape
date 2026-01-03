# Currier Showcase Demo for Reddit
# A complete workflow showing what makes Currier feature-complete:
# 1. Create and send a request
# 2. Vim keybindings (j/k/gg/G)
# 3. Pre-request scripts
# 4. Test scripts and assertions
# 5. Environment switching with variable preview
# 6. Environment editor
# 7. Collection runner with test results
# 8. History view

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
Sleep 100ms
Type "j"
Sleep 100ms
Type "j"
Sleep 100ms
Type "j"
Sleep 150ms

# Jump to bottom with G
Type "G"
Sleep 350ms

# Jump to top with gg
Type "gg"
Sleep 400ms

# ============================================
# PART 3: Create POST with pre-request script and tests
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

# Go to Headers tab and add header
Type "]"
Sleep 200ms
Type "a"
Sleep 250ms
Type "X-Custom"
Tab
Type "demo"
Sleep 200ms
Escape
Sleep 250ms

# Go to Body tab (] twice more)
Type "]"
Sleep 200ms
Type "]"
Sleep 250ms

# Set JSON body type
Type "t"
Sleep 350ms

# Edit body
Type "e"
Sleep 150ms
Type@35ms '{"name": "currier"}'
Sleep 250ms
Escape
Sleep 250ms

# ============================================
# PART 4: Add Pre-request Script
# ============================================

# Go to Pre-Request tab (one more ])
Type "]"
Sleep 400ms

# Edit pre-request script
Type "e"
Sleep 200ms
Type@30ms 'console.log("Pre-request!");'
Enter
Type@30ms 'pm.environment.set("timestamp", Date.now());'
Sleep 400ms
Escape
Sleep 400ms

# ============================================
# PART 5: Add Test Script
# ============================================

# Go to Tests tab
Type "]"
Sleep 400ms

# Edit test script
Type "e"
Sleep 200ms
Type@30ms 'pm.test("Status is 200", function() {'
Enter
Type@30ms '  pm.response.to.have.status(200);'
Enter
Type@30ms '});'
Sleep 400ms
Escape
Sleep 400ms

# ============================================
# PART 6: Send and view test results
# ============================================

# Send POST request
Enter
Sleep 2.5s

# View response - go to Tests tab to see results
Type "3"
Sleep 400ms

# Navigate to Tests tab in response (5 times ])
Type "]"
Sleep 200ms
Type "]"
Sleep 200ms
Type "]"
Sleep 200ms
Type "]"
Sleep 200ms
Type "]"
Sleep 600ms

# Show test passed
Sleep 800ms

# ============================================
# PART 7: Environment switching with variable preview
# ============================================

# Open environment switcher
Type "V"
Sleep 700ms

# Navigate environments - variables update in right panel
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "k"
Sleep 500ms

# ============================================
# PART 8: Edit environment variables
# ============================================

# Press 'e' to edit the selected environment
Type "e"
Sleep 700ms

# Navigate in editor
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Edit a variable value
Type "e"
Sleep 200ms
# Add some text
Type "https://api.example.com"
Sleep 400ms
Escape
Sleep 400ms

# Save and close editor
Escape
Sleep 500ms

# Close environment switcher
Escape
Sleep 400ms

# ============================================
# PART 9: Save to collection
# ============================================

# Save current request
Type "s"
Sleep 500ms
Type "Demo Collection"
Enter
Sleep 600ms

# ============================================
# PART 10: Collection Runner
# ============================================

# Go to collections pane
Type "1"
Sleep 300ms

# Navigate to the collection
Type "j"
Sleep 200ms

# Expand collection
Type "l"
Sleep 300ms

# Go back to collection level
Type "h"
Sleep 250ms

# Run collection with Ctrl+R
Ctrl+R
Sleep 700ms

# Watch runner execute
Sleep 1.2s
Sleep 1.2s

# View results
Sleep 600ms

# Navigate results
Type "j"
Sleep 200ms

# Close runner
Escape
Sleep 400ms

# ============================================
# PART 11: History view
# ============================================

# Switch to History
Type "H"
Sleep 500ms

# Navigate history with j/k
Type "j"
Sleep 150ms
Type "j"
Sleep 150ms

# Show search works
Type "/"
Sleep 250ms
Type "post"
Sleep 400ms
Escape
Sleep 350ms

# Back to Collections
Type "C"
Sleep 400ms

# ============================================
# PART 12: Help overlay
# ============================================

# Show help
Type "?"
Sleep 1.5s

# Close help
Type "?"
Sleep 300ms

# Quit
Type "q"
Sleep 300ms
