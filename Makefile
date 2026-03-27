BINARY=port-forward
BUILD_DIR=./bin

.PHONY: build run clean install tidy

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) ./cmd/main.go
	@echo "Built → $(BUILD_DIR)/$(BINARY)"

run:
	CGO_ENABLED=1 go run ./cmd/main.go

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

clean:
	rm -rf $(BUILD_DIR)

tidy:
	go mod tidy

lint:
	go vet ./...
