package schnorr

import (
	"crypto/rand"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func BenchmarkPointAdd(b *testing.B) {

	p1 := NewPointEmpty()
	res := NewPointEmpty()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res.Add(res, p1)
	}
}

func BenchmarkNewPoint(b *testing.B) {

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewRandomPoint()
	}
}

func BenchmarkHash(b *testing.B) {

	var kbyte [32]byte
	rand.Read(kbyte[:])

	b.ReportAllocs()
	b.ResetTimer()
	chainhash.TaggedHash(
		chainhash.TagBIP0340Challenge, kbyte[:],
	)
}
