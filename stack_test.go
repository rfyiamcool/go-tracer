package tracer

import (
	"testing"
)

func BenchmarkCaller(b *testing.B) {
	for n := 0; n < b.N; n++ {
		GetCaller()
	}
}

func BenchmarkCallerCache(b *testing.B) {
	for n := 0; n < b.N; n++ {
		GetCallerCache()
	}
}
