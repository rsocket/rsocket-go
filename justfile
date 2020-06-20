default:
	echo 'Hello, world!'
lint:
        golangci-lint run ./...
test:
        go test -race -count=1 . -v
fmt:
        @go fmt ./...
