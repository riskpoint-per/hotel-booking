TEST_DATABASE_URL ?= postgres://hotel:hotel@localhost:5432/hotel?sslmode=disable

.PHONY: up down logs run test test-integration

up:                ## Build and start the API and Postgres
	docker compose up --build -d

down:              ## Stop containers and delete the database volume
	docker compose down -v

logs:              ## Follow API logs (watch the confirmation emails here)
	docker compose logs -f api

run:               ## Run the API locally against the compose database
	DATABASE_URL="$(TEST_DATABASE_URL)" go run ./cmd/api

test:              ## Unit tests (no database needed)
	go test ./...

test-integration:  ## Unit + integration tests (needs `docker compose up -d db`; resets the DB)
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -count=1 ./...
