MIGRATE_VERSION := v4.19.1
MIGRATIONS_DIR := migrations
MIGRATE := go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)

.PHONY: migrate-up migrate-down migrate-force migrate-create

migrate-up:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

migrate-force:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" force $(VERSION)

migrate-create:
	@test -n "$(NAME)" || (echo "NAME is required" && exit 1)
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)
