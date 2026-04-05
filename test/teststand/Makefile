COMPOSE := docker compose
COMPOSE_FILE := -f docker-compose.yaml

.PHONY: up down test logs status clean

## up: Build and start all test-stand services (waits for healthy status)
up:
	$(COMPOSE) $(COMPOSE_FILE) up -d --build --wait

## down: Stop and remove all containers, networks, and volumes
down:
	$(COMPOSE) $(COMPOSE_FILE) down -v --remove-orphans

## test: Run checker in --run-once mode against the test stand config
test:
	$(COMPOSE) $(COMPOSE_FILE) run --rm checker ./checker --config /etc/checker/config.yaml --run-once

## logs: Follow logs for all services
logs:
	$(COMPOSE) $(COMPOSE_FILE) logs -f

## status: Show current service status
status:
	$(COMPOSE) $(COMPOSE_FILE) ps

## clean: Stop services, remove containers, volumes, and built images
clean: down
	$(COMPOSE) $(COMPOSE_FILE) down --rmi local -v --remove-orphans
