package trace

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	oteltrace "go.opentelemetry.io/otel/trace"
	"math/rand"
	"sync"
)

func defaultIDGenerator() randomIDGenerator {
	gen := randomIDGenerator{}
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	gen.randSource = rand.New(rand.NewSource(rngSeed))
	return gen
}

type randomIDGenerator struct {
	sync.Mutex
	randSource *rand.Rand
}

// NewSpanID returns a non-zero span ID from a randomly-chosen sequence.
func (gen *randomIDGenerator) NewSpanID(ctx context.Context, traceID oteltrace.TraceID) oteltrace.SpanID {
	gen.Lock()
	defer gen.Unlock()
	sid := oteltrace.SpanID{}
	_, _ = gen.randSource.Read(sid[:])
	return sid
}

// NewIDs returns a non-zero trace ID and a non-zero span ID from a
// randomly-chosen sequence.
func (gen *randomIDGenerator) NewIDs(ctx context.Context) (oteltrace.TraceID, oteltrace.SpanID) {
	gen.Lock()
	defer gen.Unlock()
	tid := oteltrace.TraceID{}
	_, _ = gen.randSource.Read(tid[:])
	sid := oteltrace.SpanID{}
	_, _ = gen.randSource.Read(sid[:])
	return tid, sid
}
