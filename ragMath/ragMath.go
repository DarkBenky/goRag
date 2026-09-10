package ragMath

import "simd"

func DotProductSimd(a, b []float32) float32 {
	lanes := simd.VectorBitSize() / 32
	acc := simd.BroadcastFloat32s(0)

	i := 0
	for ; i+lanes <= len(a); i += lanes {
		acc = simd.LoadFloat32s(a[i:]).MulAdd(simd.LoadFloat32s(b[i:]), acc)
	}

	out := make([]float32, lanes)
	acc.Store(out)

	r := float32(0)
	for j := 0; j < len(out); j++ {
		r += out[j]
	}

	for ; i < len(a); i++ {
		r += a[i] * b[i]
	}
	return r
}

func DotProductsSimd(a []float32, b [][]float32) []float32 {
	result := make([]float32, len(b))
	for i := range b {
		result[i] = DotProductSimd(a, b[i])
	}
	return result
}

func DotProductNative(a, b []float32) float32 {
	r := float32(0)
	for i := 0; i < len(a); i++ {
		r += a[i] * b[i]
	}
	return r
}

func DotProductsNative(a []float32, b [][]float32) []float32 {
	result := make([]float32, len(b))
	for i := range b {
		result[i] = DotProductNative(a, b[i])
	}
	return result
}

func AverageNative(vectors [][]float32) []float32 {
	sum := make([]float32, len(vectors[0]))
	for _, v := range vectors {
		for i := range v {
			sum[i] += v[i]
		}
	}
	for i := range sum {
		sum[i] /= float32(len(vectors))
	}
	return sum
}

func AverageSimd(vectors [][]float32) []float32 {
	rows := len(vectors)
	n := len(vectors[0])
	lanes := simd.VectorBitSize() / 32
	div := simd.BroadcastFloat32s(float32(rows))

	out := make([]float32, n)
	i := 0
	for ; i+lanes <= n; i += lanes {
		acc := simd.BroadcastFloat32s(0)
		for r := 0; r < rows; r++ {
			acc = acc.Add(simd.LoadFloat32s(vectors[r][i:]))
		}
		acc.Div(div).Store(out[i:])
	}
	for ; i < n; i++ {
		var sum float32
		for r := 0; r < rows; r++ {
			sum += vectors[r][i]
		}
		out[i] = sum / float32(rows)
	}
	return out
}
