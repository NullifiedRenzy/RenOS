# Android WebView regression check

Run this flow in the native APK on an Android device or emulator. Desktop browser
mobile emulation does not exercise WebView focus zoom or Android window insets.

1. Open the server address and sign in with Keep me signed in selected.
2. Dismiss the keyboard. Confirm that the dashboard heading and device count are
   visible, with the address bar below the status bar and content above navigation.
3. Close the app completely and reopen it. The dashboard should remain visible
   without entering credentials again.
4. Edit the server address and press Open or the keyboard's Go action. The address
   keyboard should close, and the page should occupy the remaining screen.
5. Repeat in landscape and at a narrow phone width.

For a debuggable APK, inspect its WebView through Chrome DevTools. After login,
`visualViewport.scale` should be 1, `visualViewport.offsetTop` should be 0, and
`innerWidth` should match the WebView width in density-independent pixels.

The v1.0.12 regression was reproduced on Android 16 at 1080x2400 with density 420:
the page had `innerWidth = 1082`, with a visual viewport offset of about 670 CSS
pixels after focusing the login inputs. The dashboard was loaded but off-screen.
The corrected APK uses a 412 CSS pixel viewport and resets the offset to zero.
