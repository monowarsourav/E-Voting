#!/bin/bash
set -e

echo "========================================="
echo "  Deploying CovertVote Chaincode"
echo "========================================="

cd "$(dirname "$0")/../network"

export PATH=$PWD/bin:$PATH
export FABRIC_CFG_PATH=$PWD/config
export GOROOT=/usr/local/go
export PATH=$GOROOT/bin:$PATH

CC_NAME=covertvote
CC_VERSION=1.0
CC_LABEL=${CC_NAME}_${CC_VERSION}
CC_PATH=../chaincode/covertvote
CHANNEL=covertvotechannel
ORDERER_CA=$PWD/crypto-config/ordererOrganizations/covertvote.com/orderers/orderer.covertvote.com/tls/ca.crt

# Helper to set peer environment
set_org1() {
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_LOCALMSPID=Org1MSP
    export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/crypto-config/peerOrganizations/org1.covertvote.com/peers/peer0.org1.covertvote.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=$PWD/crypto-config/peerOrganizations/org1.covertvote.com/users/Admin@org1.covertvote.com/msp
    export CORE_PEER_ADDRESS=localhost:7051
}

set_org2() {
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_LOCALMSPID=Org2MSP
    export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/crypto-config/peerOrganizations/org2.covertvote.com/peers/peer0.org2.covertvote.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=$PWD/crypto-config/peerOrganizations/org2.covertvote.com/users/Admin@org2.covertvote.com/msp
    export CORE_PEER_ADDRESS=localhost:9051
}

# Step 1: Package chaincode
echo "[1/5] Packaging chaincode..."
set_org1
peer lifecycle chaincode package ${CC_NAME}.tar.gz \
    --path ${CC_PATH} \
    --lang golang \
    --label ${CC_LABEL}

# Step 2: Install on both orgs
echo "[2/5] Installing on Org1..."
set_org1
peer lifecycle chaincode install ${CC_NAME}.tar.gz

echo "[3/5] Installing on Org2..."
set_org2
peer lifecycle chaincode install ${CC_NAME}.tar.gz

# Get package ID
set_org1
PACKAGE_ID=$(peer lifecycle chaincode queryinstalled 2>&1 | grep "${CC_LABEL}" | sed -n 's/.*Package ID: \(.*\), Label:.*/\1/p')
echo "Package ID: $PACKAGE_ID"

# Step 3: Approve for both orgs
echo "[4/5] Approving chaincode..."
set_org1
peer lifecycle chaincode approveformyorg \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.covertvote.com \
    --channelID ${CHANNEL} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --package-id ${PACKAGE_ID} \
    --sequence 1 \
    --tls --cafile $ORDERER_CA

set_org2
peer lifecycle chaincode approveformyorg \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.covertvote.com \
    --channelID ${CHANNEL} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --package-id ${PACKAGE_ID} \
    --sequence 1 \
    --tls --cafile $ORDERER_CA

# Step 4: Check commit readiness
peer lifecycle chaincode checkcommitreadiness \
    --channelID ${CHANNEL} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --sequence 1 \
    --output json

# Step 5: Commit chaincode
echo "[5/5] Committing chaincode..."
set_org1
peer lifecycle chaincode commit \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.covertvote.com \
    --channelID ${CHANNEL} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --sequence 1 \
    --tls --cafile $ORDERER_CA \
    --peerAddresses localhost:7051 \
    --tlsRootCertFiles $PWD/crypto-config/peerOrganizations/org1.covertvote.com/peers/peer0.org1.covertvote.com/tls/ca.crt \
    --peerAddresses localhost:9051 \
    --tlsRootCertFiles $PWD/crypto-config/peerOrganizations/org2.covertvote.com/peers/peer0.org2.covertvote.com/tls/ca.crt

# Verify
peer lifecycle chaincode querycommitted --channelID ${CHANNEL} --name ${CC_NAME}

# Cleanup
rm -f ${CC_NAME}.tar.gz

echo ""
echo "========================================="
echo "  Chaincode Deployed!"
echo "========================================="
