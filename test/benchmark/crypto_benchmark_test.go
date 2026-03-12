// test/benchmark/crypto_benchmark_test.go

package benchmark

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/covertvote/e-voting/internal/crypto"
	"github.com/covertvote/e-voting/internal/sa2"
	"github.com/covertvote/e-voting/internal/smdc"
)

// ============================================================
// PAILLIER BENCHMARKS
// ============================================================

func BenchmarkPaillierKeyGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		crypto.GeneratePaillierKeyPair(2048)
	}
}

func BenchmarkPaillierEncrypt(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	msg := big.NewInt(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.Encrypt(msg)
	}
}

func BenchmarkPaillierDecrypt(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	msg := big.NewInt(42)
	ciphertext, _ := pk.Encrypt(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Decrypt(ciphertext)
	}
}

func BenchmarkPaillierHomomorphicAdd(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	c1, _ := pk.Encrypt(big.NewInt(10))
	c2, _ := pk.Encrypt(big.NewInt(20))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.Add(c1, c2)
	}
}

// ============================================================
// PEDERSEN BENCHMARKS
// ============================================================

func BenchmarkPedersenParamsGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		crypto.GeneratePedersenParams(512)
	}
}

func BenchmarkPedersenCommit(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	msg := big.NewInt(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pp.Commit(msg)
	}
}

// ============================================================
// SMDC BENCHMARKS
// ============================================================

func BenchmarkSMDCGenerate(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewSMDCGenerator(pp, 5, "bench_election")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.GenerateCredential(fmt.Sprintf("voter_%d", i))
	}
}

func BenchmarkSMDCVerify(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewSMDCGenerator(pp, 5, "bench_election")
	cred, _, _ := gen.GenerateCredential("voter")
	pub := cred.GetPublicCredential()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.VerifyCredential(pub)
	}
}

// Different K values
func BenchmarkSMDCGenerateK3(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewSMDCGenerator(pp, 3, "bench_election")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.GenerateCredential("voter")
	}
}

func BenchmarkSMDCGenerateK5(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewSMDCGenerator(pp, 5, "bench_election")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.GenerateCredential("voter")
	}
}

func BenchmarkSMDCGenerateK10(b *testing.B) {
	pp, _ := crypto.GeneratePedersenParams(512)
	gen := smdc.NewSMDCGenerator(pp, 10, "bench_election")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.GenerateCredential("voter")
	}
}

// ============================================================
// SA² BENCHMARKS
// ============================================================

func BenchmarkSA2Split(b *testing.B) {
	sk, _ := crypto.GeneratePaillierKeyPair(2048)
	pk := sk.PublicKey
	sg := sa2.NewVoteSplitter(pk)
	encVote, _ := pk.Encrypt(big.NewInt(1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sg.SplitVote(fmt.Sprintf("voter_%d", i), encVote)
	}
}

// ============================================================
// RING SIGNATURE BENCHMARKS
// ============================================================

func BenchmarkRingSign10(b *testing.B) {
	benchmarkRingSign(b, 10)
}

func BenchmarkRingSign50(b *testing.B) {
	benchmarkRingSign(b, 50)
}

func BenchmarkRingSign100(b *testing.B) {
	benchmarkRingSign(b, 100)
}

func benchmarkRingSign(b *testing.B, ringSize int) {
	rp, _ := crypto.GenerateRingParams(256)
	ring := make([]*big.Int, ringSize)
	keyPairs := make([]*crypto.RingKeyPair, ringSize)
	for i := 0; i < ringSize; i++ {
		keyPairs[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keyPairs[i].PublicKey
	}
	message := []byte("test message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rp.Sign(message, keyPairs[0], ring, 0)
	}
}

func BenchmarkRingVerify100(b *testing.B) {
	rp, _ := crypto.GenerateRingParams(256)
	ringSize := 100
	ring := make([]*big.Int, ringSize)
	keyPairs := make([]*crypto.RingKeyPair, ringSize)
	for i := 0; i < ringSize; i++ {
		keyPairs[i], _ = rp.GenerateRingKeyPair()
		ring[i] = keyPairs[i].PublicKey
	}
	message := []byte("test message")
	sig, _ := rp.Sign(message, keyPairs[0], ring, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rp.Verify(message, sig, ring)
	}
}
