# CovertVote Benchmark Results

**Date:** 2026-05-09 14:53:43
**System:** Go E-Voting System

## Performance Table

| Voters | Total Time | Per Vote | Cred Gen | Vote Cast | Aggregate | Decrypt |
|--------|------------|----------|----------|-----------|-----------|----------|
| 100 | 6.776s | 67.757ms | 499ms | 6.254s | 0s | 22.9ms |
| 1000 | 1m7.245s | 67.245ms | 4.104s | 1m3.118s | 0s | 23.248ms |
| 10000 | 11m53.666s | 71.367ms | 44.815s | 11m8.825s | 0s | 26.272ms |

## Projections for Large Scale

Based on per-vote time of 71.366584ms:

| Voters | Projected Time |
|--------|----------------|
| 100000 | 1h58m57s |
| 1000000 | 19h49m27s |
| 10000000 | 198h14m26s |
| 50000000 | 991h12m9s |
