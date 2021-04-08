PACKAGES=$(shell go list ./...)

build:
	go build $(PACKAGES)

lint:
	golangci-lint run

# go get mvdan.cc/gofumpt
fmt:
	gofumpt -w **/*.go

# go get go get github.com/daixiang0/gci
imp:
	gci -w **/*.go

test:
	go test github.com/e-money/client/keys

# integration test
grpc-test:
	@echo 'ensure local em-chain gRPC listener is up'
	@echo ''
	@go test -v github.com/e-money/client

.PHONY: build lint fmt imp test
