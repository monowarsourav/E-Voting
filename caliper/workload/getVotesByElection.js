'use strict';

const { WorkloadModuleBase } = require('@hyperledger/caliper-core');

class GetElectionWorkload extends WorkloadModuleBase {
    constructor() {
        super();
        this.electionIDs = [];
    }

    async initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext) {
        await super.initializeWorkloadModule(workerIndex, totalWorkers, roundIndex, roundArguments, sutAdapter, sutContext);

        // Create some elections to query
        for (let i = 0; i < 5; i++) {
            const electionID = `election-read-w${workerIndex}-${i}-${Date.now()}`;
            this.electionIDs.push(electionID);

            try {
                await this.sutAdapter.sendRequests({
                    contractId: 'covertvote',
                    contractFunction: 'CreateElection',
                    contractArguments: [
                        electionID,
                        `Read Benchmark Election ${i}`,
                        'For read benchmarks',
                        JSON.stringify(['Candidate A', 'Candidate B', 'Candidate C']),
                        String(Math.floor(Date.now() / 1000) - 3600),
                        String(Math.floor(Date.now() / 1000) + 36000)
                    ],
                    readOnly: false
                });
            } catch (e) {
                // Election may already exist
            }
        }
    }

    async submitTransaction() {
        const idx = Math.floor(Math.random() * this.electionIDs.length);
        const args = {
            contractId: 'covertvote',
            contractFunction: 'GetElection',
            contractArguments: [this.electionIDs[idx]],
            readOnly: true
        };

        await this.sutAdapter.sendRequests(args);
    }
}

function createWorkloadModule() {
    return new GetElectionWorkload();
}

module.exports.createWorkloadModule = createWorkloadModule;
