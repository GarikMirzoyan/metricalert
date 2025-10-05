package metrics

import (
	"context"
	"math/rand"
	"strconv"
	"testing"

	"github.com/GarikMirzoyan/metricalert/internal/constants"
	"github.com/GarikMirzoyan/metricalert/internal/models"
)

func BenchmarkMemStorage_UpdateCounter(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	names := make([]string, 1024)
	for i := range names {
		names[i] = "counter_" + strconv.Itoa(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var idx int
		for pb.Next() {
			name := names[idx&1023]
			idx++
			_ = ms.UpdateCounter(&models.CounterMetric{
				Name:  name,
				Type:  constants.CounterName,
				Value: 1,
			}, ctx)
		}
	})
}

func BenchmarkMemStorage_GetCounter(b *testing.B) {
	ms := NewMemStorage()
	ctx := context.Background()

	for i := 0; i < 10_000; i++ {
		_ = ms.UpdateCounter(&models.CounterMetric{
			Name:  "ctr_" + strconv.Itoa(i),
			Type:  constants.CounterName,
			Value: int64(rand.Intn(10)),
		}, ctx)
	}

	keys := make([]string, 10_000)
	for i := range keys {
		keys[i] = "ctr_" + strconv.Itoa(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int
		for pb.Next() {
			_, _ = ms.GetCounter(keys[i%len(keys)], ctx)
			i++
		}
	})
}
