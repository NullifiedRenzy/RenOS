# Fedora LAN proxy

This deployment uses port 18081 for Android/LAN access and proxies it to the
existing Nginx listener on port 80. Docker must be enabled at boot and the
production containers must retain their `unless-stopped` restart policy.

Install as the account that owns the existing proxy:

```bash
command -v ncat
install -D -m 644 deploy/systemd/renos-lan-proxy.service \
  "$HOME/.config/systemd/user/renos-lan-proxy.service"
loginctl enable-linger "$USER"
systemctl --user daemon-reload
# Stop any manually started listener on 18081 before starting this service.
systemctl --user enable --now renos-lan-proxy.service
```

Check startup and recovery:

```bash
loginctl show-user "$USER" -p Linger
systemctl is-enabled docker
systemctl --user is-enabled renos-lan-proxy.service
systemctl --user status renos-lan-proxy.service
curl --fail http://127.0.0.1:18081/api/v1/healthz
```

The proxy retries after a process failure and starts without an interactive
login. Connections made before Docker/Nginx is ready may fail until the backend
starts. This does not add HTTPS or make the LAN endpoint safe to expose publicly.
An actual host reboot remains a separate maintenance test.
