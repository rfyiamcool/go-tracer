package otel

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	machineID = loadMachineID()

	randSource     = rand.New(rand.NewSource(time.Now().UnixNano()))
	randSourceLock sync.Mutex

	incr uint32 = rand.Uint32()
)

var _ tracesdk.IDGenerator = &traceIDGenerator{}

type traceIDGenerator struct {
	sync.Mutex
	randSource *rand.Rand
}

func NewIDGenerator() *traceIDGenerator {
	return &traceIDGenerator{
		randSource: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewSpanID returns a non-zero span ID from a randomly-chosen sequence.
func (gen *traceIDGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	gen.Lock()
	defer gen.Unlock()

	sid := trace.SpanID{}
	gen.randSource.Read(sid[:])
	return sid
}

func (gen *traceIDGenerator) makeSpanID() trace.SpanID {
	sid := trace.SpanID{}
	gen.randSource.Read(sid[:])
	return sid
}

func (gen *traceIDGenerator) makeTraceID() trace.TraceID {
	return trace.TraceID(NewObjectID())
}

// NewIDs returns a non-zero trace ID and a non-zero span ID from a
// randomly-chosen sequence.
func (gen *traceIDGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	gen.Lock()
	defer gen.Unlock()

	tid := gen.makeTraceID()
	sid := gen.makeSpanID()
	return tid, sid
}

// object id layout:
// 4byte timestamp sec
// 3byte hostname
// 2byte pid
// 3byte coutner
// 4byte int32 random
type objectID [16]byte

func NewObjectID() objectID {
	var bs [16]byte

	// time
	binary.BigEndian.PutUint32(bs[:], uint32(time.Now().Unix()))

	// machine host
	bs[4] = machineID[0]
	bs[5] = machineID[1]
	bs[6] = machineID[2]

	// pid
	pid := os.Getpid()
	bs[7] = byte(pid >> 8)
	bs[8] = byte(pid)

	// seq
	i := atomic.AddUint32(&incr, 1)
	bs[9] = byte(i >> 16)
	bs[10] = byte(i >> 8)
	bs[11] = byte(i)

	// random
	randSourceLock.Lock()
	rnum := randSource.Uint32()
	randSourceLock.Unlock()
	binary.BigEndian.PutUint32(bs[12:], uint32(rnum))

	return bs
}

func (id objectID) Hex() string {
	return hex.EncodeToString(id[:])
}

func loadMachineID() []byte {
	var (
		err      error
		sum      [3]byte
		hostname string
	)

	machineID := sum[:]
	hostname, err = os.Hostname()
	if err != nil || hostname == "localhost" {
		hostname = randomString(10)
	}

	hw := md5.New()
	hw.Write([]byte(hostname))
	copy(machineID, hw.Sum(nil))

	return machineID
}

func randomString(n int) string {
	randBytes := make([]byte, n/2)
	randSource.Read(randBytes)
	return string(randBytes)
}
