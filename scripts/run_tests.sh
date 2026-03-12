#!/bin/bash

echo "=========================================="
echo "   CovertVote Test Suite                 "
echo "=========================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Run unit tests
echo -e "\n${GREEN}[1/4] Running Unit Tests...${NC}"
go test -v ./internal/crypto/...
go test -v ./internal/smdc/...
go test -v ./internal/sa2/...
go test -v ./internal/biometric/...
go test -v ./internal/voting/...
go test -v ./internal/tally/...

# Run handler tests
echo -e "\n${GREEN}[2/4] Running API Handler Tests...${NC}"
go test -v ./api/handlers/...

# Run integration tests
echo -e "\n${GREEN}[3/4] Running Integration Tests...${NC}"
go test -v ./tests/...

# Run tests with coverage
echo -e "\n${GREEN}[4/4] Generating Coverage Report...${NC}"
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

echo -e "\n${GREEN}✅ All tests completed!${NC}"
echo "Coverage report: coverage.html"

# Display coverage summary
echo -e "\n${YELLOW}Coverage Summary:${NC}"
go tool cover -func=coverage.out | grep total
