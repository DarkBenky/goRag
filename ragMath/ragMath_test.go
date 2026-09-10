package ragMath

import (
	"math"
	"testing"
)

var sink float32
var sinkSlice []float32

const vecSize = 1024
const vecCount = 256

func makeData() []float32 {
	data := make([]float32, vecSize)
	for i := range data {
		data[i] = float32(i + 1)
	}
	return data
}

func makeRows() [][]float32 {
	rows := make([][]float32, vecCount)
	for i := range rows {
		rows[i] = makeData()
	}
	return rows
}

func TestDotProductsMatch(t *testing.T) {
	a := makeData()
	b := makeData()

	got := DotProductSimd(a, b)
	want := DotProductNative(a, b)

	if math.Abs(float64(got-want)) > 1e-3*math.Abs(float64(want)) {
		t.Fatalf("DotProductSimd = %v, DotProductNative = %v", got, want)
	}
}

func TestDotProductsBatchMatch(t *testing.T) {
	query := makeData()
	rows := makeRows()

	got := DotProductsSimd(query, rows)
	want := DotProductsNative(query, rows)

	if len(got) != len(want) {
		t.Fatalf("len(DotProductsSimd) = %d, len(DotProductsNative) = %d", len(got), len(want))
	}

	for i := range want {
		if math.Abs(float64(got[i]-want[i])) > 1e-3*math.Abs(float64(want[i])) {
			t.Fatalf("row %d: DotProductsSimd = %v, DotProductsNative = %v", i, got[i], want[i])
		}
	}
}

func TestAverageMatch(t *testing.T) {
	rows := makeRows()

	got := AverageSimd(rows)
	want := AverageNative(rows)

	if len(got) != len(want) {
		t.Fatalf("len(AverageSimd) = %d, len(AverageNative) = %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: AverageSimd = %v, AverageNative = %v", i, got[i], want[i])
		}
	}
}

func BenchmarkDotProductSimd(b *testing.B) {
	left := makeData()
	right := makeData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink += DotProductSimd(left, right)
	}
}

func BenchmarkDotProductNative(b *testing.B) {
	left := makeData()
	right := makeData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink += DotProductNative(left, right)
	}
}

func BenchmarkDotProductsSimd(b *testing.B) {
	query := makeData()
	rows := makeRows()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = DotProductsSimd(query, rows)
	}
}

func BenchmarkDotProductsNative(b *testing.B) {
	query := makeData()
	rows := makeRows()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = DotProductsNative(query, rows)
	}
}

func BenchmarkAverageSimd(b *testing.B) {
	rows := makeRows()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = AverageSimd(rows)
	}
}

func BenchmarkAverageNative(b *testing.B) {
	rows := makeRows()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = AverageNative(rows)
	}
}
