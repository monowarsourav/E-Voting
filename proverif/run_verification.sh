#!/bin/bash
echo "========================================"
echo "  CovertVote ProVerif Verification"
echo "========================================"

if ! command -v proverif &> /dev/null; then
    echo "ERROR: ProVerif not installed."
    echo "Install: opam install proverif"
    exit 1
fi

echo ""
echo "[1/3] Ballot Privacy (observational equivalence)..."
proverif privacy.pv 2>&1 | grep -E "RESULT|Error"

echo ""
echo "[2/3] Individual Verifiability (correspondence)..."
proverif verifiability.pv 2>&1 | grep -E "RESULT|Error"

echo ""
echo "[3/3] Voter Eligibility (correspondence)..."
proverif eligibility.pv 2>&1 | grep -E "RESULT|Error"

echo ""
echo "========================================"
echo "  Done!"
echo "========================================"
