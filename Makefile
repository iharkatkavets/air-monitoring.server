PORT ?= 4001
DB   ?= "api.db"
ENV  ?= "development"

BIN_DIR := bin
BIN     := $(BIN_DIR)/air-server

LOG_DIR  := logs
LOG      := $(LOG_DIR)/air-server.log
PID      := $(LOG_DIR)/air-server.pid

.PHONY: build clean release start stop logs release_for_linux_on_mac

$(BIN_DIR):
	@mkdir -p $@

clean:
	@echo "[CLEAN] Start"
	@-rm -rf $(BIN_DIR)
	@go clean
	@echo "[CLEAN] Done"

build: clean | $(BIN_DIR)
	@echo "[BUILD_API] Start"
	@go build -o $(BIN) ./cmd/api
	@echo "[BUILD_API] Done"

release_for_linux_on_mac: clean | $(BIN_DIR)
	@echo "[BUILD_RELEASE] Start"
	@open -a Docker
	@echo "Waiting for Docker..."
	@timeout=60; \
	while ! docker info >/dev/null 2>&1; do \
		if [ $$timeout -le 0 ]; then \
			echo "Docker did not start in time"; \
			exit 1; \
		fi; \
		sleep 1; \
		timeout=$$((timeout-1)); \
	done
	@docker run --rm --platform linux/arm64 \
		-v "$$PWD":/src -w /src golang:1.25-alpine \
		/bin/sh -lc '\
		  set -euo pipefail; \
		  apk add --no-cache build-base musl-dev sqlite-dev sqlite-static; \
		  echo "Building $(BIN)"; \
		  CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=gcc \
		  /usr/local/go/bin/go build \
		    -tags "sqlite_omit_load_extension,netgo,osusergo,timetzdata" \
		    -trimpath \
		    -ldflags "-s -w -linkmode external -extldflags -static" \
		    -o $(BIN) ./cmd/api/ \
		'
	@echo "[BUILD_RELEASE] Done"

start: stop build 
	@echo "[START_API] Start"
	@mkdir -p "$(LOG_DIR)"
	@nohup $(BIN) -port=$(PORT) -env=$(ENV) -db=$(DB) \
	    >>"$(LOG)" 2>&1 </dev/null & echo $$! >"$(PID)"
	@echo "PID: $$(cat $(PID))  Logs: $(LOG)"
	@echo "[START_API] Done"

stop: 
	@echo "[STOP_API] Start"
	@if [ -f "$(PID)" ]; then \
		kill "$$(cat $(PID))" 2>/dev/null || true; \
		rm -f "$(PID)"; \
	else \
		echo "No PID file."; \
	fi
	@echo "[STOP_API] Done"

logs:
	@echo "Tailing $(LOG)… (Ctrl-C to stop)"
	@tail -f "$(LOG)"
