build:
	@go build -o bin/userservice cmd/main.go

unit-test:
	@go test -v userservice/cmd/migration
	@go test -v userservice/internal/auth
	@go test -v userservice/internal/components/role
	@go test -v userservice/internal/components/user
	@go test -v userservice/internal/components/service
	@go test -v userservice/internal/configs
	@go test -v userservice/internal/middleware
	@go test -v  userservice/internal/misc
	@go test -v  userservice/internal/utils

integration-test:
	@go test -timeout 10s -run TestIntegrationSuite userservice/tests/integration
