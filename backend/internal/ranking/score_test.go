package ranking

import (
	"testing"
	"time"
)

func TestComputeClusterScore(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)
	oneWeekAgo := now.Add(-7 * 24 * time.Hour)

	tests := []struct {
		name               string
		articleCount       int
		newestPublishedAt  *time.Time
		oldestPublishedAt  *time.Time
		uniqueSourceCount  int
		recentArticleCount int
		wantMin            float64
		wantMax            float64
		description        string
	}{
		{
			name:               "zero articles",
			articleCount:       0,
			newestPublishedAt:  nil,
			oldestPublishedAt:  nil,
			uniqueSourceCount:  0,
			recentArticleCount: 0,
			wantMin:            0.0,
			wantMax:            0.0,
			description:        "should return 0 for zero articles",
		},
		{
			// recency=exp(-1/8)≈0.882, novelty=exp(-1/48)≈0.980, diversity=1.0, vel=1.0
			// score ≈ 1.0 × 0.882 × 0.980 × 1.0 × 1.0 ≈ 0.864
			name:               "single article, very recent",
			articleCount:       1,
			newestPublishedAt:  &oneHourAgo,
			oldestPublishedAt:  &oneHourAgo,
			uniqueSourceCount:  1,
			recentArticleCount: 0,
			wantMin:            0.82,
			wantMax:            0.93,
			description:        "should return ~0.86 for single recent article",
		},
		{
			// recency=exp(-3)≈0.050, novelty=exp(-0.5)≈0.607, diversity=1.0, vel=1.0
			// score ≈ 1.0 × 0.050 × 0.607 ≈ 0.030
			name:               "single article, one day old",
			articleCount:       1,
			newestPublishedAt:  &oneDayAgo,
			oldestPublishedAt:  &oneDayAgo,
			uniqueSourceCount:  1,
			recentArticleCount: 0,
			wantMin:            0.020,
			wantMax:            0.045,
			description:        "should return much lower score for 24h-old article with 8h decay",
		},
		{
			// sqrt(10)≈3.162, recency≈0.882, novelty≈0.980, diversity=1.0, vel=1.0
			// score ≈ 3.162 × 0.882 × 0.980 ≈ 2.73
			name:               "multiple articles, very recent",
			articleCount:       10,
			newestPublishedAt:  &oneHourAgo,
			oldestPublishedAt:  &oneHourAgo,
			uniqueSourceCount:  10,
			recentArticleCount: 0,
			wantMin:            2.5,
			wantMax:            3.0,
			description:        "should return higher score for more articles",
		},
		{
			// sqrt(10)≈3.162, recency=exp(-3)≈0.050, novelty≈0.607, diversity=1.0
			// score ≈ 3.162 × 0.050 × 0.607 ≈ 0.096
			name:               "multiple articles, one day old",
			articleCount:       10,
			newestPublishedAt:  &oneDayAgo,
			oldestPublishedAt:  &oneDayAgo,
			uniqueSourceCount:  10,
			recentArticleCount: 0,
			wantMin:            0.07,
			wantMax:            0.13,
			description:        "should be much lower with 8h decay constant at 24h",
		},
		{
			// sqrt(100)=10, recency≈0.882, novelty≈0.980, diversity=1.0
			// score ≈ 10 × 0.882 × 0.980 ≈ 8.64
			name:               "many articles, very recent",
			articleCount:       100,
			newestPublishedAt:  &oneHourAgo,
			oldestPublishedAt:  &oneHourAgo,
			uniqueSourceCount:  100,
			recentArticleCount: 0,
			wantMin:            8.2,
			wantMax:            9.2,
			description:        "should scale with sqrt of article count",
		},
		{
			// sqrt(100)=10, recency=exp(-6)≈0.0025, novelty=exp(-1)≈0.368, diversity=1.0
			// score ≈ 10 × 0.0025 × 0.368 ≈ 0.009
			name:               "many articles, two days old",
			articleCount:       100,
			newestPublishedAt:  &twoDaysAgo,
			oldestPublishedAt:  &twoDaysAgo,
			uniqueSourceCount:  100,
			recentArticleCount: 0,
			wantMin:            0.005,
			wantMax:            0.02,
			description:        "48h-old articles heavily penalised with 8h recency decay",
		},
		{
			// recency=1.0, novelty=1.0, diversity=1.0, vel=1.0 → sqrt(5)≈2.236
			name:               "articles with nil published_at",
			articleCount:       5,
			newestPublishedAt:  nil,
			oldestPublishedAt:  nil,
			uniqueSourceCount:  5,
			recentArticleCount: 0,
			wantMin:            2.1,
			wantMax:            2.4,
			description:        "should use default multipliers when published_at is nil",
		},
		{
			// recency=exp(-168/8)=exp(-21)≈~0, no floor → near zero
			name:               "very old articles",
			articleCount:       5,
			newestPublishedAt:  &oneWeekAgo,
			oldestPublishedAt:  &oneWeekAgo,
			uniqueSourceCount:  5,
			recentArticleCount: 0,
			wantMin:            0.0,
			wantMax:            0.00001,
			description:        "very old articles should score near zero without a recency floor",
		},
		{
			// sqrt(10)≈3.162, recency≈0.882, novelty≈0.980, diversity=0.521
			// score ≈ 3.162 × 0.882 × 0.980 × 0.521 ≈ 1.42
			name:               "low source diversity penalty",
			articleCount:       10,
			newestPublishedAt:  &oneHourAgo,
			oldestPublishedAt:  &oneHourAgo,
			uniqueSourceCount:  1,
			recentArticleCount: 0,
			wantMin:            1.3,
			wantMax:            1.6,
			description:        "should penalize clusters dominated by a single source",
		},
		{
			// same as "multiple articles, very recent" but 5 recent articles
			// velBoost = 1 + 0.5×log(6) ≈ 1 + 0.5×1.792 ≈ 1.896
			// score ≈ 2.73 × 1.896 ≈ 5.18
			name:               "velocity boost from recent articles",
			articleCount:       10,
			newestPublishedAt:  &oneHourAgo,
			oldestPublishedAt:  &oneHourAgo,
			uniqueSourceCount:  10,
			recentArticleCount: 5,
			wantMin:            4.8,
			wantMax:            5.6,
			description:        "recent article activity should boost the score",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeClusterScore(tt.articleCount, tt.newestPublishedAt, tt.oldestPublishedAt, tt.uniqueSourceCount, tt.recentArticleCount)

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("ComputeClusterScore(%d, %v, %v, %d, %d) = %f, want between %f and %f. %s",
					tt.articleCount, tt.newestPublishedAt, tt.oldestPublishedAt, tt.uniqueSourceCount, tt.recentArticleCount, got, tt.wantMin, tt.wantMax, tt.description)
			}
		})
	}
}

func TestComputeClusterScore_Deterministic(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	oneHourAgo := fixedTime.Add(-1 * time.Hour)

	score1 := ComputeClusterScore(10, &oneHourAgo, &oneHourAgo, 10, 0)
	score2 := ComputeClusterScore(10, &oneHourAgo, &oneHourAgo, 10, 0)

	if score1 != score2 {
		t.Errorf("ComputeClusterScore should be deterministic: got %f and %f", score1, score2)
	}
}

func TestComputeClusterScore_Ordering(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	// More articles should score higher
	scoreFew := ComputeClusterScore(5, &oneHourAgo, &oneHourAgo, 5, 0)
	scoreMany := ComputeClusterScore(20, &oneHourAgo, &oneHourAgo, 20, 0)
	if scoreMany <= scoreFew {
		t.Errorf("More articles should score higher: %f (many) <= %f (few)", scoreMany, scoreFew)
	}

	// Newer articles should score higher
	scoreOld := ComputeClusterScore(10, &oneDayAgo, &oneDayAgo, 10, 0)
	scoreNew := ComputeClusterScore(10, &oneHourAgo, &oneHourAgo, 10, 0)
	if scoreNew <= scoreOld {
		t.Errorf("Newer articles should score higher: %f (new) <= %f (old)", scoreNew, scoreOld)
	}

	// Velocity boost should increase the score
	scoreNoVel := ComputeClusterScore(10, &oneHourAgo, &oneHourAgo, 10, 0)
	scoreWithVel := ComputeClusterScore(10, &oneHourAgo, &oneHourAgo, 10, 5)
	if scoreWithVel <= scoreNoVel {
		t.Errorf("Velocity boost should increase score: %f (with vel) <= %f (no vel)", scoreWithVel, scoreNoVel)
	}
}
