default:
	echo 'Hello, world!'
lint:
        golangci-lint run ./...
test:
        go test -count=1 -coverprofile=coverage.out \
        ./core/... \
        ./extension/... \
        ./internal/... \
        ./lease/... \
        ./logger/... \
        ./payload/... \
        ./rx/... \
        .
test-no-cover:
        go test -count=1 ./... -v
test-race:
        go test -race -count=1 ./... -v
fmt:
        @go fmt ./...
cover:
        @go tool cover -html=coverage.out
