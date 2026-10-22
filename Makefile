.PHONY: build run test clean tidy

APP_NAME=biz-credit-ai
BUILD_DIR=./build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server/

run:
	go run ./cmd/server/

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f biz-credit-ai-server

tidy:
	go mod tidy

deps:
	go mod download
