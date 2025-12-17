.PHONY: up down build logs test backend-shell clean

# Start all services
up:
	docker compose --env-file config.env up

# Start all services in detached mode (background)
up-d:
	docker compose --env-file config.env up -d --build

# Stop and remove containers
down:
	docker compose --env-file config.env down

# Build containers
build:
	docker compose --env-file config.env build

# View logs
logs:
	docker compose logs -f

# Run backend tests inside the container
test:
	cd backend && go test ./... -v

# Open a shell inside the backend container
backend-shell:
	docker compose exec backend sh

# Clean up volumes (Warning: Deletes all wallet data)
clean:
	docker compose down -v

# Generate regtest activity (mine blocks, send funds to your wallet)
regtest-activity:
	./scripts/regtest-activity.sh

