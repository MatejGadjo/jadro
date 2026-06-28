package storage

// blendCentroid returns the running mean of a cluster's article embeddings:
// (old*n + next)/(n+1), where old is the current centroid of n articles and
// next is the newly attached article's embedding.
//
// The cluster similarity search uses cosine distance (pgvector <=>), which is
// scale-invariant, so the result is intentionally left unnormalized — this
// stores the exact incremental arithmetic mean. It falls back to next (a fresh
// 1-article centroid) when there is no usable prior centroid: empty old, a
// non-positive count, or a dimension mismatch.
func blendCentroid(old []float32, n int, next []float32) []float32 {
	if n <= 0 || len(old) == 0 || len(old) != len(next) {
		return next
	}

	blended := make([]float32, len(next))
	nf := float32(n)
	inv := 1.0 / (nf + 1.0)
	for i := range next {
		blended[i] = (old[i]*nf + next[i]) * inv
	}
	return blended
}
