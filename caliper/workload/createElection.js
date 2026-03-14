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
