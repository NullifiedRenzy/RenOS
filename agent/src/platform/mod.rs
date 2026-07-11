// Platform-specific implementations.
// Each module is conditionally compiled based on target OS.
//
// Data directory resolution:
//   The agent stores identity keys, certificates, and the device secret
//   under a single data directory.  The directory is resolved by
//   resolve_data_dir() in this order:
//
//     1. $RENOS_DATA_DIR environment variable (if set)
//     2. Platform default:
//        - Linux:   /var/lib/renos
//        - macOS:   /Library/Application Support/com.renos.agent
//        - Windows: %PROGRAMDATA%\RenOS\agent
//
//   Files stored in this directory:
//     identity.json      — device UUID + enrollment flag
//     device-key.pem     — Ed25519 private key (PEM)
//     device-cert.pem    — device certificate (PEM)
//     ca-cert.pem        — Core CA certificate (PEM)
//     device-secret.txt  — shared secret used for HTTP auth (until mTLS)
//
//   If RENOS_DATA_DIR is not set and the platform default is not writable,
//   identity generation and enrollment will fail silently.  Always set
//   RENOS_DATA_DIR to a writable path when running the agent as a
//   non-root user on Linux.

#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "macos")]
mod macos;

#[cfg(target_os = "windows")]
mod windows;

pub fn data_dir() -> &'static str {
    #[cfg(target_os = "linux")]
    {
        linux::data_dir()
    }
    #[cfg(target_os = "macos")]
    {
        macos::data_dir()
    }
    #[cfg(target_os = "windows")]
    {
        windows::data_dir()
    }
}

pub fn resolve_data_dir() -> String {
    std::env::var("RENOS_DATA_DIR").unwrap_or_else(|_| data_dir().to_string())
}

pub fn ensure_dirs() -> std::io::Result<()> {
    let dir = resolve_data_dir();
    std::fs::create_dir_all(format!("{dir}/plugins"))?;
    std::fs::create_dir_all(format!("{dir}/cache"))?;
    Ok(())
}
