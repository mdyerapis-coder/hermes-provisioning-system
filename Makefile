build:
	go build -o bin/hps ./cmd/hps

run:
	go run ./cmd/hps

test:
	go test ./...

fmt:
	go fmt ./...
