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
	go test -v $(PACKAGES)

.PHONY: build lint fmt imp test
