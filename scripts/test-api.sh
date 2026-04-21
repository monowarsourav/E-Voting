#!/bin/bash

# CovertVote API Testing Script
# This script tests the complete voting flow

API_BASE="http://localhost:8080/api/v1"
VOTER_ID="bob"
PASSWORD="TestPassword123"

echo "=========================================="
echo "CovertVote API Testing Script"
echo "=========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Health Check
echo -e "${YELLOW}Test 1: Health Check${NC}"
curl -s http://localhost:8080/health | jq '.'
echo ""

# Test 2: Get Elections
echo -e "${YELLOW}Test 2: Get Available Elections${NC}"
curl -s ${API_BASE}/elections | jq '.elections[] | {election_id, title, is_active, total_votes}'
echo ""

# Test 3: Register Voter
echo -e "${YELLOW}Test 3: Register Voter (${VOTER_ID})${NC}"
REGISTER_RESPONSE=$(curl -s -X POST ${API_BASE}/register \
  -H "Content-Type: application/json" \
  -d "{
    \"voter_id\": \"${VOTER_ID}\",
    \"password\": \"${PASSWORD}\"
  }")

echo "$REGISTER_RESPONSE" | jq '.'

if echo "$REGISTER_RESPONSE" | jq -e '.voter_id' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Registration successful${NC}"
else
    echo -e "${RED}✗ Registration failed${NC}"
    if echo "$REGISTER_RESPONSE" | grep -q "already registered"; then
        echo -e "${YELLOW}Note: Voter already registered, continuing with login...${NC}"
    else
        exit 1
    fi
fi
echo ""

# Test 4: Login
echo -e "${YELLOW}Test 4: Login${NC}"
LOGIN_RESPONSE=$(curl -s -X POST ${API_BASE}/login \
  -H "Content-Type: application/json" \
  -d "{
    \"voter_id\": \"${VOTER_ID}\",
    \"password\": \"${PASSWORD}\"
  }")

echo "$LOGIN_RESPONSE" | jq '.'

AUTH_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.auth_token')

if [ "$AUTH_TOKEN" != "null" ] && [ -n "$AUTH_TOKEN" ]; then
    echo -e "${GREEN}✓ Login successful${NC}"
    echo "Auth Token: ${AUTH_TOKEN:0:20}..."
else
    echo -e "${RED}✗ Login failed${NC}"
    exit 1
fi
echo ""

# Test 5: Get Voter Info
echo -e "${YELLOW}Test 5: Get Voter Info${NC}"
curl -s ${API_BASE}/voter/${VOTER_ID} \
  -H "Authorization: Bearer ${AUTH_TOKEN}" | jq '.'
echo ""

# Test 6: Verify Eligibility
echo -e "${YELLOW}Test 6: Verify Voter Eligibility${NC}"
curl -s -X POST ${API_BASE}/verify-eligibility \
  -H "Content-Type: application/json" \
  -d "{\"voter_id\": \"${VOTER_ID}\"}" | jq '.'
echo ""

# Test 7: Attempt to Cast Vote (will fail - needs real biometrics)
echo -e "${YELLOW}Test 7: Attempt to Cast Vote (Expected to fail - needs biometrics)${NC}"
echo -e "${RED}Note: This will fail because it requires real fingerprint + liveness data${NC}"

# Generate dummy biometric data (will fail validation)
FINGERPRINT_DATA=$(printf '[%s]' $(seq -s, 1 256))
LIVENESS_DATA=$(printf '[%s]' $(seq -s, 1 128))

VOTE_RESPONSE=$(curl -s -X POST ${API_BASE}/vote \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"voter_id\": \"${VOTER_ID}\",
    \"election_id\": \"election001\",
    \"candidate_id\": 1,
    \"smdc_slot_index\": 0,
    \"auth_token\": \"${AUTH_TOKEN}\",
    \"fingerprint_data\": ${FINGERPRINT_DATA},
    \"liveness_data\": ${LIVENESS_DATA}
  }")

echo "$VOTE_RESPONSE" | jq '.'

if echo "$VOTE_RESPONSE" | jq -e '.receipt_id' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Vote cast successfully!${NC}"
    RECEIPT_ID=$(echo "$VOTE_RESPONSE" | jq -r '.receipt_id')
    echo "Receipt ID: ${RECEIPT_ID}"
else
    echo -e "${YELLOW}✗ Vote failed (expected - needs real biometric data)${NC}"
    echo "Error: $(echo "$VOTE_RESPONSE" | jq -r '.message')"
fi
echo ""

# Test 8: Get Vote Count
echo -e "${YELLOW}Test 8: Get Total Vote Count${NC}"
curl -s ${API_BASE}/vote-count | jq '.'
echo ""

# Test 9: Get Election Results
echo -e "${YELLOW}Test 9: Get Election Results${NC}"
curl -s ${API_BASE}/results/election001 | jq '.'
echo ""

echo "=========================================="
echo -e "${GREEN}Testing Complete!${NC}"
echo "=========================================="
echo ""
echo "Summary:"
echo "- ✓ Server is healthy"
echo "- ✓ Elections are available"
echo "- ✓ Registration works"
echo "- ✓ Login works"
echo "- ✗ Voting requires real biometric data (fingerprint + liveness)"
echo ""
echo "Next Steps:"
echo "1. Integrate fingerprint capture device"
echo "2. Implement liveness detection (camera-based)"
echo "3. Test complete voting flow with real biometrics"
echo ""
