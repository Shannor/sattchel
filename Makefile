BINARY := sattchel
BIN_DIR := bin
PKG     := ./...

.PHONY: build run test fmt vet tidy clean

build:
	go build -o $(BIN_DIR)/$(BINARY) .

run:
	go run .

test:
	go test $(PKG)

fmt:
	go fmt $(PKG)

vet:
	go vet $(PKG)

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
