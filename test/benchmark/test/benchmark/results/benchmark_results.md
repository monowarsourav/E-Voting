# CovertVote Benchmark Results

**Date:** 2026-05-03 11:20:47
**System:** Go E-Voting System

## Performance Table

| Voters | Total Time | Per Vote | Cred Gen | Vote Cast | Aggregate | Decrypt |
|--------|------------|----------|----------|-----------|-----------|----------|
| 100 | 5.925s | 59.254ms | 436ms | 5.472s | 0s | 17.726ms |
| 1000 | 59.153s | 59.153ms | 3.594s | 55.54s | 0s | 18.866ms |
| 10000 | 10m22.046s | 62.205ms | 36.978s | 9m45.05s | 0s | 17.677ms |

## Projections for Large Scale

Based on per-vote time of 62.204571ms:

| Voters | Projected Time |
|--------|----------------|
| 100000 | 1h43m40s |
| 1000000 | 17h16m45s |
| 10000000 | 172h47m26s |
| 50000000 | 863h57m9s |
