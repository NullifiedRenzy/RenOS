use std::collections::HashMap;
use std::io::{Read, Write};

use portable_pty::{CommandBuilder, PtySize, native_pty_system, MasterPty};
use tokio::sync::{mpsc, oneshot};

enum PtyCommand {
    Spawn {
        session_id: String,
        cols: u16,
        rows: u16,
        shell: String,
        reply: oneshot::Sender<Result<mpsc::Receiver<Vec<u8>>, String>>,
    },
    WriteInput {
        session_id: String,
        data: Vec<u8>,
        reply: oneshot::Sender<Result<(), String>>,
    },
    Resize {
        session_id: String,
        cols: u16,
        rows: u16,
        reply: oneshot::Sender<Result<(), String>>,
    },
    Close {
        session_id: String,
        reply: oneshot::Sender<Result<(), String>>,
    },
}

struct PtySession {
    #[allow(dead_code)]
    master: Box<dyn MasterPty + Send>,
    writer: Box<dyn Write + Send>,
}

struct PtyManagerInner {
    sessions: HashMap<String, PtySession>,
}

impl PtyManagerInner {
    fn handle_spawn(
        &mut self,
        session_id: String,
        cols: u16,
        rows: u16,
        shell: String,
    ) -> Result<mpsc::Receiver<Vec<u8>>, String> {
        let pty_system = native_pty_system();

        let size = PtySize {
            rows,
            cols,
            pixel_width: 0,
            pixel_height: 0,
        };

        let pair = pty_system
            .openpty(size)
            .map_err(|e| format!("failed to open PTY: {e}"))?;

        let shell_cmd = if shell.is_empty() {
            detect_shell()
        } else {
            shell
        };

        let mut cmd = CommandBuilder::new(&shell_cmd);
        cmd.cwd(detect_cwd());
        cmd.env("TERM", "xterm-256color");
        cmd.env("LANG", "en_US.UTF-8");
        cmd.env("LC_ALL", "UTF-8");

        let mut child = pair
            .slave
            .spawn_command(cmd)
            .map_err(|e| format!("failed to spawn shell: {e}"))?;

        let writer = pair
            .master
            .take_writer()
            .map_err(|e| format!("failed to get PTY writer: {e}"))?;

        let mut reader = pair
            .master
            .try_clone_reader()
            .map_err(|e| format!("failed to get PTY reader: {e}"))?;

        let (tx, rx) = mpsc::channel::<Vec<u8>>(256);

        // Reader task — reads PTY output and sends to channel
        let sid = session_id.clone();
        std::thread::spawn(move || {
            let mut buf = [0u8; 8192];
            loop {
                match reader.read(&mut buf) {
                    Ok(0) => {
                        tracing::info!("PTY EOF for session {}", sid);
                        break;
                    }
                    Ok(n) => {
                        if tx.blocking_send(buf[..n].to_vec()).is_err() {
                            break;
                        }
                    }
                    Err(e) => {
                        tracing::warn!("PTY read error for session {}: {}", sid, e);
                        break;
                    }
                }
            }
        });

        // Child reaper
        let sid2 = session_id.clone();
        std::thread::spawn(move || {
            let _ = child.wait();
            tracing::info!("PTY child exited for session {}", sid2);
        });

        self.sessions.insert(
            session_id,
            PtySession {
                master: pair.master,
                writer,
            },
        );

        Ok(rx)
    }

    fn handle_input(&mut self, session_id: &str, data: &[u8]) -> Result<(), String> {
        let session = self
            .sessions
            .get_mut(session_id)
            .ok_or_else(|| format!("session {} not found", session_id))?;

        session
            .writer
            .write_all(data)
            .map_err(|e| format!("PTY write error: {e}"))?;
        session
            .writer
            .flush()
            .map_err(|e| format!("PTY flush error: {e}"))?;
        Ok(())
    }

    fn handle_resize(&self, session_id: &str, cols: u16, rows: u16) -> Result<(), String> {
        let session = self
            .sessions
            .get(session_id)
            .ok_or_else(|| format!("session {} not found", session_id))?;

        let size = PtySize {
            rows,
            cols,
            pixel_width: 0,
            pixel_height: 0,
        };

        session
            .master
            .resize(size)
            .map_err(|e| format!("PTY resize error: {e}"))?;
        Ok(())
    }

    fn handle_close(&mut self, session_id: &str) -> Result<(), String> {
        if self.sessions.remove(session_id).is_some() {
            tracing::info!("PTY session {} closed", session_id);
            Ok(())
        } else {
            Err(format!("session {} not found", session_id))
        }
    }
}

/// Manages all active PTY sessions via a command channel
#[derive(Clone)]
pub struct PtyManager {
    cmd_tx: mpsc::Sender<PtyCommand>,
}

impl PtyManager {
    pub fn new() -> Self {
        let (cmd_tx, cmd_rx) = mpsc::channel::<PtyCommand>(64);

        // Spawn the manager task that owns all sessions (single-threaded)
        tokio::spawn(async move {
            Self::run_manager(cmd_rx).await;
        });

        Self { cmd_tx }
    }

    async fn run_manager(mut cmd_rx: mpsc::Receiver<PtyCommand>) {
        let mut inner = PtyManagerInner {
            sessions: HashMap::new(),
        };

        while let Some(cmd) = cmd_rx.recv().await {
            match cmd {
                PtyCommand::Spawn {
                    session_id,
                    cols,
                    rows,
                    shell,
                    reply,
                } => {
                    let result = inner.handle_spawn(session_id, cols, rows, shell);
                    let _ = reply.send(result);
                }
                PtyCommand::WriteInput {
                    session_id,
                    data,
                    reply,
                } => {
                    let result = inner.handle_input(&session_id, &data);
                    let _ = reply.send(result);
                }
                PtyCommand::Resize {
                    session_id,
                    cols,
                    rows,
                    reply,
                } => {
                    let result = inner.handle_resize(&session_id, cols, rows);
                    let _ = reply.send(result);
                }
                PtyCommand::Close { session_id, reply } => {
                    let result = inner.handle_close(&session_id);
                    let _ = reply.send(result);
                }
            }
        }

        tracing::warn!("PTY manager task ended");
    }

    pub async fn spawn(
        &self,
        session_id: String,
        cols: u16,
        rows: u16,
        shell: String,
    ) -> Result<mpsc::Receiver<Vec<u8>>, String> {
        let (reply_tx, reply_rx) = oneshot::channel();
        let _ = self
            .cmd_tx
            .send(PtyCommand::Spawn {
                session_id,
                cols,
                rows,
                shell,
                reply: reply_tx,
            })
            .await;
        reply_rx
            .await
            .map_err(|_| "pty manager dropped".to_string())?
    }

    pub async fn write_input(&self, session_id: &str, data: &[u8]) -> Result<(), String> {
        let (reply_tx, reply_rx) = oneshot::channel();
        let _ = self
            .cmd_tx
            .send(PtyCommand::WriteInput {
                session_id: session_id.to_string(),
                data: data.to_vec(),
                reply: reply_tx,
            })
            .await;
        reply_rx
            .await
            .map_err(|_| "pty manager dropped".to_string())?
    }

    pub async fn resize(&self, session_id: &str, cols: u16, rows: u16) -> Result<(), String> {
        let (reply_tx, reply_rx) = oneshot::channel();
        let _ = self
            .cmd_tx
            .send(PtyCommand::Resize {
                session_id: session_id.to_string(),
                cols,
                rows,
                reply: reply_tx,
            })
            .await;
        reply_rx
            .await
            .map_err(|_| "pty manager dropped".to_string())?
    }

    pub async fn close(&self, session_id: &str) -> Result<(), String> {
        let (reply_tx, reply_rx) = oneshot::channel();
        let _ = self
            .cmd_tx
            .send(PtyCommand::Close {
                session_id: session_id.to_string(),
                reply: reply_tx,
            })
            .await;
        reply_rx
            .await
            .map_err(|_| "pty manager dropped".to_string())?
    }
}

fn detect_shell() -> String {
    if let Ok(shell) = std::env::var("SHELL") {
        if !shell.is_empty() {
            return shell;
        }
    }
    #[cfg(target_os = "windows")]
    {
        "powershell.exe".to_string()
    }
    #[cfg(not(target_os = "windows"))]
    {
        for shell in &["/bin/bash", "/bin/sh", "/usr/bin/bash", "/usr/bin/sh"] {
            if std::path::Path::new(shell).exists() {
                return shell.to_string();
            }
        }
        "/bin/sh".to_string()
    }
}

fn detect_cwd() -> std::path::PathBuf {
    std::env::current_dir().unwrap_or_else(|_| {
        #[cfg(not(target_os = "windows"))]
        {
            std::path::PathBuf::from("/root")
        }
        #[cfg(target_os = "windows")]
        {
            std::env::var("USERPROFILE")
                .map(std::path::PathBuf::from)
                .unwrap_or_else(|_| std::path::PathBuf::from("C:\\"))
        }
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_detect_shell() {
        let shell = detect_shell();
        assert!(!shell.is_empty());
    }

    #[test]
    fn test_detect_cwd() {
        let cwd = detect_cwd();
        assert!(cwd.exists());
    }
}
