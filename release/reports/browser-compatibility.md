# RenOS 1.0.0 — Browser Compatibility Report

| Field | Value |
|---|---|
| **Product** | RenOS Hub UI |
| **Version** | 1.0.0-RELEASE |
| **Test Date** | 2026-05-18 – 2026-05-22 |
| **Devices** | Desktop (multiple OS), Android, iOS |
| **Result** | **All modern browsers supported** |

---

## Test Matrix

Legend: ✅ Full support (all features pass), ⚠️ Partial support (minor visual/behavioral differences), ❌ Not supported.

### Dashboard

| Feature | Chrome 120+ | Firefox 120+ | Safari 17+ | Edge 120+ | Chrome Android | Safari iOS |
|---|---|---|---|---|---|---|
| Layout rendering | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Real-time device status | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Alert summary cards | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Performance charts | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Quick action buttons | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Responsive layout | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Dark mode | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### WebSocket / SSE

| Feature | Chrome 120+ | Firefox 120+ | Safari 17+ | Edge 120+ | Chrome Android | Safari iOS |
|---|---|---|---|---|---|---|
| WebSocket connection | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Reconnection logic | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| SSE events | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Binary frame handling | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| TLS/WSS | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### Terminal (xterm.js)

| Feature | Chrome 120+ | Firefox 120+ | Safari 17+ | Edge 120+ | Chrome Android | Safari iOS |
|---|---|---|---|---|---|---|
| Terminal rendering | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ |
| Keyboard input | ✅ | ✅ | ✅ | ✅ | ⚠️ | ⚠️ |
| Copy/paste | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Font ligatures | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 256-color support | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Resize handling | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Search | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Full-screen mode | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### Workflow Editor

| Feature | Chrome 120+ | Firefox 120+ | Safari 17+ | Edge 120+ | Chrome Android | Safari iOS |
|---|---|---|---|---|---|---|
| Visual drag-and-drop | ✅ | ✅ | ✅ | ✅ | ⚠️ | ⚠️ |
| Step configuration forms | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Connection lines | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Undo/redo | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Keyboard reorder buttons | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Auto-save indicator | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Template browser | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

### File Uploads

| Feature | Chrome 120+ | Firefox 120+ | Safari 17+ | Edge 120+ | Chrome Android | Safari iOS |
|---|---|---|---|---|---|---|
| File picker dialog | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Drag-and-drop upload | ✅ | ✅ | ✅ | ✅ | ⚠️ | ⚠️ |
| Multi-file selection | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Progress indication | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Chunked upload resumption | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Large file (1 GB+) | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ |

### Authentication

| Feature | Chrome 120+ | Firefox 120+ | Safari 17+ | Edge 120+ | Chrome Android | Safari iOS |
|---|---|---|---|---|---|---|
| Login form | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| SSO redirect | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| OAuth2 flows | ✅ | ✅ | ⚠️ | ✅ | ✅ | ⚠️ |
| Session persistence | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Logout | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Password managers | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Biometric (WebAuthn) | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ |

### Responsive Design

| Feature | Chrome 120+ | Firefox 120+ | Safari 17+ | Edge 120+ | Chrome Android | Safari iOS |
|---|---|---|---|---|---|---|
| 1920×1080 (desktop) | ✅ | ✅ | ✅ | ✅ | — | — |
| 1366×768 (laptop) | ✅ | ✅ | ✅ | ✅ | — | — |
| 1024×768 (tablet) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 768×1024 (tablet portrait) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 414×896 (mobile) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 375×667 (mobile small) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Sidebar collapse | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Touch targets ≥ 44 px | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| No horizontal scroll | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ |

### Dark Mode

| Feature | Chrome 120+ | Firefox 120+ | Safari 17+ | Edge 120+ | Chrome Android | Safari iOS |
|---|---|---|---|---|---|---|
| `prefers-color-scheme` detection | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Manual toggle override | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| All components styled | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Chart colors invert | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Terminal theme sync | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Contrast ratios maintained | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## Detailed Browser Notes

### Chrome 120+ (Linux, Windows, macOS)

All features pass testing. No compatibility issues identified.

### Firefox 120+ (Linux, Windows, macOS)

All features pass testing. No compatibility issues identified. Notable: Firefox's implementation of the `:has()` CSS selector (used in sidebar styling) is fully supported since Firefox 121.

### Safari 17+ (macOS 14+)

| Issue | Status | Details |
|---|---|---|
| Large file uploads (1 GB+) | ⚠️ Partial | Safari's blob storage imposes a per-origin quota. Files exceeding ~500 MB may fail on systems with constrained disk space. Users are advised to use Chrome or Firefox for transfers exceeding 500 MB. |
| OAuth2 popup windows | ⚠️ Partial | Safari's Intelligent Tracking Prevention (ITP) may block third-party cookie setting in OAuth2 popup flows. The Hub falls back to redirect-based flow when popup detection fails. |
| WebAuthn / Passkeys | ⚠️ Partial | Safari 17+ supports WebAuthn but requires user gesture for `navigator.credentials.create()` calls. The login flow queues the credential creation after a click event. |

### Edge 120+ (Windows, macOS)

All features pass testing. Edge's IE mode ("Internet Explorer mode") is **not supported** and will show a deprecation banner.

### Chrome Android (Android 12+)

| Issue | Status | Details |
|---|---|---|
| Terminal keyboard input | ⚠️ Partial | On-screen keyboard obscures terminal viewport. The terminal auto-scrolls into view when focused. Physical keyboard (Bluetooth) works without issues. |
| Drag-and-drop (workflow) | ⚠️ Partial | Touch drag-and-drop works but precision is lower than desktop. "Move Up/Down" buttons provide a reliable alternative. |
| File drag-and-drop upload | ⚠️ Partial | Android Chrome does not support drag-and-drop from file managers. Use the file picker button instead. |

### Safari iOS (iOS 17+)

| Issue | Status | Details |
|---|---|---|
| Terminal rendering | ⚠️ Partial | iOS Safari limits `requestAnimationFrame` to 30 FPS in low-power mode, causing visible flicker during rapid terminal output. The terminal uses a coalescing buffer to mitigate this. |
| Terminal keyboard input | ⚠️ Partial | iOS keyboard does not send certain control sequences (e.g., F-keys). The Hub provides an on-screen control palette for common sequences. |
| OAuth2 redirect | ⚠️ Partial | Safari iOS may not preserve session state across OAuth2 redirects in private browsing mode. Users are prompted to re-authenticate. |
| Horizontal scroll | ⚠️ Partial | On iPhone SE (375px width), the workflow editor canvas may require horizontal scrolling in complex workflows. |
| Drag-and-drop (workflow) | ⚠️ Partial | iOS drag-and-drop has a long-press delay. "Move Up/Down" buttons recommended for workflow editing on iOS. |
| File drag-and-drop upload | ⚠️ Partial | Not supported on iOS. Use the file picker button. |
| Viewport height (100dvh) | ⚠️ Partial | Safari iOS 17.0–17.2 had a bug with `100dvh` where the address bar resize was not reflected. The Hub uses `100svh` with a fallback to JS-based height calculation. Fixed in iOS 17.3+. |

---

## Summary

| Browser | Version(s) | Support Level | Notes |
|---|---|---|---|
| Google Chrome | 120+ | ✅ **Full** | Primary development target |
| Mozilla Firefox | 120+ | ✅ **Full** | Some CSS features require Firefox 121+ |
| Apple Safari | 17+ | ⚠️ **Full with caveats** | Large uploads, OAuth2 popups have workarounds |
| Microsoft Edge | 120+ | ✅ **Full** | IE mode is unsupported |
| Chrome Android | Latest | ⚠️ **Full with caveats** | Terminal keyboard, drag-and-drop have workarounds |
| Safari iOS | 17+ | ⚠️ **Full with caveats** | Terminal, OAuth2, viewport have workarounds |

**All modern browsers are supported.** Safari (desktop and iOS) and Chrome Android have minor caveats documented above, all with available workarounds. No features are completely blocked on any tested browser.

---

## Unsupported Browsers

The following browsers are **not supported** and will display an unsupported-browser banner with a link to download a supported browser:

| Browser | Reason |
|---|---|
| Internet Explorer 11 | End-of-life, missing modern JS/CSS APIs |
| Safari < 17 | Missing `:has()`, `100dvh`, WebSocket keepalive |
| Chrome < 120 | Missing CSS subgrid, `scroll-timeline` |
| Firefox < 120 | Missing `:has()` selector (added in 121) |
| Edge Legacy (non-Chromium) | End-of-life |
| Samsung Internet < 22 | Missing modern CSS features |

---

## Device / OS Breakdown

### Desktop Screens Resolutions Tested

| Resolution | Aspect Ratio | Device |
|---|---|---|
| 1920×1080 | 16:9 | Full HD monitor |
| 2560×1440 | 16:9 | QHD monitor |
| 3840×2160 | 16:9 | 4K monitor (200% scaling) |
| 1366×768 | ~16:9 | Common laptop |
| 1440×900 | 16:10 | MacBook Air/Pro |
| 1680×1050 | 16:10 | External monitor |
| 1024×768 | 4:3 | Older monitors / projectors |

### Mobile / Tablet Devices Tested

| Device | OS | Screen | Browser |
|---|---|---|---|
| iPhone 15 Pro | iOS 17.5 | 393×852 | Safari, Chrome |
| iPhone SE (3rd gen) | iOS 17.5 | 375×667 | Safari |
| iPad Air (5th gen) | iOS 17.5 | 820×1180 | Safari |
| Samsung Galaxy S24 | Android 14 | 360×780 | Chrome |
| Samsung Galaxy Tab S9 | Android 14 | 800×1280 | Chrome |
| Google Pixel 8 | Android 15 | 412×915 | Chrome |
| OnePlus 12 | Android 14 | 412×896 | Chrome |

---

## Sign-Off

| Role | Name | Date |
|---|---|---|
| Frontend Lead | [Redacted] | 2026-05-22 |
| QA Lead | [Redacted] | 2026-05-22 |
| Release Manager | [Redacted] | 2026-05-22 |
