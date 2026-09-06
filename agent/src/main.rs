mod commands;
mod config;
mod container_stats;
mod docker;
mod docker_actions;
mod docker_compose;
mod docker_images;
mod docker_networks;
mod docker_volumes;
mod enroll;
mod file_transfers;
mod files;
mod grpc_client;
mod health;
mod identity;
mod nats_exec;
mod nats_process;
mod nats_proxmox;
mod nats_terminal;
mod network;
mod platform;
mod plugins;
mod processes;
mod proto;
mod proxmox;
mod pty_exec;
mod retry;
mod service_actions;
mod storage;
mod systemd;
mod systemd_services;
mod updates;
mod virtualization;

use std::path::PathBuf;
use std::sync::Arc;

use clap::Parser;
use tokio::sync::Notify;
use tracing_subscriber::EnvFilter;

#[derive(Parser)]
#[command(
    name = "renos-agent",
    version = "0.1.0",
    about = "RenOS Agent — lightweight device management daemon"
)]
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
    Run {
        /// Core server URL (e.g., http://192.168.0.117)
        #[arg(long)]
        core: Option<String>,
    },
    /// Print version information
    Version,
}

fn load_device_secret() -> Option<String> {
    let path = PathBuf::from(platform::resolve_data_dir()).join("device-secret.txt");
    std::fs::read_to_string(path)
        .ok()
        .map(|s| s.trim().to_string())
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .json()
        .with_env_filter(EnvFilter::from_default_env())
        .init();

    let cli = Cli::parse();

    match cli.command.unwrap_or(Command::Run { core: None }) {
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
                    println!(
                        "  Device secret stored in: {}/device-secret.txt",
                        platform::resolve_data_dir()
                    );
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
            println!(
                "target: {}-{}",
                std::env::consts::ARCH,
                std::env::consts::OS
            );
            println!("data dir: {}", platform::resolve_data_dir());
        }
        Command::Run { core } => {
            let mut cfg = config::load();
            if let Some(core_url) = core {
                cfg.api_url = core_url;
            }
            let data_dir = platform::resolve_data_dir();
            tracing::info!("starting RenOS Agent v{}", env!("CARGO_PKG_VERSION"));
            tracing::info!("data dir: {data_dir} (set RENOS_DATA_DIR to override)");

            if let Err(e) = platform::ensure_dirs() {
                tracing::warn!("failed to create platform directories: {}", e);
            }

            let (identity, _key) = identity::load_or_generate();
            tracing::info!(
                "device identity: {} (enrolled: {})",
                identity.device_id,
                identity.is_enrolled
            );

            if identity.is_enrolled {
                let device_secret = match load_device_secret() {
                    Some(s) => s,
                    None => {
                        tracing::error!(
                            "device is enrolled but device-secret.txt is missing — re-enroll the agent"
                        );
                        return;
                    }
                };

                let device_id = identity.device_id.clone();
                let api_url = cfg.api_url.clone();
                let protected_services = cfg.protected_services.clone();

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

                // NATS executor for real-time command execution
                let nats_url = cfg.nats_url.clone();
                let nats_device_id = device_id.clone();
                tokio::spawn(async move {
                    match async_nats::connect(&nats_url).await {
                        Ok(nc) => {
                            tracing::info!("connected to NATS at {}", nats_url);
                            let _ = nats_exec::start_nats_executor(nc, nats_device_id).await;
                        }
                        Err(e) => {
                            tracing::warn!(
                                "failed to connect to NATS: {} (executions disabled)",
                                e
                            );
                        }
                    }
                });

                // NATS process handler for live process management
                let nats_proc_url = cfg.nats_url.clone();
                let nats_proc_device_id = device_id.clone();
                tokio::spawn(async move {
                    match async_nats::connect(&nats_proc_url).await {
                        Ok(nc) => {
                            tracing::info!(
                                "connected to NATS for process management at {}",
                                nats_proc_url
                            );
                            let _handler =
                                nats_process::start_nats_process_handler(nc, nats_proc_device_id)
                                    .await;
                        }
                        Err(e) => {
                            tracing::warn!(
                                "failed to connect to NATS for process management: {}",
                                e
                            );
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
                        match commands::poll(&cmd_api_url, &cmd_device_id, &cmd_device_secret).await
                        {
                            Ok(pending) => {
                                for cmd in &pending {
                                    tracing::info!("executing command: {}", cmd.command);
                                    let (exit_code, stdout, stderr) =
                                        commands::execute(&cmd.command);
                                    tracing::info!(
                                        "command {} completed (exit={})",
                                        cmd.id,
                                        exit_code
                                    );
                                    if !stderr.is_empty() {
                                        tracing::warn!("command {} stderr: {}", cmd.id, stderr);
                                    }
                                    if let Err(e) = commands::submit_result(
                                        &cmd_api_url,
                                        &cmd_device_id,
                                        &cmd_device_secret,
                                        &cmd.id,
                                        exit_code,
                                        &stdout,
                                        &stderr,
                                    )
                                    .await
                                    {
                                        tracing::warn!(
                                            "failed to submit result for {}: {}",
                                            cmd.id,
                                            e
                                        );
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
                                tracing::info!(
                                    "reporting {} Docker container(s)",
                                    containers.len()
                                );
                                if let Err(e) = docker::submit_containers(
                                    &docker_api_url,
                                    &docker_device_id,
                                    &docker_device_secret,
                                    &containers,
                                )
                                .await
                                {
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
                        if !docker::is_installed() {
                            continue;
                        }
                        match docker_images::list_images() {
                            Ok(images) => {
                                tracing::info!("reporting {} Docker image(s)", images.len());
                                if let Err(e) = docker_images::submit_images(
                                    &img_api_url,
                                    &img_device_id,
                                    &img_device_secret,
                                    &images,
                                )
                                .await
                                {
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
                        if !docker::is_installed() {
                            continue;
                        }
                        match docker_volumes::list_volumes() {
                            Ok(volumes) => {
                                tracing::info!("reporting {} Docker volume(s)", volumes.len());
                                if let Err(e) = docker_volumes::submit_volumes(
                                    &vol_api_url,
                                    &vol_device_id,
                                    &vol_device_secret,
                                    &volumes,
                                )
                                .await
                                {
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
                        if !docker::is_installed() {
                            continue;
                        }
                        match docker_networks::list_networks() {
                            Ok(networks) => {
                                tracing::info!("reporting {} Docker network(s)", networks.len());
                                if let Err(e) = docker_networks::submit_networks(
                                    &net_api_url,
                                    &net_device_id,
                                    &net_device_secret,
                                    &networks,
                                )
                                .await
                                {
                                    tracing::warn!("failed to submit docker networks: {}", e);
                                }
                            }
                            Err(e) => tracing::warn!("failed to list Docker networks: {}", e),
                        }
                    }
                });

                // Compose stacks
                let cmp_api_url = api_url.clone();
                let cmp_device_id = device_id.clone();
                let cmp_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(60));
                    loop {
                        interval.tick().await;
                        if !docker_compose::is_compose_available() {
                            continue;
                        }
                        match docker_compose::list_compose_stacks() {
                            Ok(stacks) => {
                                tracing::info!("reporting {} compose stack(s)", stacks.len());
                                if let Err(e) = docker_compose::submit_compose_stacks(
                                    &cmp_api_url,
                                    &cmp_device_id,
                                    &cmp_device_secret,
                                    &stacks,
                                )
                                .await
                                {
                                    tracing::warn!("failed to submit compose stacks: {}", e);
                                }
                            }
                            Err(e) => tracing::warn!("failed to list compose stacks: {}", e),
                        }
                    }
                });

                // Container metrics (live stats)
                let cms_api_url = api_url.clone();
                let cms_device_id = device_id.clone();
                let cms_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(15));
                    loop {
                        interval.tick().await;
                        if !docker::is_installed() {
                            continue;
                        }
                        match container_stats::collect_container_stats() {
                            Ok(metrics) => {
                                if !metrics.is_empty() {
                                    tracing::info!(
                                        "reporting metrics for {} container(s)",
                                        metrics.len()
                                    );
                                    if let Err(e) = container_stats::submit_container_metrics(
                                        &cms_api_url,
                                        &cms_device_id,
                                        &cms_device_secret,
                                        &metrics,
                                    )
                                    .await
                                    {
                                        tracing::warn!("failed to submit container metrics: {}", e);
                                    }
                                }
                            }
                            Err(e) => tracing::warn!("failed to collect container stats: {}", e),
                        }
                    }
                });

                // ─── Systemd state sync tasks ─────────────────────────────────────

                // Systemd services
                let sys_api_url = api_url.clone();
                let sys_device_id = device_id.clone();
                let sys_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(120));
                    loop {
                        interval.tick().await;
                        if !systemd::is_available() {
                            tracing::info!("systemd not available on this device");
                            continue;
                        }
                        match systemd_services::list_services() {
                            Ok(services) => {
                                tracing::info!("reporting {} systemd service(s)", services.len());
                                if let Err(e) = systemd_services::submit_services(
                                    &sys_api_url,
                                    &sys_device_id,
                                    &sys_device_secret,
                                    &services,
                                )
                                .await
                                {
                                    tracing::warn!("failed to submit systemd services: {}", e);
                                }
                            }
                            Err(e) => tracing::warn!("failed to list systemd services: {}", e),
                        }
                    }
                });

                // Network interface sync
                let netif_api_url = api_url.clone();
                let netif_device_id = device_id.clone();
                let netif_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(120));
                    loop {
                        interval.tick().await;
                        match network::list_interfaces() {
                            Ok(interfaces) => {
                                tracing::info!(
                                    "reporting {} network interface(s)",
                                    interfaces.len()
                                );
                                if let Err(e) = network::submit_interfaces(
                                    &netif_api_url,
                                    &netif_device_id,
                                    &netif_device_secret,
                                    &interfaces,
                                )
                                .await
                                {
                                    tracing::warn!("failed to submit network interfaces: {}", e);
                                }
                            }
                            Err(e) => tracing::warn!("failed to list network interfaces: {}", e),
                        }
                    }
                });

                // Storage device sync
                let stg_api_url = api_url.clone();
                let stg_device_id = device_id.clone();
                let stg_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(180));
                    loop {
                        interval.tick().await;
                        match storage::list_storage_devices() {
                            Ok(devices) => {
                                tracing::info!("reporting {} storage device(s)", devices.len());
                                if let Err(e) = storage::submit_storage_devices(
                                    &stg_api_url,
                                    &stg_device_id,
                                    &stg_device_secret,
                                    &devices,
                                )
                                .await
                                {
                                    tracing::warn!("failed to submit storage devices: {}", e);
                                }
                            }
                            Err(e) => tracing::warn!("failed to list storage devices: {}", e),
                        }
                    }
                });

                // Virtual machine sync
                let vm_api_url = api_url.clone();
                let vm_device_id = device_id.clone();
                let vm_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(180));
                    loop {
                        interval.tick().await;
                        if !virtualization::is_available() {
                            continue;
                        }
                        match virtualization::list_vms() {
                            Ok(vms) => {
                                tracing::info!("reporting {} VM(s)", vms.len());
                                if let Err(e) = virtualization::submit_vms(
                                    &vm_api_url,
                                    &vm_device_id,
                                    &vm_device_secret,
                                    &vms,
                                )
                                .await
                                {
                                    tracing::warn!("failed to submit VMs: {}", e);
                                }
                            }
                            Err(e) => tracing::warn!("failed to list VMs: {}", e),
                        }
                    }
                });

                // Update sync
                let upd_api_url = api_url.clone();
                let upd_device_id = device_id.clone();
                let upd_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(3600));
                    loop {
                        interval.tick().await;
                        if !updates::is_available() {
                            continue;
                        }
                        match updates::list_updates() {
                            Ok(packages) => {
                                tracing::info!("reporting {} package update(s)", packages.len());
                                if let Err(e) = updates::submit_updates(
                                    &upd_api_url,
                                    &upd_device_id,
                                    &upd_device_secret,
                                    &packages,
                                )
                                .await
                                {
                                    tracing::warn!("failed to submit updates: {}", e);
                                }
                            }
                            Err(e) => tracing::warn!("failed to list updates: {}", e),
                        }
                    }
                });

                // Plugin sync
                let plg_api_url = api_url.clone();
                let plg_device_id = device_id.clone();
                let plg_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(300));
                    loop {
                        interval.tick().await;
                        match plugins::list_plugins() {
                            Ok(list) => {
                                tracing::info!("reporting {} plugin(s)", list.len());
                                if let Err(e) = plugins::submit_plugins(
                                    &plg_api_url,
                                    &plg_device_id,
                                    &plg_device_secret,
                                    &list,
                                )
                                .await
                                {
                                    tracing::warn!("failed to submit plugins: {}", e);
                                }
                            }
                            Err(e) => tracing::warn!("failed to list plugins: {}", e),
                        }
                    }
                });

                // Systemd service actions
                let svc_act_api_url = api_url.clone();
                let svc_act_device_id = device_id.clone();
                let svc_act_device_secret = device_secret.clone();
                let svc_act_protected = protected_services.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(15));
                    loop {
                        interval.tick().await;

                        if !systemd::is_available() {
                            continue;
                        }

                        match service_actions::poll(
                            &svc_act_api_url,
                            &svc_act_device_id,
                            &svc_act_device_secret,
                        )
                        .await
                        {
                            Ok(pending) => {
                                for act in &pending {
                                    tracing::info!(
                                        "executing systemctl {} on {}",
                                        act.action,
                                        act.unit_name
                                    );
                                    let (exit_code, stdout, stderr) =
                                        service_actions::execute_action(
                                            &act.action,
                                            &act.unit_name,
                                            &svc_act_protected,
                                        );
                                    tracing::info!(
                                        "service action {} completed (exit={})",
                                        act.id,
                                        exit_code
                                    );
                                    if !stderr.is_empty() {
                                        tracing::warn!(
                                            "service action {} stderr: {}",
                                            act.id,
                                            stderr
                                        );
                                    }
                                    if let Err(e) = service_actions::submit_result(
                                        &svc_act_api_url,
                                        &svc_act_device_id,
                                        &svc_act_device_secret,
                                        &act.id,
                                        exit_code,
                                        &stdout,
                                        &stderr,
                                    )
                                    .await
                                    {
                                        tracing::warn!(
                                            "failed to submit service action result for {}: {}",
                                            act.id,
                                            e
                                        );
                                    }
                                }
                            }
                            Err(e) => tracing::warn!("service actions poll failed: {}", e),
                        }
                    }
                });

                // File actions (browse, read, search, create, delete, rename, move, copy, preview)
                let file_api_url = api_url.clone();
                let file_device_id = device_id.clone();
                let file_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(15));
                    loop {
                        interval.tick().await;

                        match files::poll(&file_api_url, &file_device_id, &file_device_secret).await
                        {
                            Ok(pending) => {
                                for act in &pending {
                                    tracing::info!(
                                        "executing file {} on {}",
                                        act.action_type,
                                        act.path
                                    );
                                    let (exit_code, stdout, stderr, result_json) =
                                        files::execute_action_with_target(
                                            &act.action_type,
                                            &act.path,
                                            &act.target_path,
                                            &act.query,
                                        );
                                    tracing::info!(
                                        "file action {} completed (exit={})",
                                        act.id,
                                        exit_code
                                    );
                                    if !stderr.is_empty() {
                                        tracing::warn!("file action {} stderr: {}", act.id, stderr);
                                    }
                                    if let Err(e) = files::submit_result(
                                        &file_api_url,
                                        &file_device_id,
                                        &file_device_secret,
                                        &act.id,
                                        exit_code,
                                        &stdout,
                                        &stderr,
                                        &result_json,
                                    )
                                    .await
                                    {
                                        tracing::warn!(
                                            "failed to submit file action result for {}: {}",
                                            act.id,
                                            e
                                        );
                                    }
                                }
                            }
                            Err(e) => tracing::warn!("file actions poll failed: {}", e),
                        }
                    }
                });

                // File transfers (upload/download chunks)
                let ft_api_url = api_url.clone();
                let ft_device_id = device_id.clone();
                let ft_device_secret = device_secret.clone();
                tokio::spawn(async move {
                    let mut interval = tokio::time::interval(std::time::Duration::from_secs(10));
                    loop {
                        interval.tick().await;

                        match file_transfers::poll_pending(
                            &ft_api_url,
                            &ft_device_id,
                            &ft_device_secret,
                        )
                        .await
                        {
                            Ok(pending) => {
                                for transfer in &pending {
                                    tracing::info!(
                                        "processing file transfer {} ({})",
                                        transfer.id,
                                        transfer.transfer_type
                                    );
                                    let result = match transfer.transfer_type.as_str() {
                                        "upload" => {
                                            file_transfers::process_upload(
                                                &ft_api_url,
                                                &ft_device_id,
                                                &ft_device_secret,
                                                transfer,
                                            )
                                            .await
                                        }
                                        "download" => {
                                            file_transfers::process_download(
                                                &ft_api_url,
                                                &ft_device_id,
                                                &ft_device_secret,
                                                transfer,
                                            )
                                            .await
                                        }
                                        _ => {
                                            tracing::warn!(
                                                "unknown transfer type: {}",
                                                transfer.transfer_type
                                            );
                                            Ok(())
                                        }
                                    };
                                    if let Err(e) = result {
                                        tracing::warn!(
                                            "file transfer {} failed: {}",
                                            transfer.id,
                                            e
                                        );
                                    }
                                }
                            }
                            Err(e) => tracing::warn!("file transfers poll failed: {}", e),
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

                        match docker_actions::poll(&act_api_url, &act_device_id, &act_device_secret)
                            .await
                        {
                            Ok(pending) => {
                                for act in &pending {
                                    tracing::info!(
                                        "executing docker {} on container {}",
                                        act.action,
                                        act.container_id
                                    );
                                    let (exit_code, stdout, stderr) =
                                        docker_actions::execute_action(
                                            &act.action,
                                            &act.container_id,
                                        );
                                    tracing::info!(
                                        "docker action {} completed (exit={})",
                                        act.id,
                                        exit_code
                                    );
                                    if !stderr.is_empty() {
                                        tracing::warn!(
                                            "docker action {} stderr: {}",
                                            act.id,
                                            stderr
                                        );
                                    }
                                    if let Err(e) = docker_actions::submit_result(
                                        &act_api_url,
                                        &act_device_id,
                                        &act_device_secret,
                                        &act.id,
                                        exit_code,
                                        &stdout,
                                        &stderr,
                                    )
                                    .await
                                    {
                                        tracing::warn!(
                                            "failed to submit docker action result for {}: {}",
                                            act.id,
                                            e
                                        );
                                    }
                                }
                            }
                            Err(e) => tracing::warn!("docker actions poll failed: {}", e),
                        }
                    }
                });

                // ─── Proxmox state sync ──────────────────────────────────────────

                if proxmox::is_proxmox() {
                    tracing::info!("Proxmox host detected, starting proxmox sync");

                    // Proxmox data collection
                    let px_api_url = api_url.clone();
                    let px_device_id = device_id.clone();
                    let px_device_secret = device_secret.clone();
                    let px_conns = cfg.proxmox_connections.clone();
                    tokio::spawn(async move {
                        let mut interval =
                            tokio::time::interval(std::time::Duration::from_secs(60));
                        loop {
                            interval.tick().await;

                            for conn_cfg in &px_conns {
                                let conn = proxmox::ProxmoxConnection {
                                    host: conn_cfg.host.clone(),
                                    port: conn_cfg.port.unwrap_or(8006),
                                    auth: match conn_cfg.auth_type.as_deref().unwrap_or("token") {
                                        "password" => proxmox::AuthMethod::Password {
                                            username: conn_cfg.username.clone().unwrap_or_default(),
                                            password: conn_cfg.password.clone().unwrap_or_default(),
                                        },
                                        _ => proxmox::AuthMethod::Token {
                                            token_id: conn_cfg.token_id.clone().unwrap_or_default(),
                                            token_secret: conn_cfg
                                                .token_secret
                                                .clone()
                                                .unwrap_or_default(),
                                        },
                                    },
                                    verify_ssl: conn_cfg.verify_ssl.unwrap_or(true),
                                };

                                let mut client = proxmox::ProxmoxClient::new(conn);
                                match client.collect_all().await {
                                    Ok(data) => {
                                        tracing::info!(
                                            "collected proxmox: {} nodes, {} vms, {} lxcs, {} storage, {} networks",
                                            data.nodes.len(), data.vms.len(), data.lxcs.len(),
                                            data.storage.len(), data.networks.len()
                                        );
                                        if let Err(e) = proxmox::submit_proxmox_data(
                                            &px_api_url,
                                            &px_device_id,
                                            &px_device_secret,
                                            &data,
                                        )
                                        .await
                                        {
                                            tracing::warn!("failed to submit proxmox data: {}", e);
                                        }
                                    }
                                    Err(e) => {
                                        tracing::warn!("proxmox collection failed: {}", e);
                                    }
                                }
                            }
                        }
                    });

                    // NATS proxmox power action handler
                    let nats_px_url = cfg.nats_url.clone();
                    let nats_px_device_id = device_id.clone();
                    let nats_px_conns: Vec<proxmox::ProxmoxConnection> = cfg
                        .proxmox_connections
                        .iter()
                        .map(|c| proxmox::ProxmoxConnection {
                            host: c.host.clone(),
                            port: c.port.unwrap_or(8006),
                            auth: match c.auth_type.as_deref().unwrap_or("token") {
                                "password" => proxmox::AuthMethod::Password {
                                    username: c.username.clone().unwrap_or_default(),
                                    password: c.password.clone().unwrap_or_default(),
                                },
                                _ => proxmox::AuthMethod::Token {
                                    token_id: c.token_id.clone().unwrap_or_default(),
                                    token_secret: c.token_secret.clone().unwrap_or_default(),
                                },
                            },
                            verify_ssl: c.verify_ssl.unwrap_or(true),
                        })
                        .collect();
                    tokio::spawn(async move {
                        match async_nats::connect(&nats_px_url).await {
                            Ok(nc) => {
                                tracing::info!(
                                    "connected to NATS for proxmox management at {}",
                                    nats_px_url
                                );
                                let _handler = nats_proxmox::start_nats_proxmox_handler(
                                    nc,
                                    nats_px_device_id,
                                    nats_px_conns,
                                )
                                .await;
                            }
                            Err(e) => {
                                tracing::warn!("failed to connect to NATS for proxmox: {}", e);
                            }
                        }
                    });
                }

                // ─── Terminal (PTY) via NATS ─────────────────────────────────────

                let term_nats_url = cfg.nats_url.clone();
                let term_device_id = device_id.clone();
                let pty_manager = pty_exec::PtyManager::new();
                tokio::spawn(async move {
                    match async_nats::connect(&term_nats_url).await {
                        Ok(nc) => {
                            tracing::info!(
                                "connected to NATS for terminal management at {}",
                                term_nats_url
                            );
                            let _handler = nats_terminal::start_nats_terminal_handler(
                                nc,
                                term_device_id,
                                pty_manager,
                            )
                            .await;
                        }
                        Err(e) => {
                            tracing::warn!("failed to connect to NATS for terminal: {}", e);
                        }
                    }
                });
            } else {
                tracing::warn!(
                    "device not enrolled — run `renos-agent enroll --token <token>` first"
                );
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
        let mut term = signal(SignalKind::terminate()).expect("failed to install SIGTERM handler");
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
