#!/bin/bash
set -e

echo "========================================="
echo "  CovertVote: Caliper Benchmark"
echo "========================================="

cd "$(dirname "$0")/../caliper"

# Install Caliper CLI if needed
if [ ! -d "node_modules/@hyperledger/caliper-cli" ]; then
    echo "[1/3] Installing Caliper..."
    npm init -y
    npm install --only=prod @hyperledger/caliper-cli@0.6.0
    npx caliper bind --caliper-bind-sut fabric:2.5
else
    echo "[1/3] Caliper already installed"
fi

# Create results directory
mkdir -p ../test/benchmark/results

# Run benchmark
echo "[2/3] Running benchmark..."
npx caliper launch manager \
    --caliper-workspace ./ \
    --caliper-benchconfig benchconfig.yaml \
    --caliper-networkconfig networkconfig.yaml \
    --caliper-flow-only-test \
    2>&1 | tee ../test/benchmark/results/caliper_results.txt

# Copy HTML report
echo "[3/3] Saving results..."
cp report.html ../test/benchmark/results/caliper_report.html 2>/dev/null || true

echo ""
echo "========================================="
echo "  Benchmark Complete!"
echo "  Results: test/benchmark/results/caliper_results.txt"
echo "  Report:  test/benchmark/results/caliper_report.html"
echo "========================================="
