# Deployment Guide

## Production Deployment

### Prerequisites

- Docker 24+ with Docker Compose V2
- 2GB+ RAM
- 20GB+ disk space
- Network access between devices and server

### Steps

1. **Clone the repository**
   ```bash
   git clone https://github.com/renos-project/renos.git
   cd renos
   ```

2. **Run the installer**
   ```bash
   ./install.sh
   ```

3. **Configure TLS** (recommended)
   
   Place your SSL certificates in `deploy/nginx/ssl/`:
   ```
   deploy/nginx/ssl/fullchain.pem
   deploy/nginx/ssl/privkey.pem
   ```
   
   Then uncomment the HTTPS server block in `deploy/nginx/nginx.conf`.

4. **Configure external access**
   
   Update `.env`:
   ```
   RENOS_EXTERNAL_URL=https://renos.yourdomain.com
   ```

5. **Set up backups**
   ```bash
   # Add to crontab for daily backups
   0 2 * * * /path/to/renos/scripts/backup.sh
   ```

### Updating

```bash
# Pull latest images
docker compose -f docker-compose.prod.yml pull

# Restart with new images
docker compose -f docker-compose.prod.yml up -d

# Run any new migrations
docker compose -f docker-compose.prod.yml exec core renos-core migrate
```

### Monitoring

```bash
# Check all services
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f

# Health check
curl http://localhost/api/v1/healthz

# Readiness check
curl http://localhost/api/v1/readyz
```

## Development Deployment

```bash
# Start infrastructure
docker compose up -d

# Start all services
./scripts/dev-up.sh

# Stop all services
./scripts/dev-down.sh
```

## Docker Compose Files

| File | Purpose | Services |
|---|---|---|
| `docker-compose.yml` | Development infrastructure | PostgreSQL, Redis, NATS |
| `docker-compose.prod.yml` | Full production stack | All of the above + Core, Hub, Nginx |

## Environment Variables

See `.env.example` for the complete reference with documentation for each variable.

### Security Checklist

- [ ] `RENOS_JWT_SECRET` is a secure random value (min 32 chars)
- [ ] `RENOS_ENCRYPTION_KEY` is a secure random value (min 32 chars)
- [ ] `POSTGRES_PASSWORD` is a strong password
- [ ] `RENOS_ADMIN_PASSWORD` is not the default "admin"
- [ ] `RENOS_EXTERNAL_URL` uses HTTPS
- [ ] TLS certificates are configured for Nginx
- [ ] Firewall limits access to required ports only

## Troubleshooting

### Services won't start
```bash
# Check Docker daemon
docker info

# Check service logs
docker compose -f docker-compose.prod.yml logs core
docker compose -f docker-compose.prod.yml logs postgres

# Check health
docker compose -f docker-compose.prod.yml ps
```

### Database connection errors
```bash
# Verify PostgreSQL is running
docker compose -f docker-compose.prod.yml exec postgres pg_isready -U renos

# Check connection string
echo $RENOS_DATABASE_URL
```

### Migration errors
```bash
# Run migrations manually
docker compose -f docker-compose.prod.yml exec core renos-core migrate

# Rollback last migration
docker compose -f docker-compose.prod.yml exec core renos-core migrate down 1
```
