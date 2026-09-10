package main

import (
	"simd"
	"testing"
)

var sink float32
var sinkSlice []float32

const vecCount = 256

func makeVectors() ([]simd.Float32s, [][]float32) {
	lanes := simd.VectorBitSize() / 32
	vecs := make([]simd.Float32s, vecCount)
	rows := make([][]float32, vecCount)
	for i := range vecs {
		row := make([]float32, lanes)
		for j := range row {
			row[j] = float32(i + j + 1)
		}
		rows[i] = row
		vecs[i] = simd.LoadFloat32s(row)
	}
	return vecs, rows
}

func BenchmarkDotProductSimd(b *testing.B) {
	n := simd.VectorBitSize() / 32
	data := make([]float32, n)
	for i := range data {
		data[i] = float32(i)
	}
	va := simd.LoadFloat32s(data)
	vb := simd.LoadFloat32s(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink += dotProductSimd(va, vb)
	}
}

func BenchmarkDotProductNative(b *testing.B) {
	n := simd.VectorBitSize() / 32
	data := make([]float32, n)
	for i := range data {
		data[i] = float32(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink += dotProductNative(data, data)
	}
}

func BenchmarkDotProductsSimdV1(b *testing.B) {
	va, _ := makeVectors()
	vb, _ := makeVectors()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = dotProductsSimdV1(va, vb)
	}
}

func BenchmarkDotProductsSimdV2(b *testing.B) {
	va, _ := makeVectors()
	vb, _ := makeVectors()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = dotProductsSimdV2(va, vb)
	}
}

func BenchmarkDotProductsNative(b *testing.B) {
	_, ra := makeVectors()
	_, rb := makeVectors()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = dotProductsNative(ra, rb)
	}
}
