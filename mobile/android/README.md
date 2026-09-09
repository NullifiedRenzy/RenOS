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

The release workflow publishes a non-debuggable, release-signed APK as
`RenOS-Android.apk`. Every release must use the same private signing key.

## Release signing

GitHub Actions requires these repository secrets:

- `RENOS_ANDROID_KEYSTORE_BASE64`: base64-encoded PKCS12 keystore
- `RENOS_ANDROID_STORE_PASSWORD`
- `RENOS_ANDROID_KEY_ALIAS`
- `RENOS_ANDROID_KEY_PASSWORD`

For a local release build, set `RENOS_ANDROID_KEYSTORE` to the absolute keystore
path and supply the other three variables, then run:

```bash
gradle :app:assembleRelease :app:lintRelease
```

Missing credentials fail the release build; it never falls back to a debug key.
The release workflow checks the certificate against `release-certificate.sha256`
and refuses to publish if the configured key changes unexpectedly. CI tests use
an isolated temporary key, never the production signing secrets.
Do not commit the keystore or passwords. Keep a separate encrypted backup of the
keystore and its credentials: losing the key prevents in-place app updates.

Versions through v1.0.13 used disposable debug keys. Moving from those versions
to the first permanently signed release requires one final uninstall/reinstall
and login. Subsequent releases using this key can be installed over the existing
app without deleting saved settings or sessions. Do not uninstall for a normal
upgrade.

## Use

Install the APK, enter the RenOS Hub URL, and tap Open. The URL is saved on the
device for the next launch.

For the current local deployment, use:

```text
http://192.168.0.117:18081
```
