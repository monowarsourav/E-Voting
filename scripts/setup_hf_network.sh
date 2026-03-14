#!/bin/bash
set -e

echo "========================================="
echo "  CovertVote: HF Test Network Setup"
echo "========================================="

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Docker required"; exit 1; }
# Support both docker-compose (v1) and docker compose (v2 plugin)
if command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    echo "Docker Compose required"; exit 1
fi

cd "$(dirname "$0")/../network"

# Step 1: Download fabric binaries if not present
if [ ! -d "bin" ]; then
    echo "[1/6] Downloading Fabric binaries..."
    curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.5.0 1.5.7 -d -s
    # Binaries are extracted directly into bin/ and config/
else
    echo "[1/6] Fabric binaries already present"
fi

export PATH=$PWD/bin:$PATH
export FABRIC_CFG_PATH=$PWD

# Step 2: Clean previous artifacts and generate fresh crypto material
echo "[2/6] Generating crypto material..."
rm -rf crypto-config channel-artifacts
mkdir -p channel-artifacts
cryptogen generate --config=./crypto-config.yaml --output="crypto-config"

# Step 3: Generate genesis block
echo "[3/6] Generating channel artifacts..."
configtxgen -profile CovertVoteChannel -outputBlock ./channel-artifacts/covertvotechannel.block -channelID covertvotechannel

# Step 4: Start network
echo "[4/6] Starting HF network..."
$DOCKER_COMPOSE -f docker-compose-hf.yml up -d

echo "Waiting for containers to start..."
sleep 5

# Step 5: Create channel using osnadmin (channel participation API)
echo "[5/6] Creating channel via osnadmin..."
export ORDERER_ADMIN_ADDR=localhost:7053

osnadmin channel join \
    --channelID covertvotechannel \
    --config-block ./channel-artifacts/covertvotechannel.block \
    -o $ORDERER_ADMIN_ADDR

# Wait for Raft leader election on single-node cluster
echo "Waiting for Raft leader election..."
sleep 5

# Verify channel is active
CHANNEL_STATUS=$(osnadmin channel list --channelID covertvotechannel -o $ORDERER_ADMIN_ADDR 2>&1 || true)
echo "Channel status: $CHANNEL_STATUS"

# Step 6: Join peers to channel
echo "[6/6] Joining peers to channel..."

export ORDERER_CA=$PWD/crypto-config/ordererOrganizations/covertvote.com/orderers/orderer.covertvote.com/tls/ca.crt

# Fetch genesis block and join peer0.org1
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/crypto-config/peerOrganizations/org1.covertvote.com/peers/peer0.org1.covertvote.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$PWD/crypto-config/peerOrganizations/org1.covertvote.com/users/Admin@org1.covertvote.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export FABRIC_CFG_PATH=$PWD/config

peer channel fetch 0 covertvotechannel.block \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.covertvote.com \
    -c covertvotechannel \
    --tls --cafile $ORDERER_CA

peer channel join -b covertvotechannel.block

# Join peer0.org2
export CORE_PEER_LOCALMSPID=Org2MSP
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/crypto-config/peerOrganizations/org2.covertvote.com/peers/peer0.org2.covertvote.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$PWD/crypto-config/peerOrganizations/org2.covertvote.com/users/Admin@org2.covertvote.com/msp
export CORE_PEER_ADDRESS=localhost:9051

peer channel join -b covertvotechannel.block

# Clean up temp block file
rm -f covertvotechannel.block

echo ""
echo "========================================="
echo "  HF Network Running!"
echo "  Channel: covertvotechannel"
echo "  Orderer: localhost:7050"
echo "  Peer0.Org1: localhost:7051"
echo "  Peer0.Org2: localhost:9051"
echo "========================================="
