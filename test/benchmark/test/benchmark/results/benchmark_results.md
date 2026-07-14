# CovertVote Benchmark Results

**Date:** 2026-05-13 00:14:54
**System:** Go E-Voting System

## Performance Table

| Voters | Total Time | Per Vote | Cred Gen | Vote Cast | Aggregate | Decrypt |
|--------|------------|----------|----------|-----------|-----------|----------|
| 100 | 5.95s | 59.497ms | 513ms | 5.419s | 0s | 17.269ms |
| 1000 | 1m2.527s | 62.527ms | 5.32s | 57.189s | 0s | 17.926ms |
| 10000 | 11m7.086s | 66.709ms | 54.752s | 10m12.317s | 0s | 17.615ms |

## Projections for Large Scale

Based on per-vote time of 66.70861ms:

| Voters | Projected Time |
|--------|----------------|
| 100000 | 1h51m11s |
| 1000000 | 18h31m49s |
| 10000000 | 185h18m6s |
| 50000000 | 926h30m31s |
