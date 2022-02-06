package tracer

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestTrimFuncname(t *testing.T) {
	ss := trimClassFuncname("git.github.com/ocean/internal/controller.(*obj).run2")
	assert.Equal(t, "controller.(*obj).run2", ss)

	ss = trimFuncname("git.github.com/ocean/internal/controller.(*obj).run2")
	assert.Equal(t, "(*obj).run2", ss)
}
