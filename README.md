# Udharo Pro API

## Database migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) for
PostgreSQL database migrations. Migration files live in the `migrations`
directory.

Set `DATABASE_URL` before running migration commands:

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
