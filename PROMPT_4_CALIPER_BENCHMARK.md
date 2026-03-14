# TASK 4: Hyperledger Caliper Benchmark
# This is a multi-step task. Follow steps in order.

## OVERVIEW
Caliper measures the FULL blockchain pipeline: submit tx → endorsement → ordering → commit.
Your current benchmarks only measure crypto computation (70.5 ms). Caliper will show the
real-world total including blockchain overhead.

## PREREQUISITES
- Docker + Docker Compose installed
- Node.js 18+ installed
- Your chaincode compiles: cd chaincode/covertvote && go build .

---

## STEP 1: Create HF Test Network Config

```
Create a Hyperledger Fabric test network configuration for CovertVote benchmarking.

Create these files:

### FILE: network/docker-compose-hf.yml

version: '3.7'

volumes:
  orderer.covertvote.com:
  peer0.org1.covertvote.com:
  peer0.org2.covertvote.com:

networks:
  covertvote-net:
    name: covertvote-net

services:
  orderer.covertvote.com:
    container_name: orderer.covertvote.com
    image: hyperledger/fabric-orderer:2.5
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_LOGGING_SPEC=INFO
      - ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
      - ORDERER_GENERAL_LISTENPORT=7050
      - ORDERER_GENERAL_LOCALMSPID=OrdererMSP
      - ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp
      - ORDERER_GENERAL_TLS_ENABLED=false
      - ORDERER_GENERAL_BOOTSTRAPMETHOD=none
      - ORDERER_CHANNELPARTICIPATION_ENABLED=true
      - ORDERER_ADMIN_TLS_ENABLED=false
      - ORDERER_ADMIN_LISTENADDRESS=0.0.0.0:7053
    working_dir: /root
    command: orderer
    volumes:
      - ./crypto-config/ordererOrganizations/covertvote.com/orderers/orderer.covertvote.com/msp:/var/hyperledger/orderer/msp
      - orderer.covertvote.com:/var/hyperledger/production/orderer
    ports:
      - "7050:7050"
      - "7053:7053"
    networks:
      - covertvote-net

  peer0.org1.covertvote.com:
    container_name: peer0.org1.covertvote.com
    image: hyperledger/fabric-peer:2.5
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_LOGGING_SPEC=INFO
      - CORE_PEER_ID=peer0.org1.covertvote.com
      - CORE_PEER_ADDRESS=peer0.org1.covertvote.com:7051
      - CORE_PEER_LISTENADDRESS=0.0.0.0:7051
      - CORE_PEER_CHAINCODEADDRESS=peer0.org1.covertvote.com:7052
      - CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052
      - CORE_PEER_GOSSIP_BOOTSTRAP=peer0.org1.covertvote.com:7051
      - CORE_PEER_GOSSIP_EXTERNALENDPOINT=peer0.org1.covertvote.com:7051
      - CORE_PEER_LOCALMSPID=Org1MSP
      - CORE_PEER_TLS_ENABLED=false
      - CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp
      - CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock
      - CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=covertvote-net
    working_dir: /root
    command: peer node start
    volumes:
      - ./crypto-config/peerOrganizations/org1.covertvote.com/peers/peer0.org1.covertvote.com/msp:/etc/hyperledger/fabric/msp
      - peer0.org1.covertvote.com:/var/hyperledger/production
      - /var/run/docker.sock:/host/var/run/docker.sock
    ports:
      - "7051:7051"
    networks:
      - covertvote-net

  peer0.org2.covertvote.com:
    container_name: peer0.org2.covertvote.com
    image: hyperledger/fabric-peer:2.5
    labels:
      service: hyperledger-fabric
    environment:
      - FABRIC_LOGGING_SPEC=INFO
      - CORE_PEER_ID=peer0.org2.covertvote.com
      - CORE_PEER_ADDRESS=peer0.org2.covertvote.com:9051
      - CORE_PEER_LISTENADDRESS=0.0.0.0:9051
      - CORE_PEER_CHAINCODEADDRESS=peer0.org2.covertvote.com:9052
      - CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:9052
      - CORE_PEER_GOSSIP_BOOTSTRAP=peer0.org2.covertvote.com:9051
      - CORE_PEER_GOSSIP_EXTERNALENDPOINT=peer0.org2.covertvote.com:9051
      - CORE_PEER_LOCALMSPID=Org2MSP
      - CORE_PEER_TLS_ENABLED=false
      - CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp
      - CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock
      - CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=covertvote-net
    working_dir: /root
    command: peer node start
    volumes:
      - ./crypto-config/peerOrganizations/org2.covertvote.com/peers/peer0.org2.covertvote.com/msp:/etc/hyperledger/fabric/msp
      - peer0.org2.covertvote.com:/var/hyperledger/production
      - /var/run/docker.sock:/host/var/run/docker.sock
    ports:
      - "9051:9051"
    networks:
      - covertvote-net


### FILE: network/configtx.yaml

Organizations:
  - &OrdererOrg
    Name: OrdererOrg
    ID: OrdererMSP
    MSPDir: crypto-config/ordererOrganizations/covertvote.com/msp
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('OrdererMSP.member')"
      Writers:
        Type: Signature
        Rule: "OR('OrdererMSP.member')"
      Admins:
        Type: Signature
        Rule: "OR('OrdererMSP.admin')"
    OrdererEndpoints:
      - orderer.covertvote.com:7050

  - &Org1
    Name: Org1MSP
    ID: Org1MSP
    MSPDir: crypto-config/peerOrganizations/org1.covertvote.com/msp
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('Org1MSP.admin', 'Org1MSP.peer', 'Org1MSP.client')"
      Writers:
        Type: Signature
        Rule: "OR('Org1MSP.admin', 'Org1MSP.client')"
      Admins:
        Type: Signature
        Rule: "OR('Org1MSP.admin')"
      Endorsement:
        Type: Signature
        Rule: "OR('Org1MSP.peer')"
    AnchorPeers:
      - Host: peer0.org1.covertvote.com
        Port: 7051

  - &Org2
    Name: Org2MSP
    ID: Org2MSP
    MSPDir: crypto-config/peerOrganizations/org2.covertvote.com/msp
    Policies:
      Readers:
        Type: Signature
        Rule: "OR('Org2MSP.admin', 'Org2MSP.peer', 'Org2MSP.client')"
      Writers:
        Type: Signature
        Rule: "OR('Org2MSP.admin', 'Org2MSP.client')"
      Admins:
        Type: Signature
        Rule: "OR('Org2MSP.admin')"
      Endorsement:
        Type: Signature
        Rule: "OR('Org2MSP.peer')"
    AnchorPeers:
      - Host: peer0.org2.covertvote.com
        Port: 9051

Capabilities:
  Channel: &ChannelCapabilities
    V2_0: true
  Orderer: &OrdererCapabilities
    V2_0: true
  Application: &ApplicationCapabilities
    V2_0: true

Application: &ApplicationDefaults
  Organizations:
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
    LifecycleEndorsement:
      Type: ImplicitMeta
      Rule: "MAJORITY Endorsement"
    Endorsement:
      Type: ImplicitMeta
      Rule: "MAJORITY Endorsement"
  Capabilities:
    <<: *ApplicationCapabilities

Orderer: &OrdererDefaults
  OrdererType: etcdraft
  Addresses:
    - orderer.covertvote.com:7050
  BatchTimeout: 2s
  BatchSize:
    MaxMessageCount: 10
    AbsoluteMaxBytes: 99 MB
    PreferredMaxBytes: 512 KB
  Organizations:
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
    BlockValidation:
      Type: ImplicitMeta
      Rule: "ANY Writers"
  Capabilities:
    <<: *OrdererCapabilities
  EtcdRaft:
    Consenters:
      - Host: orderer.covertvote.com
        Port: 7050
        ClientTLSCert: crypto-config/ordererOrganizations/covertvote.com/orderers/orderer.covertvote.com/tls/server.crt
        ServerTLSCert: crypto-config/ordererOrganizations/covertvote.com/orderers/orderer.covertvote.com/tls/server.crt

Channel: &ChannelDefaults
  Policies:
    Readers:
      Type: ImplicitMeta
      Rule: "ANY Readers"
    Writers:
      Type: ImplicitMeta
      Rule: "ANY Writers"
    Admins:
      Type: ImplicitMeta
      Rule: "MAJORITY Admins"
  Capabilities:
    <<: *ChannelCapabilities

Profiles:
  CovertVoteChannel:
    <<: *ChannelDefaults
    Orderer:
      <<: *OrdererDefaults
      Organizations:
        - *OrdererOrg
    Application:
      <<: *ApplicationDefaults
      Organizations:
        - *Org1
        - *Org2


### FILE: network/crypto-config.yaml

OrdererOrgs:
  - Name: Orderer
    Domain: covertvote.com
    EnableNodeOUs: true
    Specs:
      - Hostname: orderer

PeerOrgs:
  - Name: Org1
    Domain: org1.covertvote.com
    EnableNodeOUs: true
    Template:
      Count: 1
    Users:
      Count: 1

  - Name: Org2
    Domain: org2.covertvote.com
    EnableNodeOUs: true
    Template:
      Count: 1
    Users:
      Count: 1
```

---

## STEP 2: Create Caliper Benchmark Configuration

```
Create Caliper benchmark files for CovertVote.

### FILE: caliper/networkconfig.yaml

name: CovertVote Fabric Network
version: "1.0"
mutual-tls: false

caliper:
  blockchain: fabric

channels:
  - channelName: covertvotechannel
    contracts:
      - id: covertvote
        version: "1.0"

organizations:
  - mspid: Org1MSP
    identities:
      certificates:
        - name: 'User1'
          clientPrivateKey:
            path: '../network/crypto-config/peerOrganizations/org1.covertvote.com/users/User1@org1.covertvote.com/msp/keystore/priv_sk'
          clientSignedCert:
            path: '../network/crypto-config/peerOrganizations/org1.covertvote.com/users/User1@org1.covertvote.com/msp/signcerts/User1@org1.covertvote.com-cert.pem'
    connectionProfile:
      path: './connection-org1.yaml'
      discover: true


### FILE: caliper/connection-org1.yaml

name: covertvote-org1
version: "1.0"

client:
  organization: Org1
  connection:
    timeout:
      peer:
        endorser: 300
      orderer: 300

channels:
  covertvotechannel:
    orderers:
      - orderer.covertvote.com
    peers:
      peer0.org1.covertvote.com:
        endorsingPeer: true
        chaincodeQuery: true
        ledgerQuery: true
        eventSource: true

organizations:
  Org1:
    mspid: Org1MSP
    peers:
      - peer0.org1.covertvote.com

orderers:
  orderer.covertvote.com:
    url: grpc://localhost:7050

peers:
  peer0.org1.covertvote.com:
    url: grpc://localhost:7051


### FILE: caliper/benchconfig.yaml

test:
  name: CovertVote Blockchain Benchmark
  description: Measure CastVote, GetVote, and CreateElection performance on Hyperledger Fabric
  workers:
    number: 5
  rounds:
    - label: createElection
      description: Create a test election
      txNumber: 1
      rateControl:
        type: fixed-rate
        opts:
          tps: 1
      workload:
        module: workload/createElection.js

    - label: castVote-fixed10
      description: Cast votes at fixed 10 TPS
      txNumber: 100
      rateControl:
        type: fixed-rate
        opts:
          tps: 10
      workload:
        module: workload/castVote.js

    - label: castVote-fixed50
      description: Cast votes at fixed 50 TPS
      txNumber: 500
      rateControl:
        type: fixed-rate
        opts:
          tps: 50
      workload:
        module: workload/castVote.js

    - label: castVote-fixed100
      description: Cast votes at fixed 100 TPS
      txNumber: 1000
      rateControl:
        type: fixed-rate
        opts:
          tps: 100
      workload:
        module: workload/castVote.js

    - label: getVote
      description: Query vote by ID
      txNumber: 200
      rateControl:
        type: fixed-rate
        opts:
          tps: 50
      workload:
        module: workload/getVote.js

    - label: getVotesByElection
      description: Query all votes for an election
      txNumber: 50
      rateControl:
        type: fixed-rate
        opts:
          tps: 10
      workload:
        module: workload/getVotesByElection.js


### FILE: caliper/workload/createElection.js

'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

class CreateElectionWorkload extends WorkloadModuleBase {
    constructor() {
        super();
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
    }

    async submitTransaction() {
        const electionID = `election-bench-${Date.now()}`;
        const args = {
            contractId: 'covertvote',
            contractFunction: 'CreateElection',
            contractArguments: [
                electionID,
                'Benchmark Election',
                'Performance test election',
                JSON.stringify(['Candidate A', 'Candidate B', 'Candidate C']),
                String(Math.floor(Date.now() / 1000) - 3600),
                String(Math.floor(Date.now() / 1000) + 36000)
            ],
            readOnly: false
        };

        await this.sutAdapter.sendRequests(args);
    }
}

function createWorkloadModule() {
    return new CreateElectionWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;


### FILE: caliper/workload/castVote.js

'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');
const crypto = require('crypto');

class CastVoteWorkload extends WorkloadModuleBase {
    constructor() {
        super();
        this.voteCounter = 0;
        this.electionID = 'election-bench';
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
        this.workerIndex = workerIndex;

        // Create election if first worker
        if (workerIndex === 0) {
            const args = {
                contractId: 'covertvote',
                contractFunction: 'CreateElection',
                contractArguments: [
                    this.electionID,
                    'Caliper Benchmark Election',
                    'Automated benchmark',
                    JSON.stringify(['Candidate A', 'Candidate B']),
                    String(Math.floor(Date.now() / 1000) - 3600),
                    String(Math.floor(Date.now() / 1000) + 36000)
                ],
                readOnly: false
            };
            try {
                await this.sutAdapter.sendRequests(args);
            } catch (e) {
                // Election may already exist
            }
        }
    }

    async submitTransaction() {
        this.voteCounter++;
        const voteID = `vote-w${this.workerIndex}-${this.voteCounter}-${Date.now()}`;
        
        // Simulate encrypted vote data (in real system this would be Paillier ciphertext)
        const encryptedVote = crypto.randomBytes(256).toString('hex');
        const ringSignature = crypto.randomBytes(128).toString('hex');
        const keyImage = crypto.randomBytes(32).toString('hex') + `-${voteID}`;
        const smdcCommitment = crypto.randomBytes(64).toString('hex');
        const merkleProof = crypto.randomBytes(32).toString('hex');

        const args = {
            contractId: 'covertvote',
            contractFunction: 'CastVote',
            contractArguments: [
                voteID,
                this.electionID,
                encryptedVote,
                ringSignature,
                keyImage,
                smdcCommitment,
                merkleProof
            ],
            readOnly: false
        };

        await this.sutAdapter.sendRequests(args);
    }
}

function createWorkloadModule() {
    return new CastVoteWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;


### FILE: caliper/workload/getVote.js

'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

class GetVoteWorkload extends WorkloadModuleBase {
    constructor() {
        super();
        this.voteIDs = [];
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);
        // Pre-create some votes to query
        for (let i = 0; i < 10; i++) {
            const voteID = `query-vote-w${workerIndex}-${i}`;
            this.voteIDs.push(voteID);
        }
    }

    async submitTransaction() {
        const idx = Math.floor(Math.random() * this.voteIDs.length);
        const args = {
            contractId: 'covertvote',
            contractFunction: 'GetVote',
            contractArguments: [this.voteIDs[idx]],
            readOnly: true
        };

        await this.sutAdapter.sendRequests(args);
    }
}

function createWorkloadModule() {
    return new GetVoteWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;


### FILE: caliper/workload/getVotesByElection.js

'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

class GetVotesByElectionWorkload extends WorkloadModuleBase {
    constructor() {
        super();
    }

    async submitTransaction() {
        const args = {
            contractId: 'covertvote',
            contractFunction: 'GetVotesByElection',
            contractArguments: ['election-bench'],
            readOnly: true
        };

        await this.sutAdapter.sendRequests(args);
    }
}

function createWorkloadModule() {
    return new GetVotesByElectionWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
```

---

## STEP 3: Create Setup & Run Scripts

```
Create automation scripts for network setup and Caliper benchmark.

### FILE: scripts/setup_hf_network.sh

#!/bin/bash
set -e

echo "========================================="
echo "  CovertVote: HF Test Network Setup"
echo "========================================="

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Docker required"; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "Docker Compose required"; exit 1; }

cd "$(dirname "$0")/../network"

# Step 1: Download fabric binaries if not present
if [ ! -d "bin" ]; then
    echo "[1/5] Downloading Fabric binaries..."
    curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.5.0 1.5.7 -d -s
    mv fabric-samples/bin .
    mv fabric-samples/config .
    rm -rf fabric-samples
else
    echo "[1/5] Fabric binaries already present"
fi

export PATH=$PWD/bin:$PATH

# Step 2: Generate crypto material
echo "[2/5] Generating crypto material..."
cryptogen generate --config=./crypto-config.yaml --output="crypto-config"

# Step 3: Generate genesis block and channel tx
echo "[3/5] Generating channel artifacts..."
configtxgen -profile CovertVoteChannel -outputBlock ./channel-artifacts/covertvotechannel.block -channelID covertvotechannel

# Step 4: Start network
echo "[4/5] Starting HF network..."
docker-compose -f docker-compose-hf.yml up -d

sleep 5

# Step 5: Create channel and join peers
echo "[5/5] Creating channel and joining peers..."

# Create channel
docker exec peer0.org1.covertvote.com peer channel create \
    -o orderer.covertvote.com:7050 \
    -c covertvotechannel \
    -f /etc/hyperledger/fabric/channel-artifacts/covertvotechannel.block

# Join peers
docker exec peer0.org1.covertvote.com peer channel join -b covertvotechannel.block
docker exec peer0.org2.covertvote.com peer channel join -b covertvotechannel.block

echo ""
echo "========================================="
echo "  HF Network Running!"
echo "========================================="


### FILE: scripts/deploy_chaincode.sh

#!/bin/bash
set -e

echo "========================================="
echo "  Deploying CovertVote Chaincode"
echo "========================================="

export PATH=$PWD/network/bin:$PATH
CC_NAME=covertvote
CC_VERSION=1.0
CC_PATH=../chaincode/covertvote
CHANNEL=covertvotechannel

# Package chaincode
echo "[1/4] Packaging chaincode..."
peer lifecycle chaincode package ${CC_NAME}.tar.gz \
    --path ${CC_PATH} \
    --lang golang \
    --label ${CC_NAME}_${CC_VERSION}

# Install on Org1
echo "[2/4] Installing on Org1..."
docker exec peer0.org1.covertvote.com peer lifecycle chaincode install /opt/gopath/src/${CC_NAME}.tar.gz

# Install on Org2
echo "[3/4] Installing on Org2..."
docker exec peer0.org2.covertvote.com peer lifecycle chaincode install /opt/gopath/src/${CC_NAME}.tar.gz

# Approve and commit
echo "[4/4] Approving and committing..."
PACKAGE_ID=$(docker exec peer0.org1.covertvote.com peer lifecycle chaincode queryinstalled | grep ${CC_NAME} | awk '{print $3}' | sed 's/,$//')

docker exec peer0.org1.covertvote.com peer lifecycle chaincode approveformyorg \
    -o orderer.covertvote.com:7050 \
    --channelID ${CHANNEL} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --package-id ${PACKAGE_ID} \
    --sequence 1

docker exec peer0.org2.covertvote.com peer lifecycle chaincode approveformyorg \
    -o orderer.covertvote.com:7050 \
    --channelID ${CHANNEL} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --package-id ${PACKAGE_ID} \
    --sequence 1

docker exec peer0.org1.covertvote.com peer lifecycle chaincode commit \
    -o orderer.covertvote.com:7050 \
    --channelID ${CHANNEL} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --sequence 1

echo ""
echo "========================================="
echo "  Chaincode Deployed!"
echo "========================================="


### FILE: scripts/run_caliper.sh

#!/bin/bash
set -e

echo "========================================="
echo "  CovertVote: Caliper Benchmark"
echo "========================================="

cd "$(dirname "$0")/../caliper"

# Install Caliper CLI
if ! command -v npx caliper >/dev/null 2>&1; then
    echo "[1/3] Installing Caliper..."
    npm install --only=prod @hyperledger/caliper-cli@0.6.0
    npx caliper bind --caliper-bind-sut fabric:2.5
else
    echo "[1/3] Caliper already installed"
fi

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


### FILE: scripts/cleanup_hf.sh

#!/bin/bash
echo "Stopping and cleaning HF network..."
cd "$(dirname "$0")/../network"
docker-compose -f docker-compose-hf.yml down -v
docker rm -f $(docker ps -aq --filter label=service=hyperledger-fabric) 2>/dev/null || true
docker rmi -f $(docker images -q --filter reference='dev-peer*') 2>/dev/null || true
echo "Done!"


Make all scripts executable:
chmod +x scripts/setup_hf_network.sh
chmod +x scripts/deploy_chaincode.sh
chmod +x scripts/run_caliper.sh
chmod +x scripts/cleanup_hf.sh

Create needed directories:
mkdir -p network/channel-artifacts
mkdir -p caliper/workload

After creating all files, commit:
git add network/ caliper/ scripts/
git commit -m "Add Hyperledger Caliper benchmark setup: HF test network, workload modules, automation scripts"
git push
```

---

## STEP 4: Run Everything (on your machine)

```bash
# 1. Setup HF network
bash scripts/setup_hf_network.sh

# 2. Deploy chaincode
bash scripts/deploy_chaincode.sh

# 3. Run Caliper benchmark
bash scripts/run_caliper.sh

# 4. When done, cleanup
bash scripts/cleanup_hf.sh
```

## Expected Results

| Round | TPS Target | Expected Throughput | Expected Latency |
|-------|:----------:|:-------------------:|:----------------:|
| CastVote @10 TPS | 10 | ~8-10 TPS | ~200-500 ms |
| CastVote @50 TPS | 50 | ~30-50 TPS | ~300-800 ms |
| CastVote @100 TPS | 100 | ~50-80 TPS | ~500-2000 ms |
| GetVote (query) | 50 | ~40-50 TPS | ~50-100 ms |

## Paper Statement

"With Hyperledger Fabric 2.5 (1 orderer, 2 peers, Raft consensus, batch timeout 2s), CovertVote achieves X TPS for vote submission with average latency of Y ms. Combined with the 70.5 ms cryptographic computation, the total per-vote pipeline time is approximately Z ms."
