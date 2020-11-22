alias e := echo
alias b := echo-bench
alias t := test-no-cover

build:
	go build .
lint:
    golangci-lint run ./...
test:
    go test -count=1 -coverprofile=coverage.out ./...
test-no-cover:
    go test -count=1 ./...
test-race:
    go test -count=1 -race ./...
fmt:
    @go fmt ./...
cover:
    @go tool cover -html=coverage.out
echo:
    GODEBUG=gctrace=1 go run ./examples/echo/echo.go
echo-bench:
    GODEBUG=gctrace=1 go run ./examples/echo_bench/echo_bench.go
