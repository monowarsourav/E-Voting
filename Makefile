.PHONY: all build test clean install deps fmt vet lint

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Binary names
API_BINARY=covertvote-api
AGGR_A_BINARY=aggregator-a
AGGR_B_BINARY=aggregator-b

# Directories
CMD_DIR=./cmd
BUILD_DIR=./build
DATA_DIR=./data
MIGRATIONS_DIR=./migrations

all: deps test build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies installed"

# Build all binaries
build: build-api build-aggregator-a build-aggregator-b

# Build main API server
build-api:
	@echo "Building API server..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(API_BINARY) $(CMD_DIR)/api-server/main.go
	@echo "API server build complete"

# Build aggregator A
build-aggregator-a:
	@echo "Building aggregator A..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(AGGR_A_BINARY) $(CMD_DIR)/aggregator-a/main.go
	@echo "Aggregator A build complete"

# Build aggregator B
build-aggregator-b:
	@echo "Building aggregator B..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(AGGR_B_BINARY) $(CMD_DIR)/aggregator-b/main.go
	@echo "Aggregator B build complete"

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Test specific module
test-crypto:
	$(GOTEST) -v ./internal/crypto/

test-smdc:
	$(GOTEST) -v ./internal/smdc/

test-sa2:
	$(GOTEST) -v ./internal/sa2/

test-biometric:
	$(GOTEST) -v ./internal/biometric/

test-voter:
	$(GOTEST) -v ./internal/voter/

test-utils:
	$(GOTEST) -v ./pkg/utils/

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install binaries to GOPATH/bin
install: build
	@echo "Installing binaries..."
	$(GOCMD) install $(CMD_DIR)/api-server/main.go
	$(GOCMD) install $(CMD_DIR)/aggregator-a/main.go
	$(GOCMD) install $(CMD_DIR)/aggregator-b/main.go
	@echo "Install complete"

# Initialize database
init-db:
	@echo "Initializing database..."
	@mkdir -p $(DATA_DIR)
	@echo "Database directory created"

# Run API server
run-api: build-api init-db
	@echo "Running API server..."
	$(BUILD_DIR)/$(API_BINARY)

# Run aggregator A
run-aggregator-a: build-aggregator-a
	@echo "Running aggregator A..."
	SERVER_PORT=8081 $(BUILD_DIR)/$(AGGR_A_BINARY)

# Run aggregator B
run-aggregator-b: build-aggregator-b
	@echo "Running aggregator B..."
	SERVER_PORT=8082 $(BUILD_DIR)/$(AGGR_B_BINARY)

# Run all services (in separate terminals)
run-all:
	@echo "To run all services, execute these in separate terminals:"
	@echo "  make run-aggregator-a"
	@echo "  make run-aggregator-b"
	@echo "  make run-api"
	@echo ""
	@echo "Or use Docker Compose:"
	@echo "  make docker-up"

# Development mode with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Security scan (requires gosec)
security:
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec ./...

# Generate documentation
docs:
	@echo "Generating documentation..."
	@which godoc > /dev/null || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	@echo "Run 'godoc -http=:6060' and visit http://localhost:6060/pkg/"

# Docker build (when Dockerfile is created)
docker-build:
	docker build -t covertvote:latest .

# Docker compose (when docker-compose.yml is created)
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Help
help:
	@echo "Available targets:"
	@echo "  make deps              - Install dependencies"
	@echo "  make build             - Build all binaries"
	@echo "  make init-db           - Initialize database directory"
	@echo "  make run-api           - Run API server"
	@echo "  make run-aggregator-a  - Run SA² aggregator A"
	@echo "  make run-aggregator-b  - Run SA² aggregator B"
	@echo "  make run-all           - Show commands to run all services"
	@echo "  make test              - Run all tests"
	@echo "  make test-coverage     - Run tests with coverage report"
	@echo "  make test-<module>     - Test specific module (crypto, smdc, sa2, etc.)"
	@echo "  make fmt               - Format code"
	@echo "  make vet               - Run go vet"
	@echo "  make clean             - Clean build artifacts"
	@echo "  make bench             - Run benchmarks"
	@echo "  make security          - Run security scan"
	@echo "  make docker-build      - Build Docker image"
	@echo "  make docker-up         - Start all services with Docker Compose"
	@echo "  make docker-down       - Stop all services"
	@echo "  make help              - Show this help"
