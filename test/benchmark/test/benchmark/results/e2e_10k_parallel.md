# CovertVote Parallel Benchmark — 10K voters

**Date:** 2026-05-14 21:07:50
**Workers:** 12 (NumCPU)
**Candidates:** 3 (1-hot encoding)

## Results

| Metric | Value |
|--------|-------|
| Voters | 10000 |
| Workers | 12 |
| CredGen (sequential) | 54.312s |
| VoteCast (parallel) | 13m23.348s |
| Per vote (parallel) | 80.335ms |
| Throughput | 12.45 votes/sec |
| Total (cred+cast) | 14m17.66s |
| Cast errors | 0 |
