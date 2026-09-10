package main

import (
	"fmt"
	"simd"
)

func dotProductSimd(a, b simd.Float32s) float32 {
	res := a.Mul(b)
	var sum float32
	for i := 0; i < res.Len(); i++ {
		sum += res[i]
	}
	return sum
}

// TODO: dotProductsSimd

func dotProductNative(a, b []float32) float32 {
	var sum float32
	for i := 0; i < len(a); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

// TODO: dotProductsNative