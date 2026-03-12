# CovertVote Benchmark Results

**Date:** 2026-01-15 02:00:31
**System:** Go E-Voting System

## Performance Table

| Voters | Total Time | Per Vote | Cred Gen | Vote Cast | Aggregate | Decrypt |
|--------|------------|----------|----------|-----------|-----------|----------|
| 100 | 6.042s | 60.423ms | 389ms | 5.636s | 0s | 17.093ms |
| 1000 | 1m3.432s | 63.432ms | 3.871s | 59.543s | 0s | 18.63ms |
| 10000 | 11m38.981s | 69.898ms | 39.603s | 10m59.355s | 0s | 21.838ms |

## Projections for Large Scale

Based on per-vote time of 69.898056ms:

| Voters | Projected Time |
|--------|----------------|
| 100000 | 1h56m30s |
| 1000000 | 19h24m58s |
| 10000000 | 194h9m41s |
| 50000000 | 970h48m23s |
