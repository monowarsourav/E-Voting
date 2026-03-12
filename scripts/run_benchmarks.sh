#!/bin/bash

echo "=========================================="
echo "   CovertVote Benchmark Suite            "
echo "=========================================="

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Create results directory
mkdir -p test/benchmark/results

# Run crypto benchmarks
echo -e "\n${YELLOW}[1/4] Running Crypto Benchmarks...${NC}"
go test -bench=. -benchmem ./test/benchmark/crypto_benchmark_test.go -run=^$ | tee test/benchmark/results/crypto_bench.txt

# Run existing PQ benchmarks
echo -e "\n${YELLOW}[2/4] Running Post-Quantum Benchmarks...${NC}"
go test -bench=. -benchmem ./internal/pq/... -run=^$ | tee test/benchmark/results/pq_bench.txt

# Run scalability test
echo -e "\n${YELLOW}[3/4] Running Scalability Benchmark...${NC}"
go test -v -run TestScalabilityBenchmark ./test/benchmark/... -timeout 30m

# Run individual timing
echo -e "\n${YELLOW}[4/4] Running Individual Operation Timing...${NC}"
go test -v -run TestIndividualOperationTiming ./test/benchmark/...

echo -e "\n${GREEN}✅ Benchmarks completed!${NC}"
echo "Results saved to: test/benchmark/results/"
echo ""
echo "View results:"
echo "  - Crypto benchmarks: test/benchmark/results/crypto_bench.txt"
echo "  - PQ benchmarks: test/benchmark/results/pq_bench.txt"
echo "  - Scalability results: test/benchmark/results/benchmark_results.md"
