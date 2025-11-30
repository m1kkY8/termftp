.PHONY: run build

APP_NAME := termftp
BIN_DIR := bin
ENTRY := ./cmd

run:
	go run $(ENTRY)

build:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(BIN_DIR)/$(APP_NAME) $(ENTRY)
