IMAGE ?= ghcr.io/manny-ovena/inventory-adjust:local

.PHONY: build test vet image

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

image:
	docker build -t $(IMAGE) -f Dockerfile ../../..
