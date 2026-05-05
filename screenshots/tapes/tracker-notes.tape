Output screenshots/tracker-notes.gif

Set Shell "powershell"
Set FontSize 13
Set Width 1400
Set Height 800
Set TypingSpeed 100ms

Sleep 100ms

Type ".\tetrackt.exe"
Enter

Sleep 3000ms

# Load quickstart for a clean blank canvas to enter notes into
Ctrl+l
Sleep 500ms
Enter

Sleep 1500ms

Screenshot screenshots/tracker.png

# Enter EDIT mode — cursor changes to show we're ready for note input
Space
Sleep 400ms

# Enter a short melody on track 1 using the upper QWERTY row
# Q=C  W=D  E=E  R=F  T=G  Y=A  U=B
Type "q"
Sleep 300ms
Type "w"
Sleep 300ms
Type "e"
Sleep 300ms
Type "r"
Sleep 300ms
Type "t"
Sleep 300ms
Type "y"
Sleep 300ms
Type "u"
Sleep 300ms

# Back to NAV mode, move to track 2 and add a bass line on lower row
Space
Sleep 300ms
Right
Sleep 300ms
Space
Sleep 300ms

# Z=C  X=D  C=E  V=F  B=G
Type "z"
Sleep 300ms
Type "z"
Sleep 300ms
Type "x"
Sleep 300ms
Type "x"
Sleep 300ms
Type "c"
Sleep 300ms
Type "c"
Sleep 300ms

# Back to NAV, return to track 1, navigate to the FX subcolumn
Space
Sleep 300ms
Left
Sleep 300ms
Up
Up
Up
Up
Up
Up
Up
Up
Up
Up
Sleep 300ms

# Tab twice: Note → Vol → FX
Tab
Sleep 200ms
Tab
Sleep 300ms

# Type a vibrato effect: V = vibrato, 84 = speed 8 depth 4
Type "v"
Sleep 200ms
Type "8"
Sleep 200ms
Type "4"
Sleep 500ms

# Move down one row and add a portamento
Down
Sleep 200ms
Type "p"
Sleep 200ms
Type "0"
Sleep 200ms
Type "8"
Sleep 500ms

# Go back to Note column, row 0
Shift+Tab
Shift+Tab
Sleep 200ms
Up
Up
Up
Sleep 300ms

# Open the effects panel dialog to show all FX options at once
Ctrl+e
Sleep 800ms

# Navigate through the dialog fields to show them
Down
Sleep 400ms
# Show off some pitch effects
Right
Sleep 400ms

Screenshot screenshots/tracker-effects.png

Right
Sleep 400ms
Right
Sleep 400ms
Right
Sleep 400ms

# Cycle through Volume and Timing Effects
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 500ms

# Close dialog
Escape
Sleep 500ms
