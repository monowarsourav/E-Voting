'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');
const crypto = require('crypto');

class GetVoteWorkload extends WorkloadModuleBase {
    constructor() {
        super();
        this.voteIDs = [];
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);

        // Ensure election exists
        if (workerIndex === 0) {
            try {
                await this.sutAdapter.sendRequests({
                    contractId: 'covertvote',
                    contractFunction: 'CreateElection',
                    contractArguments: [
                        'election-query',
                        'Query Benchmark Election',
                        'For read benchmarks',
                        JSON.stringify(['Candidate A', 'Candidate B']),
                        String(Math.floor(Date.now() / 1000) - 3600),
                        String(Math.floor(Date.now() / 1000) + 36000)
                    ],
                    readOnly: false
                });
            } catch (e) {
                // Election may already exist
            }
        }

        // Pre-create votes to query
        for (let i = 0; i < 10; i++) {
            const voteID = `qvote-w${workerIndex}-${i}-${Date.now()}`;
            this.voteIDs.push(voteID);

            const args = {
                contractId: 'covertvote',
                contractFunction: 'CastVote',
                contractArguments: [
                    voteID,
                    'election-query',
                    crypto.randomBytes(64).toString('hex'),
                    crypto.randomBytes(32).toString('hex'),
                    crypto.randomBytes(32).toString('hex') + `-${voteID}`,
                    crypto.randomBytes(32).toString('hex'),
                    crypto.randomBytes(16).toString('hex')
                ],
                readOnly: false
            };

            try {
                await this.sutAdapter.sendRequests(args);
            } catch (e) {
                // Vote may already exist
            }
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
