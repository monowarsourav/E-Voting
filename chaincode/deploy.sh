#!/bin/bash

# CovertVote Chaincode Deployment Script
# This script deploys the chaincode to Hyperledger Fabric

set -e

# Configuration
CHANNEL_NAME="covertvote-channel"
CHAINCODE_NAME="covertvote-chaincode"
CHAINCODE_VERSION="1.0"
CHAINCODE_SEQUENCE="1"
CHAINCODE_PATH="../chaincode/covertvote"

echo "========================================="
echo "CovertVote Chaincode Deployment"
echo "========================================="
echo "Channel: $CHANNEL_NAME"
echo "Chaincode: $CHAINCODE_NAME"
echo "Version: $CHAINCODE_VERSION"
echo "========================================="

# Step 1: Package the chaincode
echo "Step 1: Packaging chaincode..."
peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz \
    --path ${CHAINCODE_PATH} \
    --lang golang \
    --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}

echo "Chaincode packaged successfully"

# Step 2: Install chaincode on peer
echo "Step 2: Installing chaincode on peer..."
peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz

echo "Getting package ID..."
PACKAGE_ID=$(peer lifecycle chaincode queryinstalled | grep ${CHAINCODE_NAME}_${CHAINCODE_VERSION} | awk '{print $3}' | cut -d',' -f1)
echo "Package ID: $PACKAGE_ID"

# Step 3: Approve chaincode for organization
echo "Step 3: Approving chaincode for organization..."
peer lifecycle chaincode approveformyorg \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --version $CHAINCODE_VERSION \
    --package-id $PACKAGE_ID \
    --sequence $CHAINCODE_SEQUENCE \
    --tls \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem

echo "Chaincode approved"

# Step 4: Check commit readiness
echo "Step 4: Checking commit readiness..."
peer lifecycle chaincode checkcommitreadiness \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --version $CHAINCODE_VERSION \
    --sequence $CHAINCODE_SEQUENCE \
    --tls \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    --output json

# Step 5: Commit chaincode definition
echo "Step 5: Committing chaincode definition..."
peer lifecycle chaincode commit \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --version $CHAINCODE_VERSION \
    --sequence $CHAINCODE_SEQUENCE \
    --tls \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    --peerAddresses localhost:7051 \
    --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
    --peerAddresses localhost:9051 \
    --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt

echo "Chaincode committed"

# Step 6: Query committed chaincode
echo "Step 6: Querying committed chaincode..."
peer lifecycle chaincode querycommitted \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem

echo "========================================="
echo "Chaincode deployment completed!"
echo "========================================="
echo "You can now invoke chaincode functions:"
echo "peer chaincode invoke -o localhost:7050 --channelID $CHANNEL_NAME --name $CHAINCODE_NAME ..."
echo "========================================="
