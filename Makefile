# Database configuration
DB_HOST ?= 127.0.0.1
DB_PORT ?= 3306
DB_USER ?= root
DB_PASSWORD ?= 
DB_NAME ?= komkomunikacijos
DB_DSN = "$(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?parseTime=true"

# Migration directory
MIGRATIONS_DIR = migrations

# Goose binary
GOOSE = $(shell go env GOPATH)/bin/goose

.PHONY: help install-goose migrate-up migrate-down migrate-status migrate-reset migrate-create db-create

# Default target
help:
	@echo "Available commands:"
	@echo "  install-goose    - Install goose migration tool"
	@echo "  db-create        - Create database"
	@echo "  migrate-up       - Run all pending migrations"
	@echo "  migrate-down     - Rollback last migration"
	@echo "  migrate-status   - Show migration status"
	@echo "  migrate-reset    - Reset all migrations (WARNING: drops all data)"
	@echo "  migrate-create   - Create new migration file (usage: make migrate-create NAME=migration_name)"
	@echo ""
	@echo "Database configuration (can be overridden):"
	@echo "  DB_HOST=$(DB_HOST)"
	@echo "  DB_PORT=$(DB_PORT)"
	@echo "  DB_USER=$(DB_USER)"
	@echo "  DB_NAME=$(DB_NAME)"
	@echo ""
	@echo "Example usage:"
	@echo "  make migrate-up"
	@echo "  make migrate-up DB_PASSWORD=mypassword"
	@echo "  make migrate-create NAME=add_users_table"

# Install goose migration tool
install-goose:
	@echo "Installing goose..."
	go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "Goose installed successfully!"

# Create database
db-create:
	@echo "Creating database $(DB_NAME)..."
	mysql -h$(DB_HOST) -P$(DB_PORT) -u$(DB_USER) $(if $(DB_PASSWORD),-p$(DB_PASSWORD)) -e "CREATE DATABASE IF NOT EXISTS $(DB_NAME);"
	@echo "Database $(DB_NAME) created successfully!"

# Run all pending migrations
migrate-up:
	@echo "Running migrations..."
	$(GOOSE) -dir $(MIGRATIONS_DIR) mysql $(DB_DSN) up
	@echo "Migrations completed!"

# Rollback last migration
migrate-down:
	@echo "Rolling back last migration..."
	$(GOOSE) -dir $(MIGRATIONS_DIR) mysql $(DB_DSN) down
	@echo "Rollback completed!"

# Show migration status
migrate-status:
	@echo "Migration status:"
	$(GOOSE) -dir $(MIGRATIONS_DIR) mysql $(DB_DSN) status

# Reset all migrations (WARNING: drops all data)
migrate-reset:
	@echo "WARNING: This will reset all migrations and drop all data!"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	$(GOOSE) -dir $(MIGRATIONS_DIR) mysql $(DB_DSN) reset
	@echo "All migrations reset!"

# Create new migration file
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)"
	$(GOOSE) -dir $(MIGRATIONS_DIR) create $(NAME) sql
	@echo "Migration file created in $(MIGRATIONS_DIR)/"

# Development helpers
dev-setup: install-goose db-create migrate-up
	@echo "Development environment setup complete!"

# Clean up (for testing)
clean:
	@echo "Dropping database $(DB_NAME)..."
	mysql -h$(DB_HOST) -P$(DB_PORT) -u$(DB_USER) $(if $(DB_PASSWORD),-p$(DB_PASSWORD)) -e "DROP DATABASE IF EXISTS $(DB_NAME);"
	@echo "Database $(DB_NAME) dropped!"

# Full reset and setup
reset-dev: clean dev-setup
	@echo "Development environment reset and setup complete!"
