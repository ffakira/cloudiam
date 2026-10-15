-include .env.local

CLOUDIAM_DATABASE_URL ?= postgres://cloudiam:cloudiam@127.0.0.1:6000/cloudiam?sslmode=disable
CLOUDIAM_MIGRATE_DATABASE_URL ?= postgres://cloudiam:cloudiam@postgres:5432/cloudiam?sslmode=disable
CLOUDIAM_SERVICE_GRPC_PORT ?= 50051
CLOUDIAM_SERVICE_HTTP_PORT ?= 18080
CLOUDIAM_REDIS_URL ?= redis://127.0.0.1:6379/0
LOCAL_UID ?= $(shell id -u)
LOCAL_GID ?= $(shell id -g)
DB_WAIT_SECONDS ?= 60
CLOUDIAM_BROKER_USER_NAME ?= cloud-iam-broker
SERVER_TOOLS_BIN ?= /tmp/cloudiam-go-tools/bin
GO_BUILD_CACHE ?= /tmp/cloudiam-go-cache

export CLOUDIAM_DATABASE_URL
export CLOUDIAM_MIGRATE_DATABASE_URL
export CLOUDIAM_SERVICE_GRPC_PORT
export CLOUDIAM_SERVICE_HTTP_PORT
export CLOUDIAM_REDIS_URL
export CLOUDIAM_DEPLOYMENT_ENV
export CLOUDIAM_CONSOLE_COOKIE_SECURE
export CLOUDIAM_ALLOW_CONSOLE_SEED_IN_PRODUCTION
export CLOUDIAM_ALLOW_INSECURE_DATABASE_IN_PRODUCTION
export LOCAL_UID
export LOCAL_GID
export DB_WAIT_SECONDS
export CLOUDIAM_BROKER_USER_NAME
export AWS_PROFILE
export CLOUDIAM_AWS_ACCOUNT_ID
export CLOUDIAM_HTTP_ADDR
export CLOUDIAM_AUTH_WORKSPACE
export CLOUDIAM_AUTH_ENDPOINT
export CLOUDIAM_AUTH_VERIFICATION_URI
export CLOUDIAM_CONSOLE_SEED_USERNAME
export CLOUDIAM_CONSOLE_SEED_PASSWORD
export CLOUDIAM_CONSOLE_JWT_SECRET
export CLOUDIAM_CONSOLE_JWT_TTL
export CLOUDIAM_DASHBOARD_DIST_DIR
export SERVER_TOOLS_BIN
export GO_BUILD_CACHE

MIGRATE := docker compose run --rm migrate \
	-path=/migrations \
	-database "$(CLOUDIAM_MIGRATE_DATABASE_URL)"

.PHONY: help
help:
	@printf '%s\n' \
		'Targets:' \
		'  make env-example       Print the local env template' \
		'  make db-up             Start local Postgres on 127.0.0.1:6000' \
		'  make redis-up          Start local Redis on 127.0.0.1:6379' \
		'  make db-wait           Wait until local Postgres accepts connections' \
		'  make db-status         Show local Postgres container status' \
		'  make db-down           Stop local Postgres' \
		'  make redis-down        Stop local Redis' \
		'  make db-logs           Follow local Postgres logs' \
		'  make migrate-up        Apply all pending migrations' \
		'  make migrate-down      Roll back one migration (override MIGRATE_STEPS=N)' \
		'  make migrate-status    Show migration version and dirty state' \
		'  make migrate-force VERSION=N  Force migration version after manual repair' \
		'  make service-build    Build the local CloudIAM server image' \
		'  make service-up       Start Postgres, migrate, and run the server container' \
		'  make service-down     Stop local Compose services' \
		'  make service-logs     Follow CloudIAM server container logs' \
		'  make proto             Regenerate server gRPC code from proto files' \
		'  make server-run        Run local server with .env.local loaded' \
		'  make config-validate   Validate local CloudIAM YAML config' \
		'  make aws-scan-policies Scan configured AWS role trust policies' \
		'  make server-test       Run Go tests for the server module' \
		'  make server-test-integration  Run server integration tests' \
		'  make server-security-check  Run server tests, vet, race, vuln, SAST, and lint checks' \
		'' \
		'Variables:' \
		'  CLOUDIAM_DATABASE_URL          Override the host local CloudIAM Postgres URL' \
		'  CLOUDIAM_MIGRATE_DATABASE_URL  Override the Compose-network migration URL' \
		'  CLOUDIAM_SERVICE_GRPC_PORT     Override server container host gRPC port' \
		'  CLOUDIAM_SERVICE_HTTP_PORT     Override server container host HTTP port' \
		'  CLOUDIAM_REDIS_URL             Override Redis URL for shared rate limits' \
		'  CLOUDIAM_BROKER_USER_NAME      Override broker IAM user name for policy scanning' \
		'  DB_WAIT_SECONDS                Override Postgres readiness timeout'

.PHONY: env-example
env-example:
	@cat .env.example

.PHONY: db-up
db-up:
	docker compose up -d postgres

.PHONY: redis-up
redis-up:
	docker compose up -d redis

.PHONY: redis-down
redis-down:
	docker compose stop redis

.PHONY: db-wait
db-wait:
	@printf 'waiting for local postgres'
	@deadline=$$(( $$(date +%s) + $(DB_WAIT_SECONDS) )); \
	until docker compose exec -T postgres pg_isready -U cloudiam -d cloudiam >/dev/null 2>&1; do \
		if [ $$(date +%s) -ge $$deadline ]; then \
			printf '\npostgres did not become ready within %s seconds\n' "$(DB_WAIT_SECONDS)"; \
			docker compose ps postgres; \
			exit 1; \
		fi; \
		printf '.'; \
		sleep 1; \
	done
	@printf '\npostgres is ready\n'

.PHONY: db-status
db-status:
	docker compose ps postgres

.PHONY: db-down
db-down:
	docker compose down

.PHONY: db-logs
db-logs:
	docker compose logs -f postgres

.PHONY: migrate-up
migrate-up: db-wait
	$(MIGRATE) up

.PHONY: migrate-down
migrate-down:
	$(MIGRATE) down $(or $(MIGRATE_STEPS),1)

.PHONY: migrate-status
migrate-status: db-wait
	$(MIGRATE) version

.PHONY: migrate-force
migrate-force:
ifndef VERSION
	$(error VERSION is required, for example: make migrate-force VERSION=1)
endif
	$(MIGRATE) force $(VERSION)

.PHONY: service-build
service-build:
	docker compose build cloudiam-server

.PHONY: service-up
service-up: migrate-up
	docker compose up -d cloudiam-server

.PHONY: service-down
service-down:
	docker compose down

.PHONY: service-logs
service-logs:
	docker compose logs -f cloudiam-server

.PHONY: server-test
server-test:
	cd server && go test ./...

.PHONY: server-security-check
server-security-check:
	cd server/tools && GOBIN="$(SERVER_TOOLS_BIN)" go install tool
	cd server && GOCACHE="$(GO_BUILD_CACHE)" go mod verify
	cd server && GOCACHE="$(GO_BUILD_CACHE)" go vet ./...
	cd server && GOCACHE="$(GO_BUILD_CACHE)" go test ./...
	cd server && GOCACHE="$(GO_BUILD_CACHE)" go test -race ./...
	cd server && GOCACHE="$(GO_BUILD_CACHE)" "$(SERVER_TOOLS_BIN)/govulncheck" ./...
	cd server && GOCACHE="$(GO_BUILD_CACHE)" "$(SERVER_TOOLS_BIN)/gosec" -quiet -exclude-dir=gen ./...
	cd server && GOCACHE="$(GO_BUILD_CACHE)" "$(SERVER_TOOLS_BIN)/golangci-lint" run --no-config --default=standard ./...

.PHONY: proto
proto:
	cd server && PATH="$$(go env GOPATH)/bin:$$PATH" protoc \
		-I proto \
		--go_out=. \
		--go_opt=module=github.com/ffakira/cloudiam/server \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/ffakira/cloudiam/server \
		proto/access/v1/access.proto \
		proto/auth/v1/auth.proto

.PHONY: server-run
server-run:
	cd server && go run ./cmd/access-server --config ./configs/cloudiam.yaml

.PHONY: config-validate
config-validate:
	cd server && GOCACHE=/tmp/cloudiam-go-cache go run ./cmd/access-server validate-config --config ./configs/cloudiam.yaml

.PHONY: aws-scan-policies
aws-scan-policies:
	cd server && GOCACHE=/tmp/cloudiam-go-cache go run ./cmd/access-server scan-aws-policies --config ./configs/cloudiam.yaml --broker-user "$(CLOUDIAM_BROKER_USER_NAME)"

.PHONY: server-test-integration
server-test-integration: migrate-up
	cd server && go test -tags=integration ./internal/adapters/postgres

.PHONY: client-install
client-install:
	cd client && bun install

.PHONY: client-dev
client-dev:
	cd client && bun run dev

.PHONY: client-build
client-build:
	cd client && bun run build
