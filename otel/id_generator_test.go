package otel

import (
	"context"
	"testing"
)

func TestNewObjectId(t *testing.T) {
	oid := NewObjectID()
	t.Log(oid)
	t.Log(oid.Hex())
}

func BenchmarkNewObjectId(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewObjectID().Hex()
	}
}

func BenchmarkGenerator(b *testing.B) {
	idg := NewIDGenerator()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		idg.NewIDs(ctx)
	}
}

func TestTraceSpanGenerator(t *testing.T) {
	idg := NewIDGenerator()
	ctx := context.Background()
	cnt := 10000000
	set := make(map[string]struct{}, cnt)

	for i := 0; i < cnt; i++ {
		oid, _ := idg.NewIDs(ctx)
		hex := oid.String()
		_, ok := set[hex]
		if ok {
			t.Error("conflict", len(set))
			t.Fail()
		}
		set[hex] = struct{}{}
	}
}

func TestGenConflict(t *testing.T) {
	cnt := 10000000
	set := make(map[string]struct{}, cnt)

	for i := 0; i < cnt; i++ {
		oid := NewObjectID().Hex()
		_, ok := set[oid]
		if ok {
			t.Error("conflict", len(set))
			t.Fail()
		}
		set[oid] = struct{}{}
	}
}
