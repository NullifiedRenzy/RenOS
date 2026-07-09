pub fn data_dir() -> &'static str {
    "C:\\ProgramData\\RenOS"
}

pub fn ensure_dirs() -> std::io::Result<()> {
    std::fs::create_dir_all(format!("{}\\plugins", data_dir()))?;
    std::fs::create_dir_all(format!("{}\\cache", data_dir()))?;
    Ok(())
}
