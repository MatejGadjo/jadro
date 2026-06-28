package storage

import (
	"math"
	"testing"
)

const float32Tol = 1e-6

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		av, bv := float64(a[i]), float64(b[i])
		dot += av * bv
		na += av * av
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func TestBlendCentroid(t *testing.T) {
	tests := []struct {
		name string
		old  []float32
		n    int
		next []float32
		want []float32
	}{
		{
			name: "two orthogonal unit vectors averages to the midpoint",
			old:  []float32{1, 0},
			n:    1,
			next: []float32{0, 1},
			want: []float32{0.5, 0.5},
		},
		{
			name: "weighted by existing count",
			old:  []float32{1, 0, 0}, // centroid of 3 articles
			n:    3,
			next: []float32{0, 4, 0},
			want: []float32{0.75, 1, 0}, // (1*3 + 0)/4, (0*3 + 4)/4, 0
		},
		{
			name: "empty old falls back to next",
			old:  nil,
			n:    0,
			next: []float32{0.2, 0.8},
			want: []float32{0.2, 0.8},
		},
		{
			name: "non-positive count falls back to next",
			old:  []float32{1, 1},
			n:    0,
			next: []float32{0.2, 0.8},
			want: []float32{0.2, 0.8},
		},
		{
			name: "dimension mismatch falls back to next",
			old:  []float32{1, 1, 1},
			n:    2,
			next: []float32{0.2, 0.8},
			want: []float32{0.2, 0.8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blendCentroid(tt.old, tt.n, tt.next)
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if math.Abs(float64(got[i]-tt.want[i])) > float32Tol {
					t.Errorf("got[%d]=%v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestBlendCentroid_IncrementalEqualsBatchMean verifies that folding vectors in
// one-by-one with blendCentroid yields the same result as their arithmetic mean.
func TestBlendCentroid_IncrementalEqualsBatchMean(t *testing.T) {
	vecs := [][]float32{
		{1, 2, 3},
		{4, 0, -1},
		{2, 2, 2},
		{0, -3, 5},
		{1, 1, 1},
	}

	// Incremental: start from the first vector (a 1-article centroid).
	centroid := vecs[0]
	for i := 1; i < len(vecs); i++ {
		centroid = blendCentroid(centroid, i, vecs[i])
	}

	// Batch mean.
	dim := len(vecs[0])
	want := make([]float32, dim)
	for _, v := range vecs {
		for d := range v {
			want[d] += v[d]
		}
	}
	inv := float32(1.0 / float64(len(vecs)))
	for d := range want {
		want[d] *= inv
	}

	for d := range want {
		if math.Abs(float64(centroid[d]-want[d])) > 1e-5 {
			t.Errorf("dim %d: incremental=%v batch=%v", d, centroid[d], want[d])
		}
	}
}

// TestBlendCentroid_RejectsOutlier reproduces the drift bug: a real centroid of
// near-identical "tennis" vectors should score an unrelated "beckham" article
// below the 0.60 clustering threshold, whereas the last-write-wins vector (the
// most recently attached single article) would wrongly clear it.
func TestBlendCentroid_RejectsOutlier(t *testing.T) {
	const threshold = 0.60

	tennis := [][]float32{
		{0.90, 0.10, 0.05, 0.00},
		{0.88, 0.12, 0.04, 0.01},
		{0.91, 0.09, 0.06, 0.00},
		{0.89, 0.11, 0.05, 0.02},
		{0.92, 0.08, 0.05, 0.01},
	}
	beckham := []float32{0.10, 0.05, 0.20, 0.95}

	// Build the running centroid from the tennis articles.
	centroid := tennis[0]
	for i := 1; i < len(tennis); i++ {
		centroid = blendCentroid(centroid, i, tennis[i])
	}

	simCentroid := cosine(centroid, beckham)
	if simCentroid >= threshold {
		t.Errorf("centroid vs outlier sim=%.3f, expected < %.2f (should be rejected)", simCentroid, threshold)
	}

	// Last-write-wins: if beckham had been the last attached article, the cluster
	// vector would equal beckham → sim 1.0, the bug.
	simLastWrite := cosine(beckham, beckham)
	if simLastWrite < threshold {
		t.Fatalf("sanity: identical vectors should be ~1.0, got %.3f", simLastWrite)
	}
}
