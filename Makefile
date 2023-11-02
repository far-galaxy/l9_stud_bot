build:
	@go build -ldflags "-X main.build=$(git rev-parse --abbrev-ref HEAD)+$(date '+%F-%T')"  main.go


lint:
	@golangci-lint run -E gosec
