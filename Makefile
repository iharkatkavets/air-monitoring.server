PORT ?= 4001
DB   ?= "api.db"
ENV  ?= "development"

BIN_DIR := dist
BIN := $(BIN_DIR)/api-server
PID := $(BIN_DIR)/api.pid

LOG_DIR  := logs
LOG      := $(LOG_DIR)/api.log

.PHONY: build clean release start stop logs

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

release: | $(BIN_DIR)
	@echo "[BUILD_RELEASE] Start"
	@CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=musl-gcc \
	  go build -tags "sqlite_omit_load_extension" \
	  -trimpath -ldflags "-s -w -extldflags -static" \
	  -o $(BIN) ./cmd/api/
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

