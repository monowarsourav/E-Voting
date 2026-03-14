#!/bin/bash
echo "========================================"
echo "  CovertVote ProVerif Verification"
echo "========================================"

# Check if ProVerif is installed
if ! command -v proverif &> /dev/null; then
    echo "ERROR: ProVerif not installed."
    echo "Install: sudo apt-get install proverif"
    echo "    or:  brew install proverif"
    echo "    or:  https://bblanche.gitlabpages.inria.fr/proverif/"
    exit 1
fi

echo ""
echo "ProVerif version:"
proverif -help 2>&1 | head -1

echo ""
echo "[1/3] Checking Ballot Privacy (diff-equivalence)..."
echo "------"
proverif covertvote.pv 2>&1 | grep -E "RESULT|Error|cannot"

echo ""
echo "[2/3] To check Verifiability:"
echo "  Edit covertvote.pv: uncomment OPTION B, comment OPTION A"
echo "  Then run: proverif covertvote.pv"

echo ""
echo "[3/3] To check Eligibility:"
echo "  Edit covertvote.pv: uncomment OPTION C, comment OPTION A"
echo "  Then run: proverif covertvote.pv"

echo ""
echo "========================================"
echo "  Done!"
echo "========================================"
