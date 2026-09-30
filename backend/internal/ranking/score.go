package ranking

import (
	"math"
	"time"
)

// ComputeClusterScore calculates a score for a cluster based on article count, recency,
// novelty decay, source diversity, and recent article velocity.
// The formula combines:
//   - Article count component: sqrt(articleCount) to reward clusters with more articles
//   - Recency component: exponential decay based on hours since the newest article (8h half-life)
//   - Novelty component: exponential decay based on hours since the oldest article (48h half-life)
//   - Diversity component: penalty for low source diversity
//   - Velocity boost: reward for stories actively being covered in the last 6 hours
//
// Parameters:
//   - articleCount: number of articles in the cluster
//   - newestPublishedAt: published_at timestamp of the newest article (can be nil)
//   - oldestPublishedAt: published_at timestamp of the oldest article (can be nil)
//   - uniqueSourceCount: number of distinct sources in the cluster
//   - recentArticleCount: number of articles added to the cluster in the last 6 hours
//
// Returns a float64 score where higher values indicate more important/relevant clusters.
// The function is deterministic and does not access the database.
func ComputeClusterScore(articleCount int, newestPublishedAt *time.Time, oldestPublishedAt *time.Time, uniqueSourceCount int, recentArticleCount int) float64 {
	if articleCount <= 0 {
		return 0.0
	}

	// Article count component: use square root to avoid linear growth
	articleComponent := math.Sqrt(float64(articleCount))

	now := time.Now()

	// Recency component: 8h decay constant — stories from this morning feel old by evening.
	// No floor: let old stories decay to zero so they stop blocking the feed.
	recencyMultiplier := 1.0
	if newestPublishedAt != nil {
		hoursSinceNewest := now.Sub(*newestPublishedAt).Hours()
		recencyMultiplier = math.Exp(-hoursSinceNewest / 8.0)
	}

	// Novelty component: 48h decay constant — a story older than 2 days should nearly disappear.
	noveltyMultiplier := 1.0
	if oldestPublishedAt != nil {
		hoursSinceOldest := now.Sub(*oldestPublishedAt).Hours()
		if hoursSinceOldest < 0 {
			hoursSinceOldest = 0
		}
		noveltyMultiplier = math.Exp(-hoursSinceOldest / 48.0)
		if noveltyMultiplier < 0.05 {
			noveltyMultiplier = 0.05
		}
	}

	// Diversity component: penalize clusters dominated by a single source.
	if uniqueSourceCount <= 0 {
		uniqueSourceCount = 1
	}
	if uniqueSourceCount > articleCount {
		uniqueSourceCount = articleCount
	}
	diversityRatio := float64(uniqueSourceCount) / float64(articleCount)
	diversityMultiplier := 0.3 + 0.7*math.Sqrt(diversityRatio)

	// Velocity boost: reward stories actively being covered right now.
	// log(1+n) gives diminishing returns: 0 recent → 1.0x, 1 recent → 1.35x, 5 recent → 1.90x, 10 recent → 2.20x
	velBoost := 1.0 + 0.5*math.Log(1.0+float64(recentArticleCount))

	return articleComponent * recencyMultiplier * noveltyMultiplier * diversityMultiplier * velBoost
}
