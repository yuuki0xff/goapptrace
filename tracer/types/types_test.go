package types

import (
	"math/rand"
	"sync/atomic"
	"testing"
)

func BenchmarkNewTxID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewTxID()
	}
	b.StopTimer()
}
func BenchmarkRandInt63(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rand.Int63()
	}
	b.StopTimer()
}
func BenchmarkAtomicAddUint64(b *testing.B) {
	var cnt uint64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atomic.AddUint64(&cnt, 1)
	}
	b.StopTimer()
}
func BenchmarkAtomicAddUint32(b *testing.B) {
	var cnt uint32
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atomic.AddUint32(&cnt, 1)
	}
	b.StopTimer()
}
