# ============================================================
# 📦 CloudRip - Makefile
# Arquitectura Hexagonal - Build, Test, Distribución
# ============================================================

# --- Variables generales ---
APP_NAME      := cloudrip
BIN_DIR       := bin
SRC_DIR       := ./cmd/$(APP_NAME)
#SRC_DIR       := ./main_bck.go
BUILD_DIR     := build
VERSION       := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DATE          := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT        := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
GOFLAGS       := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
PLATFORMS     := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# --- Colores para output ---
BLUE=\033[0;34m
GREEN=\033[0;32m
YELLOW=\033[1;33m
RESET=\033[0m

# --- PHONY targets ---
.PHONY: all build run clean fmt lint test docker dist help

# ============================================================
# 🔧 Comandos principales
# ============================================================

all: build

build:
	@echo "$(BLUE)🔨 Compilando $(APP_NAME)...$(RESET)"
	@go build $(GOFLAGS) -o $(BIN_DIR)/$(APP_NAME) $(SRC_DIR)
	@echo "$(GREEN)✅ Build completo: $(BIN_DIR)/$(APP_NAME)$(RESET)"

run:
	@echo "$(BLUE)🚀 Ejecutando CloudRip...$(RESET)"
	@go run $(SRC_DIR) -h

clean:
	@echo "$(YELLOW)🧹 Limpiando artefactos...$(RESET)"
	@rm -rf $(BIN_DIR) $(BUILD_DIR)
	@go clean
	@echo "$(GREEN)✅ Limpieza completa$(RESET)"

# ============================================================
# 🧪 Test, Lint y Formato
# ============================================================

fmt:
	@echo "$(BLUE)🧩 Formateando código...$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)✅ Código formateado$(RESET)"

lint:
	@echo "$(BLUE)🔍 Ejecutando linter...$(RESET)"
	@golangci-lint run ./... || echo "⚠️  Linter encontró advertencias"

test:
	@echo "$(BLUE)🧪 Ejecutando tests...$(RESET)"
	@go test ./... -v -count=1
	@echo "$(GREEN)✅ Tests completados$(RESET)"

# ============================================================
# 🐳 Docker Build & Run
# ============================================================

docker:
	@echo "$(BLUE)🐳 Construyendo imagen Docker...$(RESET)"
	@docker build -t $(APP_NAME):$(VERSION) .
	@echo "$(GREEN)✅ Imagen lista: $(APP_NAME):$(VERSION)$(RESET)"

docker-run:
	@docker run --rm $(APP_NAME):$(VERSION) -h

# ============================================================
# 🚀 Distribución (multi-plataforma)
# ============================================================

dist: clean
	@echo "$(BLUE)📦 Generando binarios para múltiples plataformas...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@$(foreach platform, $(PLATFORMS), \
		GOOS=$(word 1, $(subst /, ,$(platform))) \
		GOARCH=$(word 2, $(subst /, ,$(platform))) \
		; \
		output_name=$(BUILD_DIR)/$(APP_NAME)_$${GOOS}_$${GOARCH}; \
		echo "🔧 Compilando para $${GOOS}/$${GOARCH}"; \
		CGO_ENABLED=0 GOOS=$${GOOS} GOARCH=$${GOARCH} go build $(GOFLAGS) -o $$output_name $(SRC_DIR); \
	)
	@echo "$(GREEN)✅ Binarios generados en $(BUILD_DIR)/$(RESET)"

# ============================================================
# 📘 Ayuda
# ============================================================

help:
	@echo "$(YELLOW)Comandos disponibles:$(RESET)"
	@echo "  make build         - Compila binario local"
	@echo "  make run           - Ejecuta app (modo desarrollo)"
	@echo "  make clean         - Limpia artefactos"
	@echo "  make fmt           - Formatea código"
	@echo "  make lint          - Linter (usa golangci-lint)"
	@echo "  make test          - Ejecuta tests"
	@echo "  make docker        - Construye imagen Docker"
	@echo "  make dist          - Compila multi-plataforma"
	@echo "  make help          - Muestra esta ayuda"
