IMAGE ?= ghcr.io/manny-ovena/kafka-publisher:local

.PHONY: build test vet image

build:
	GOWORK=off go build ./...

test:
	GOWORK=off go test ./...

vet:
	GOWORK=off go vet ./...

image:
	docker build -t $(IMAGE) -f Dockerfile ../../..
