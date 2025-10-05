package security

import (
	"crypto/rand"
	"testing"
)

func BenchmarkComputeHMACSHA256(b *testing.B) {
	data := make([]byte, 4<<10)
	_, _ = rand.Read(data)
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ComputeHMACSHA256(data, key)
	}
}
