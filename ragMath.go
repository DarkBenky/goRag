package main

import "simd"

func dotProductSimd(a, b simd.Float32s) float32 {
	res := a.Mul(b)
	buf := make([]float32, res.Len())
	res.Store(buf[:])
	var sum float32
	for _, v := range buf {
		sum += v
	}
	return sum
}

func dotProductsSimdV1(a, b []simd.Float32s) []float32 {
	result := make([]float32, len(a))
	for i := 0; i < len(a); i++ {
		result[i] = dotProductSimd(a[i], b[i])
	}
	return result
}

func dotProductsSimdV2(a, b []simd.Float32s) []float32 {
	lanes := a[0].Len()
	buf := make([]float32, lanes)
	results := make([]float32, len(a))
	for i := range a {
		prod := a[i].Mul(b[i])
		prod.Store(buf)
		for j := 0; j < lanes; j++ {
			results[i] += buf[j]
		}
	}
	return results
}

func dotProductNative(a, b []float32) float32 {
	var sum float32
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

func dotProductsNative(a, b [][]float32) []float32 {
	result := make([]float32, len(a))
	size := len(b[0])
	for i := 0; i < len(a); i++ {
		for j := 0; j < size; j++ {
			result[i] += a[i][j] * b[i][j]
		}
	}
	return result
}
