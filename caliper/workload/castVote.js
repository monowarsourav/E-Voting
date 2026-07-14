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

        // Simulate the 1-hot Paillier ciphertext array — 3 candidates × 2048-bit
        // modulus. Each ciphertext is a decimal string of a random Z_{n^2}
        // element (~1234 chars); the array is JSON-encoded into the
        // encryptedVote argument, matching what
        // internal/blockchain/fabric.go SubmitVote actually sends in the
        // production 1-hot pipeline.
        const NUM_CANDIDATES = 3;
        const ciphertexts = [];
        for (let j = 0; j < NUM_CANDIDATES; j++) {
            const ct = BigInt('0x' + crypto.randomBytes(512).toString('hex')).toString();
            ciphertexts.push(ct);
        }
        const encryptedVote = JSON.stringify(ciphertexts);
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
