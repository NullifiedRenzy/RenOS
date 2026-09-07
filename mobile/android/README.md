# RenOS Android

RenOS Android is a small native WebView shell for accessing a self-hosted RenOS
Hub from an Android phone or tablet.

This app is for mobile access to RenOS. It is not a privileged Android device
management agent.

## Build

```bash
cd mobile/android
gradle :app:assembleDebug
```

The release workflow publishes the debug-signed APK as `RenOS-Android.apk` so it
can be sideloaded for testing.

## Use

Install the APK, enter the RenOS Hub URL, and tap Open. The URL is saved on the
device for the next launch.

For the current local deployment, use:

```text
http://192.168.0.117
```
