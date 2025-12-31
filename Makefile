IMAGE_NAME := docker-socket-proxy

.PHONY: build
build:
	docker build -t $(IMAGE_NAME) .

.PHONY: test
test:
	go test -v ./pkg/...

.PHONY: lint
lint:
	go fmt ./...
	go vet ./...
	golangci-lint run

.PHONY: check-lint
check-lint:
	golangci-lint run
