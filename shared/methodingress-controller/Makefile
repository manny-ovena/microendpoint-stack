.PHONY: build test vet image

build:
	CGO_ENABLED=0 go build -o bin/manager ./cmd

test:
	go test ./...

vet:
	go vet ./...

image:
	docker build -t methodingress-controller:ci -f Dockerfile ../..
