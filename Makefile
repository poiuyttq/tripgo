#start
-include .env
export
run:
	go run ./cmd/trip-service
#generating code from openapi spec	
generate:


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