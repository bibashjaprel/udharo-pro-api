# VPS Deployment

This deployment runs:

- `api`: the Go backend
- `postgres`: private PostgreSQL database
- `migrate`: one-shot database migration job

Postgres is not published to the VPS network interface. The API binds to
`127.0.0.1:8080` by default so a reverse proxy can terminate TLS.

## First Deploy

1. Install Docker and the Docker Compose plugin on the VPS.
2. Clone the repository.
3. Create production environment values:

```sh
cp .env.production.example .env.production
openssl rand -base64 48
```

Edit `.env.production` and replace all secrets.

4. Build and start:

```sh
docker compose --env-file .env.production up -d --build
```

5. Check status:

```sh
docker compose --env-file .env.production ps
docker compose --env-file .env.production logs -f api
```

6. Verify locally on the VPS:

```sh
curl http://127.0.0.1:8080/health
```

## Updates

```sh
git pull
docker compose --env-file .env.production up -d --build
docker image prune -f
```

The `migrate` service runs before the API starts. If there are no pending
migrations, it exits successfully.

## Rollback Notes

Keep database backups before deploying migrations. To roll back app code:

```sh
git checkout <previous-commit>
docker compose --env-file .env.production up -d --build
```

Database migration rollback should be handled deliberately with
`golang-migrate`; do not blindly run down migrations on production data.

## Reverse Proxy

Use Nginx, Caddy, or Traefik on the VPS host to proxy HTTPS traffic to:

```text
http://127.0.0.1:8080
```

An Nginx example is included at `deploy/nginx.conf.example`.
