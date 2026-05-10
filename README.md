# Udharo Pro API

## Production deployment

Docker Compose deployment files are included for running the API on a VPS:

- `Dockerfile`
- `docker-compose.yml`
- `.env.production.example`
- `deploy/README.md`
- `deploy/nginx.conf.example`

Quick start:

```sh
cp .env.production.example .env.production
docker compose --env-file .env.production up -d --build
```

Read `deploy/README.md` before using this in production, especially the notes
about secrets, TLS reverse proxying, and database backups.

## Database migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) for
PostgreSQL database migrations. Migration files live in the `migrations`
directory.

Set `DATABASE_URL` in `.env` or export it before running migration commands:

```sh
export DATABASE_URL="postgres://user:password@localhost:5432/udharo_pro?sslmode=disable"
```

Apply all pending migrations:

```sh
make migrate-up
```

Roll back the most recent migration:

```sh
make migrate-down
```

Create a new migration pair:

```sh
make migrate-create NAME=create_customers
```

If a migration is interrupted and the database is left dirty, force the version
after checking the database state:

```sh
make migrate-force VERSION=1
```
