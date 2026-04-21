# CovertVote: Production Readiness Audit Report
## সম্পূর্ণ কোড বিশ্লেষণ — VPS Deployment-এর জন্য

**তারিখ:** April 2026
**মোট Go ফাইল:** 72 | **মোট LOC:** 14,807 | **টেস্ট ফাইল:** 19

---

# 🔴 BLOCKER — এগুলো ঠিক না করলে deploy করা যাবে না

## B1. cmd/aggregator-a/ এবং cmd/aggregator-b/ ফোল্ডার নেই
- **সমস্যা:** Makefile, Dockerfile.aggregator, docker-compose.yml সব `cmd/aggregator-a/main.go` এবং `cmd/aggregator-b/main.go` reference করছে — কিন্তু এই ফোল্ডার/ফাইল exist করে না! শুধু `cmd/api-server/` এবং `cmd/cli/` আছে।
- **প্রভাব:** `make build` fail হবে। Docker build fail হবে। SA² servers চালানো অসম্ভব।
- **ফাইল:** `Makefile:15-20`, `Dockerfile.aggregator:18-21`, `docker-compose.yml`

## B2. docker-compose.yml-এ Dockerfile নাম ভুল
- **সমস্যা:** `dockerfile: Dockerfilesorrythi` — এটা একটা typo!
- **প্রভাব:** `docker-compose up` fail হবে।
- **ফাইল:** `docker-compose.yml:12`
- **ফিক্স:** `Dockerfilesorrythi` → `Dockerfile`

## B3. Dockerfile-এ Go version mismatch
- **সমস্যা:** Dockerfile ব্যবহার করছে `golang:1.21-alpine`, কিন্তু go.mod বলছে `go 1.24.0`। Dockerfile.aggregator-ও একই সমস্যা।
- **প্রভাব:** Build fail হতে পারে কারণ Go 1.24 features 1.21-এ নেই।
- **ফাইল:** `Dockerfile:4`, `Dockerfile.aggregator:4`
- **ফিক্স:** `golang:1.21-alpine` → `golang:1.24-alpine`

## B4. Dockerfile-এ config.yaml path ভুল
- **সমস্যা:** `COPY --from=builder /app/pkg/config/config.yaml ./config.yaml` — কিন্তু config.yaml `pkg/config/` এ আছে, আর runtime-এ LoadConfig() এটা পড়ে না (env vars পড়ে)।
- **প্রভাব:** Misleading — config.yaml Docker image-এ copy হচ্ছে কিন্তু ব্যবহার হচ্ছে না।
- **ফাইল:** `Dockerfile:28`

## B5. isAdmin.(bool) PANIC — Server crash করবে
- **সমস্যা:** `registration.go:329` এ `isAdmin.(bool)` — যখন normal voter (AdminAuth ছাড়া) `/voter/:id` endpoint access করবে, `is_admin` context-এ থাকবে না, `nil.(bool)` হবে, **server PANIC করবে!**
- **প্রভাব:** যেকোনো authenticated voter `/api/v1/voter/other-voter-id` call করলে পুরো server crash।
- **ফাইল:** `api/handlers/registration.go:327-329`
- **ফিক্স:**
```go
// আগে (PANIC করবে):
isAdmin, _ := c.Get("is_admin")
if requestedBy != voterID && !isAdmin.(bool) {

// পরে (safe):
isAdminVal, exists := c.Get("is_admin")
isAdmin := exists && isAdminVal.(bool)
if requestedBy != voterID && !isAdmin {
```

## B6. AdminTokens হার্ডকোডেড
- **সমস্যা:** `auth.go:16` এ `"admin-token-example-12345"` হার্ডকোড করা। .env.example-এ `ADMIN_TOKEN` আছে কিন্তু কোড সেটা পড়ে না!
- **প্রভাব:** যেকেউ এই token জানলে admin access পাবে। GitHub public repo তাই সবাই জানে।
- **ফাইল:** `api/middleware/auth.go:14-17`
- **ফিক্স:** `os.Getenv("ADMIN_TOKEN")` থেকে পড়তে হবে, empty হলে server start করবে না।

## B7. Session Token Predictable
- **সমস্যা:** `auth.go:177` এ `SHA256(voterID + time.Now().String())` — time.Now() predictable, voterID জানা, তাই token guess করা সম্ভব।
- **প্রভাব:** Attacker voter-এর session hijack করতে পারে।
- **ফাইল:** `api/middleware/auth.go:174-175`
- **ফিক্স:** `crypto/rand` থেকে 32-byte random token generate করুন।

---

# 🟠 CRITICAL — Production-এ গুরুতর সমস্যা তৈরি করবে

## C1. Private Keys Git-এ আছে (14টি priv_sk ফাইল)
- **সমস্যা:** `network/crypto-config/` ফোল্ডারে 14টি Fabric private key committed আছে। Public repo তাই সবাই দেখতে পারে।
- **ফাইল:** `network/crypto-config/*/keystore/priv_sk`, `network/crypto-config/*/ca/priv_sk`
- **ফিক্স:** 
```bash
# Git history থেকে purge
git filter-branch --force --index-filter \
  "git rm -r --cached --ignore-unmatch network/crypto-config/" \
  --prune-empty -- --all
# .gitignore-এ যোগ করুন
echo "network/crypto-config/" >> .gitignore
# নতুন key generate করুন (পুরানো compromised)
```

## C2. 85MB কম্পাইলড বাইনারি Git-এ tracked
- **সমস্যা:** `bin/api-server` (43MB) এবং `bin/server` (43MB) committed। .gitignore-এ `bin/` আছে কিন্তু আগেই track হয়ে গেছে।
- **ফাইল:** `bin/api-server`, `bin/server`
- **ফিক্স:**
```bash
git rm --cached -r bin/
git commit -m "Remove tracked binaries"
```

## C3. Crypto Error Handling Ignored
- **সমস্যা:** ক্রিপ্টোগ্রাফিক কোডে `rand.Int()` এর error ignore করা হচ্ছে। যদি system-এর entropy pool exhausted হয়, এটা fail করবে কিন্তু কেউ জানবে না।
- **ফাইল:** 
  - `internal/crypto/ring_signature.go:179` — `responses[idx], _ = rand.Int(...)`
  - `internal/crypto/zkproof.go:69` — `d1, _ = rand.Int(...)`
  - `internal/crypto/zkproof.go:107` — `d0, _ = rand.Int(...)`
- **ফিক্স:** Error check করুন এবং error হলে function থেকে return করুন।

## C4. Fingerprint Verification Timing Attack
- **সমস্যা:** `fingerprint.go:57` এ byte-by-byte comparison (`computedHash[i] != storedHash[i]`) ব্যবহার হচ্ছে। Timing side-channel attack-এ attacker hash guess করতে পারে।
- **ফাইল:** `internal/biometric/fingerprint.go:52-60`
- **ফিক্স:** `crypto/subtle.ConstantTimeCompare()` ব্যবহার করুন।

## C5. CORS Allow-Origin: * (সব origin allow)
- **সমস্যা:** `cors.go` এ `Access-Control-Allow-Origin: *` সেট করা — production-এ এটা security risk। `CORSMiddlewareWithOrigins()` function আছে কিন্তু ব্যবহার হচ্ছে না।
- **ফাইল:** `api/middleware/cors.go:11`, `cmd/api-server/main.go:99`
- **ফিক্স:** `CORSMiddlewareWithOrigins([]string{"https://your-domain.com"})` ব্যবহার করুন।

## C6. Rate Limiter Goroutine Leak
- **সমস্যা:** `ratelimit.go:35` এ `go rl.cleanupVisitors()` — infinite loop goroutine, কোনো shutdown mechanism নেই। প্রতিটি `NewRateLimiter()` call-এ নতুন goroutine spawn হয় কিন্তু কখনো বন্ধ হয় না।
- **ফাইল:** `api/middleware/ratelimit.go:35,79-91`
- **ফিক্স:** `context.Context` pass করুন এবং `ctx.Done()` select করুন।

## C7. Config ADMIN_TOKEN পড়ে না
- **সমস্যা:** `.env.example`-এ `ADMIN_TOKEN` আছে কিন্তু `pkg/config/config.go`-এর `LoadConfig()` এটা পড়ে না। Config struct-এও AdminToken field নেই।
- **ফাইল:** `pkg/config/config.go` — `ADMIN_TOKEN`, `JWT_SECRET`, `LOG_LEVEL`, `LOG_FORMAT` কোনোটাই পড়া হয় না।
- **ফিক্স:** Config struct-এ field যোগ করুন এবং LoadConfig()-এ os.Getenv() দিয়ে পড়ুন।

---

# 🟡 HIGH — উন্নত করা দরকার

## H1. In-Memory Only Data Store — Restart-এ সব হারাবে
- **সমস্যা:** `VoteCaster.castVotes`, `VoteCaster.usedKeyImages`, `VoterSessions.sessions`, `VotingHandler.Elections` — সব in-memory map। Server restart করলে সব voter registration, session, ভোট হারিয়ে যাবে।
- **ফাইল:** 
  - `internal/voting/cast.go:51-52` — votes ও key images
  - `api/middleware/auth.go:72` — sessions
  - `api/handlers/voting.go:40` — elections
- **প্রভাব:** VPS-এ server restart (update, crash, OOM) হলে সব data lost।
- **ফিক্স:** SQLite database-এ persist করুন। Migration files আছে (`migrations/`) কিন্তু handler গুলো database ব্যবহার করে না।

## H2. KeyImageStore Interface Implement হয়নি
- **সমস্যা:** `cast.go`-তে `KeyImageStore` interface define করা হয়েছে (FIX_V1-এ), কিন্তু কোনো concrete implementation (SQLite-based) নেই। `cmd/api-server/main.go`-তে `NewVoteCaster()` কে `WithKeyImageStore()` option ছাড়া call করা হচ্ছে, তাই in-memory fallback চলছে।
- **ফাইল:** `internal/voting/cast.go:31-43` — interface আছে, implementation নেই
- **ফিক্স:** `internal/voting/sqlite_keyimage_store.go` তৈরি করুন।

## H3. Voter Registration In-Memory — No Persistence
- **সমস্যা:** `RegistrationSystem.RegisteredVoters` একটা Go map — server restart-এ সব registration হারাবে।
- **ফাইল:** `internal/voter/registration.go:25`
- **ফিক্স:** Database-backed voter store তৈরি করুন।

## H4. Election Management In-Memory
- **সমস্যা:** `main.go`-তে hardcoded sample election তৈরি হচ্ছে। Election database-এ store হচ্ছে না।
- **ফাইল:** `cmd/api-server/main.go:138-151`
- **ফিক্স:** `internal/election/` module-কে database-connected করুন।

## H5. Ring Signature-এ Bubble Sort
- **সমস্যা:** `ring_signature.go:109-114` এ manual bubble sort O(n²) ব্যবহার হচ্ছে।
- **ফাইল:** `internal/crypto/ring_signature.go:109-114`
- **ফিক্স:** `sort.Ints(indices)` ব্যবহার করুন (O(n log n))।

## H6. No Input Sanitization
- **সমস্যা:** `VoterID`, `ElectionID` ইত্যাদি কোনো format validation নেই। SQL injection risk নেই (parameterized query) কিন্তু XSS, path traversal possible।
- **ফাইল:** `api/models/requests.go` — binding:"required" আছে কিন্তু format validation নেই।
- **ফিক্স:** VoterID-তে regex validation যোগ করুন: `binding:"required,alphanum,min=3,max=50"`

## H7. No GitHub Actions CI/CD
- **সমস্যা:** `.github/workflows/` ফোল্ডার নেই। কোনো automated test, lint, build pipeline নেই।
- **ফিক্স:** GitHub Actions workflow তৈরি করুন: test → lint → build → docker push।

## H8. Blockchain Integration Mock Only
- **সমস্যা:** `internal/blockchain/fabric.go` সবসময় `MockMode: true` — real Fabric connection নেই।
- **ফাইল:** `internal/blockchain/fabric.go:30`
- **প্রভাব:** ভোট blockchain-এ record হচ্ছে না — শুধু mock response পাচ্ছে।

---

# 🔵 MEDIUM — উন্নতি করলে ভালো হয়

## M1. Missing Test Files — 8টি module-এ কোনো test নেই
- `internal/voter/` — 0 tests (registration, merkle)
- `internal/blockchain/` — 0 tests
- `internal/election/` — 0 tests
- `internal/database/` — 0 tests
- `pkg/config/` — 0 tests
- `pkg/logger/` — 0 tests
- `pkg/utils/` — 0 tests
- `pkg/audit/` — 0 tests

## M2. Logging fmt.Printf — No Structured Logging
- **সমস্যা:** `logging.go` এবং `logger.go` দুটোই `fmt.Printf` ব্যবহার করে। Production-এ structured JSON logging দরকার (ELK, Grafana integration-এর জন্য)।
- **ফিক্স:** `log/slog` (Go 1.21+) বা `zerolog` ব্যবহার করুন।

## M3. No Graceful Shutdown for Rate Limiter
- **সমস্যা:** `main.go`-তে graceful shutdown আছে HTTP server-এর জন্য, কিন্তু rate limiter-এর cleanup goroutine বন্ধ হয় না।
- **ফিক্স:** Rate limiter-এ context pass করুন, shutdown signal-এ goroutine বন্ধ করুন।

## M4. SQLite MaxOpenConns(1) — Write Bottleneck
- **সমস্যা:** `database/db.go:32` এ `SetMaxOpenConns(1)` — সঠিক (SQLite single writer), কিন্তু concurrent reads-ও block হবে।
- **ফিক্স:** WAL mode enable করুন: `PRAGMA journal_mode=WAL;` — concurrent reads allow করবে।

## M5. Audit Logger Database Connected নয়
- **সমস্যা:** `pkg/audit/audit.go`-তে AuditLogger struct আছে কিন্তু `main.go`-তে initialize/use হচ্ছে না।
- **ফিক্স:** `main.go`-তে audit logger initialize করুন এবং key operations-এ log করুন।

## M6. No HTTPS / TLS Configuration
- **সমস্যা:** Server শুধু HTTP-তে চলে। Production-এ HTTPS দরকার।
- **ফিক্স:** VPS-এ Nginx reverse proxy + Let's Encrypt, অথবা Go-তে `ListenAndServeTLS()` ব্যবহার করুন।

## M7. No Health Check for Database
- **সমস্যা:** `/health` endpoint server uptime দেখায় কিন্তু database connection check করে না।
- **ফাইল:** `api/handlers/health.go`
- **ফিক্স:** `ReadinessCheck`-এ `db.Ping()` যোগ করুন।

---

# 📋 DEPLOYMENT CHECKLIST — VPS-এ deploy করতে হলে

## Phase 1: Blockers Fix (আগে করতে হবে)
- [ ] B1: cmd/aggregator-a/ ও cmd/aggregator-b/ তৈরি করুন
- [ ] B2: docker-compose.yml typo fix
- [ ] B3: Dockerfile Go version update
- [ ] B5: isAdmin panic fix
- [ ] B6: Admin token env var থেকে পড়ুন
- [ ] B7: Secure session token generation

## Phase 2: Security Fixes
- [ ] C1: Private keys git থেকে purge + নতুন generate
- [ ] C2: Binary files git থেকে remove
- [ ] C3: Crypto error handling fix
- [ ] C4: Timing-safe fingerprint comparison
- [ ] C5: CORS restrict করুন
- [ ] C6: Rate limiter graceful shutdown
- [ ] C7: Config সব env vars পড়ুক

## Phase 3: Data Persistence
- [ ] H1: Votes, sessions, elections database-এ persist
- [ ] H2: KeyImageStore SQLite implementation
- [ ] H3: Voter registration persistence
- [ ] H4: Election management via database

## Phase 4: Production Infrastructure
- [ ] H7: GitHub Actions CI/CD
- [ ] M6: Nginx + HTTPS setup
- [ ] M2: Structured logging
- [ ] M7: Database health check
- [ ] Docker image build ও push automation
- [ ] VPS-এ docker-compose deploy script

## Phase 5: Testing
- [ ] M1: Missing module tests লিখুন
- [ ] Integration test: full voting pipeline (register → vote → tally)
- [ ] Load test: concurrent voters
- [ ] SA² server isolation test (separate machines)

---

# VPS Deployment Architecture (প্রস্তাবিত)

```
VPS 1 (API + DB):
├── Nginx (reverse proxy + HTTPS)
├── CovertVote API Server (:8080)
├── SQLite Database
└── Let's Encrypt SSL

VPS 2 (SA² Leader):
├── SA² Aggregator A (:8081)
└── API Key Authentication

VPS 3 (SA² Helper):
├── SA² Aggregator B (:8082)
└── API Key Authentication (different key)

[Important: VPS 2 ও 3 আলাদা provider-এ হলে ভালো — non-collusion assumption]
```

---

# সারসংক্ষেপ

| Category | Count | Status |
|----------|-------|--------|
| 🔴 BLOCKER | 7 | Deploy আটকে আছে |
| 🟠 CRITICAL | 7 | Security risk |
| 🟡 HIGH | 8 | Functionality gap |
| 🔵 MEDIUM | 7 | Quality improvement |
| **মোট** | **29** | **Production-ready নয়** |

**সবচেয়ে জরুরি কাজ:** B1 (aggregator cmd তৈরি), B5 (panic fix), B6 (admin token), এবং H1-H4 (data persistence)।

Data persistence ছাড়া VPS-এ deploy করলে server restart-এ সব ভোট হারিয়ে যাবে — এটাই সবচেয়ে বড় production gap।
