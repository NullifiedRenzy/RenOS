mod commands;
mod config;
mod docker;
mod docker_actions;
mod docker_images;
mod docker_networks;
mod docker_volumes;
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
            println!("data dir: {}", platform::resolve_data_dir());
        }
        Command::Run => {
            let cfg = config::load();
            let data_dir = platform::resolve_data_dir();
            tracing::info!("starting RenOS Agent v{}", env!("CARGO_PKG_VERSION"));
            tracing::info!("data dir: {data_dir} (set RENOS_DATA_DIR to override)");

            if let Err(e) = platform::ensure_dirs() {
                tracing::warn!("failed to create platform directories: {}", e);
            }

            let (identity, _key) = identity::load_or_generate();
            tracing::info!("device identity: {} (enrolled: {})", identity.device_id, identity.is_enrolled);

			if identity.is_enrolled {
				let device_secret = load_device_secret()
					.expect("device is enrolled but no device-secret.txt found — re-enroll the agent");

				let device_id = identity.device_id.clone();
				let api_url = cfg.api_url.clone();

				let mut client = grpc_client::Client::new(&cfg);
				client.set_device_id(identity.device_id.clone());
				client.set_device_secret(device_secret.clone());

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

				let cmd_api_url = api_url.clone();
				let cmd_device_id = device_id.clone();
				let cmd_device_secret = device_secret.clone();
				tokio::spawn(async move {
					let mut interval = tokio::time::interval(std::time::Duration::from_secs(15));
					loop {
						interval.tick().await;
						match commands::poll(&cmd_api_url, &cmd_device_id, &cmd_device_secret).await {
							Ok(pending) => {
								for cmd in &pending {
									tracing::info!("executing command: {}", cmd.command);
									let (exit_code, stdout, stderr) = commands::execute(&cmd.command);
									tracing::info!("command {} completed (exit={})", cmd.id, exit_code);
									if !stderr.is_empty() {
										tracing::warn!("command {} stderr: {}", cmd.id, stderr);
									}
									if let Err(e) = commands::submit_result(
										&cmd_api_url, &cmd_device_id, &cmd_device_secret,
										&cmd.id, exit_code, &stdout, &stderr,
									).await {
										tracing::warn!("failed to submit result for {}: {}", cmd.id, e);
									}
								}
							}
							Err(e) => tracing::warn!("command poll failed: {}", e),
						}
					}
				});

				// ─── Docker state sync tasks ──────────────────────────────────────

				// Containers (existing)
				let docker_api_url = api_url.clone();
				let docker_device_id = device_id.clone();
				let docker_device_secret = device_secret.clone();
				tokio::spawn(async move {
					let mut interval = tokio::time::interval(std::time::Duration::from_secs(60));
					loop {
						interval.tick().await;

						if !docker::is_installed() {
							tracing::info!("docker not installed on this device");
							continue;
						}

						match docker::list_containers() {
							Ok(containers) => {
								tracing::info!("reporting {} Docker container(s)", containers.len());
								if let Err(e) = docker::submit_containers(
									&docker_api_url, &docker_device_id, &docker_device_secret,
									&containers,
								).await {
									tracing::warn!("failed to submit containers: {}", e);
								}
							}
							Err(e) => tracing::warn!("failed to list Docker containers: {}", e),
						}
					}
				});

				// Docker images
				let img_api_url = api_url.clone();
				let img_device_id = device_id.clone();
				let img_device_secret = device_secret.clone();
				tokio::spawn(async move {
					let mut interval = tokio::time::interval(std::time::Duration::from_secs(120));
					loop {
						interval.tick().await;
						if !docker::is_installed() { continue; }
						match docker_images::list_images() {
							Ok(images) => {
								tracing::info!("reporting {} Docker image(s)", images.len());
								if let Err(e) = docker_images::submit_images(
									&img_api_url, &img_device_id, &img_device_secret, &images,
								).await {
									tracing::warn!("failed to submit docker images: {}", e);
								}
							}
							Err(e) => tracing::warn!("failed to list Docker images: {}", e),
						}
					}
				});

				// Docker volumes
				let vol_api_url = api_url.clone();
				let vol_device_id = device_id.clone();
				let vol_device_secret = device_secret.clone();
				tokio::spawn(async move {
					let mut interval = tokio::time::interval(std::time::Duration::from_secs(120));
					loop {
						interval.tick().await;
						if !docker::is_installed() { continue; }
						match docker_volumes::list_volumes() {
							Ok(volumes) => {
								tracing::info!("reporting {} Docker volume(s)", volumes.len());
								if let Err(e) = docker_volumes::submit_volumes(
									&vol_api_url, &vol_device_id, &vol_device_secret, &volumes,
								).await {
									tracing::warn!("failed to submit docker volumes: {}", e);
								}
							}
							Err(e) => tracing::warn!("failed to list Docker volumes: {}", e),
						}
					}
				});

				// Docker networks
				let net_api_url = api_url.clone();
				let net_device_id = device_id.clone();
				let net_device_secret = device_secret.clone();
				tokio::spawn(async move {
					let mut interval = tokio::time::interval(std::time::Duration::from_secs(120));
					loop {
						interval.tick().await;
						if !docker::is_installed() { continue; }
						match docker_networks::list_networks() {
							Ok(networks) => {
								tracing::info!("reporting {} Docker network(s)", networks.len());
								if let Err(e) = docker_networks::submit_networks(
									&net_api_url, &net_device_id, &net_device_secret, &networks,
								).await {
									tracing::warn!("failed to submit docker networks: {}", e);
								}
							}
							Err(e) => tracing::warn!("failed to list Docker networks: {}", e),
						}
					}
				});

				// Docker actions (existing — now also handles logs + stats)
				let act_api_url = api_url.clone();
				let act_device_id = device_id.clone();
				let act_device_secret = device_secret.clone();
				tokio::spawn(async move {
					let mut interval = tokio::time::interval(std::time::Duration::from_secs(15));
					loop {
						interval.tick().await;

						if !docker::is_installed() {
							continue;
						}

						match docker_actions::poll(&act_api_url, &act_device_id, &act_device_secret).await {
							Ok(pending) => {
								for act in &pending {
									tracing::info!("executing docker {} on container {}", act.action, act.container_id);
									let (exit_code, stdout, stderr) = docker_actions::execute_action(&act.action, &act.container_id);
									tracing::info!("docker action {} completed (exit={})", act.id, exit_code);
									if !stderr.is_empty() {
										tracing::warn!("docker action {} stderr: {}", act.id, stderr);
									}
									if let Err(e) = docker_actions::submit_result(
										&act_api_url, &act_device_id, &act_device_secret,
										&act.id, exit_code, &stdout, &stderr,
									).await {
										tracing::warn!("failed to submit docker action result for {}: {}", act.id, e);
									}
								}
							}
							Err(e) => tracing::warn!("docker actions poll failed: {}", e),
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
