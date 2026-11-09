#start
-include .env
export
run:
	go run ./cmd/trip-service
#generating code from openapi spec	
generate:
	go generate ./internal/usecase/... ./internal/http/...

	go tool oapi-codegen \
	-generate types,chi-server \
	-package api \
	-o internal/generated/api.gen.go \
	contracts/openapi/trip-service.openapi.yaml
#migration commands
migrate:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up
	

migrate_down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

lint:
	go tool golangci-lint run --config .golangci.yml ./...

test:
	@coverage_file=$$(mktemp); \
	trap 'rm -f "$$coverage_file"' EXIT; \
	go test -race -count=2 -coverprofile="$$coverage_file" ./... && \
	go tool cover -func="$$coverage_file" | tail -n 1

test-integration: migrate
	go test -race -count=2 -tags=integration ./internal/repository/...