MIGRATE_VERSION := v4.19.1
MIGRATIONS_DIR := migrations
MIGRATE := go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: migrate-up migrate-down migrate-force migrate-create docker-build docker-up docker-down docker-logs

migrate-up:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

migrate-force:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required" && exit 1)
	@$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" force $(VERSION)

migrate-create:
	@test -n "$(NAME)" || (echo "NAME is required" && exit 1)
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)

docker-build:
	docker compose --env-file .env.production build

docker-up:
	docker compose --env-file .env.production up -d --build

docker-down:
	docker compose --env-file .env.production down

docker-logs:
	docker compose --env-file .env.production logs -f api
