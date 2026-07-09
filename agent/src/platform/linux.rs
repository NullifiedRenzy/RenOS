pub fn data_dir() -> &'static str {
    "/var/lib/renos"
}

#[allow(dead_code)]
pub fn ensure_dirs() -> std::io::Result<()> {
    Ok(())
}
