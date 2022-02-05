package tracer

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var (
	seedGenerator = NewRand(time.Now().UnixNano())

	randPool = sync.Pool{
		New: func() interface{} {
			return rand.NewSource(seedGenerator.Int63())
		},
	}
)

func MakeRandomNumber() uint64 {
	generator := randPool.Get().(rand.Source)
	defer randPool.Put(generator)

	return uint64(generator.Int63())
}

func MockID() string {
	rid := MakeRandomNumber()
	return fmt.Sprintf("%016x", rid)
}

type lockedSource struct {
	mu  sync.Mutex
	src rand.Source
}

// NewRand returns a rand.Rand that is threadsafe.
func NewRand(seed int64) *rand.Rand {
	return rand.New(&lockedSource{src: rand.NewSource(seed)})
}

func (r *lockedSource) Int63() (n int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n = r.src.Int63()
	return
}

// Seed implements Seed() of Source
func (r *lockedSource) Seed(seed int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.src.Seed(seed)
}
