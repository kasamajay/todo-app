# All targets wrap docker compose - no native Go/Node/npm installation is
# required. Windows users without `make` can run the underlying
# `docker compose ...` commands directly (see README.md).

install-tools:
	docker compose build

dev-api:
	docker compose up api

dev-web:
	docker compose up web

dev:
	docker compose up

test:
	docker compose run --rm api go test ./...

build:
	docker compose build

fmt:
	docker compose run --rm api go fmt ./...

down:
	docker compose down

# Production mode: nginx serves the prebuilt bundle, no Node at runtime.
prod-build:
	docker compose -f docker-compose.prod.yml build

prod-up:
	docker compose -f docker-compose.prod.yml up -d --build

prod-down:
	docker compose -f docker-compose.prod.yml down

.PHONY: install-tools dev-api dev-web dev test build fmt down prod-build prod-up prod-down
