mod config;
mod enroll;
mod grpc_client;
mod health;
mod identity;
mod platform;
mod proto;

use std::path::PathBuf;
use std::sync::Arc;

use clap::Parser;
use tokio::sync::Notify;
use tracing_subscriber::EnvFilter;

#[derive(Parser)]
#[command(name = "renos-agent", version = "0.1.0", about = "RenOS Agent — lightweight device management daemon")]
struct Cli {
    #[command(subcommand)]
    command: Option<Command>,
}

#[derive(clap::Subcommand)]
enum Command {
    /// Enroll this device with a RenOS Core instance
    Enroll {
        /// One-time enrollment token
        #[arg(long)]
        token: String,
        /// Core server URL (e.g., https://renos.local:8443)
        #[arg(long)]
        core: Option<String>,
    },
    /// Run the agent daemon (default)
    Run,
    /// Print version information
    Version,
}

fn load_device_secret() -> Option<String> {
    let path = PathBuf::from(platform::resolve_data_dir()).join("device-secret.txt");
    std::fs::read_to_string(path).ok().map(|s| s.trim().to_string())
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .json()
        .with_env_filter(EnvFilter::from_default_env())
        .init();

    let cli = Cli::parse();

    match cli.command.unwrap_or(Command::Run) {
        Command::Enroll { token, core } => {
            let mut cfg = config::load();
            if let Some(core_url) = core {
                cfg.api_url = core_url;
            }
            tracing::info!("enrolling device with Core at {}", cfg.api_url);

            if let Err(e) = platform::ensure_dirs() {
                tracing::warn!("failed to create platform directories: {}", e);
            }

            let (identity, key) = identity::load_or_generate();

            match enroll::enroll(&cfg, &identity, &key, &token).await {
                Ok(result) => {
                    tracing::info!(
                        "enrollment successful — device_id={}, cert expires at {}",
                        result.device_id,
                        result.cert_expiry_unix
                    );
                    println!("Enrolled successfully!");
                    println!("  Device ID: {}", result.device_id);
                    println!("  Device secret stored in: {}/device-secret.txt", platform::resolve_data_dir());
                }
                Err(e) => {
                    tracing::error!("enrollment failed: {}", e);
                    eprintln!("Enrollment failed: {}", e);
                    std::process::exit(1);
                }
            }
        }
        Command::Version => {
            println!("renos-agent v{}", env!("CARGO_PKG_VERSION"));
            println!("build: {}", env!("CARGO_PKG_NAME"));
            println!("target: {}-{}", std::env::consts::ARCH, std::env::consts::OS);
        }
        Command::Run => {
            let cfg = config::load();
            tracing::info!("starting RenOS Agent v{}", env!("CARGO_PKG_VERSION"));

            if let Err(e) = platform::ensure_dirs() {
                tracing::warn!("failed to create platform directories: {}", e);
            }

            let (identity, _key) = identity::load_or_generate();
            tracing::info!("device identity: {} (enrolled: {})", identity.device_id, identity.is_enrolled);

            if identity.is_enrolled {
                let device_secret = load_device_secret()
                    .expect("device is enrolled but no device-secret.txt found — re-enroll the agent");

                let mut client = grpc_client::Client::new(&cfg);
                client.set_device_id(identity.device_id.clone());
                client.set_device_secret(device_secret);

                match client.connect_to_core().await {
                    Ok(_) => tracing::info!("connected to Core"),
                    Err(e) => tracing::warn!("failed to connect to Core: {}", e),
                }

                let hb_interval = std::time::Duration::from_secs(cfg.heartbeat_interval_secs);
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(hb_interval);
                    loop {
                        interval.tick().await;
                        let health_data = health::collect();
                        if let Err(e) = client.send_heartbeat(&health_data).await {
                            tracing::warn!("heartbeat failed: {}", e);
                        }
                    }
                });
            } else {
                tracing::warn!("device not enrolled — run `renos-agent enroll --token <token>` first");
            }

            let notify = Arc::new(Notify::new());
            let notify_clone = notify.clone();

            tokio::spawn(async move {
                wait_for_shutdown().await;
                tracing::info!("shutdown signal received");
                notify_clone.notify_one();
            });

            tracing::info!("agent running — waiting for shutdown signal");
            notify.notified().await;

            tracing::info!("shutdown complete");
        }
    }
}

async fn wait_for_shutdown() {
    #[cfg(unix)]
    {
        use tokio::signal::unix::{signal, SignalKind};
        let mut term =
            signal(SignalKind::terminate()).expect("failed to install SIGTERM handler");
        tokio::select! {
            _ = tokio::signal::ctrl_c() => {},
            _ = term.recv() => {},
        }
    }
    #[cfg(not(unix))]
    {
        tokio::signal::ctrl_c()
            .await
            .expect("failed to listen for ctrl-c");
    }
}
