// Platform-specific implementations.
// Each module is conditionally compiled based on target OS.

#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "macos")]
mod macos;

#[cfg(target_os = "windows")]
mod windows;

pub fn data_dir() -> &'static str {
    #[cfg(target_os = "linux")]
    { linux::data_dir() }
    #[cfg(target_os = "macos")]
    { macos::data_dir() }
    #[cfg(target_os = "windows")]
    { windows::data_dir() }
}

pub fn resolve_data_dir() -> String {
    std::env::var("RENOS_DATA_DIR").unwrap_or_else(|_| data_dir().to_string())
}

pub fn ensure_dirs() -> std::io::Result<()> {
    #[cfg(target_os = "linux")]
    { linux::ensure_dirs() }
    #[cfg(target_os = "macos")]
    { macos::ensure_dirs() }
    #[cfg(target_os = "windows")]
    { windows::ensure_dirs() }
}
