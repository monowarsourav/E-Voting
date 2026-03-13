# CovertVote Benchmark Results

**Date:** 2026-03-13 08:28:28
**System:** Go E-Voting System

## Performance Table

| Voters | Total Time | Per Vote | Cred Gen | Vote Cast | Aggregate | Decrypt |
|--------|------------|----------|----------|-----------|-----------|----------|
| 100 | 6.542s | 65.421ms | 365ms | 6.157s | 0s | 20.344ms |
| 1000 | 1m0.32s | 60.32ms | 3.835s | 56.468s | 0s | 17.241ms |
| 10000 | 11m7.928s | 66.793ms | 39.206s | 10m28.704s | 0s | 18.288ms |

## Projections for Large Scale

Based on per-vote time of 66.792794ms:

| Voters | Projected Time |
|--------|----------------|
| 100000 | 1h51m19s |
| 1000000 | 18h33m13s |
| 10000000 | 185h32m8s |
| 50000000 | 927h40m40s |
