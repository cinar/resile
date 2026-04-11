// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile_test

import (
	"context"
	"testing"

	"github.com/cinar/resile"
)

func BenchmarkDoErr(b *testing.B) {
	ctx := context.Background()
	action := func(ctx context.Context) error { return nil }
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = resile.DoErr(ctx, action)
	}
}

func BenchmarkDo(b *testing.B) {
	ctx := context.Background()
	action := func(ctx context.Context) (int, error) { return 42, nil }
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = resile.Do(ctx, action)
	}
}

func BenchmarkPolicy(b *testing.B) {
	p := resile.NewPolicy()
	ctx := context.Background()
	action := func(ctx context.Context) error { return nil }
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = p.DoErr(ctx, action)
	}
}
