IMAGE ?= ghcr.io/manny-ovena/kafka-publisher:local

.PHONY: tidy build test vet image

tidy:
	go mod tidy

build:
	GOWORK=off go build ./...

test:
	GOWORK=off go test ./...

vet:
	GOWORK=off go vet ./...

image:
	docker build -t $(IMAGE) -f Dockerfile ../../..
