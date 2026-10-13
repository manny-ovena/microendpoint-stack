.PHONY: tidy build test vet image

tidy:
	go mod tidy

build:
	CGO_ENABLED=0 go build -o bin/manager ./cmd

test:
	go test ./...

vet:
	go vet ./...

image:
	docker build -t methodingress-controller:ci -f Dockerfile ../..
