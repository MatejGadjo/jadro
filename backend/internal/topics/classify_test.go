package topics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustClassifierFromDefaultConfig(t *testing.T) *Classifier {
	t.Helper()
	classifier, err := loadClassifierFromFS(classifierConfigFS)
	require.NoError(t, err)
	return classifier
}

func TestClassifyArticle_FeedMappingWins(t *testing.T) {
	classifier, err := newClassifier(
		feedMapConfig{FeedIDs: map[string]string{"42": "SCENA"}},
		map[string]string{"спорт": "SPORT"},
		[]urlRuleConfig{{Pattern: "/(sport)(/|$)", Category: "SPORT", Strength: "strong"}},
		map[string][]keywordConfig{
			"SPORT": {{Pattern: "спорт", Weight: 10}},
			"SCENA": {{Pattern: "ѕвезда", Weight: 10}},
		},
	)
	require.NoError(t, err)

	feedID := int64(42)
	result := classifier.ClassifyArticle(ClassificationInput{
		FeedID:      &feedID,
		FeedURL:     "https://example.com/feed/",
		URL:         "https://example.com/sport/derbi",
		Title:       "спорт вести",
		Description: "ѕвезда",
		RSSCategories: []string{
			"спорт",
		},
	})

	assert.Equal(t, TopicScene, result.Category)
	assert.Equal(t, CategoryMethodFeedMap, result.Method)
	assert.InDelta(t, 0.98, result.Confidence, 0.0001)
}

func TestClassifyArticle_RSSCategoryMapping(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		URL:           "https://example.com/article",
		Title:         "Општ наслов",
		Description:   "Опис",
		RSSCategories: []string{"  Црна хроника! "},
	})

	assert.Equal(t, TopicChronicle, result.Category)
	assert.Equal(t, CategoryMethodRSSCategory, result.Method)
	assert.InDelta(t, 0.90, result.Confidence, 0.0001)
}

func TestClassifyArticle_URLRule(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		URL:         "https://news.example.com/sport/fudbal/123",
		Title:       "Случаен наслов",
		Description: "Без силни клучни зборови",
	})

	assert.Equal(t, TopicSport, result.Category)
	assert.Equal(t, CategoryMethodURLRule, result.Method)
	assert.InDelta(t, 0.88, result.Confidence, 0.0001)
}

func TestClassifyArticle_TechnologyRSSCategoryMapping(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		URL:           "https://example.com/article",
		Title:         "Општ технолошки наслов",
		Description:   "Опис",
		RSSCategories: []string{"Технологија"},
	})

	assert.Equal(t, TopicTechnology, result.Category)
	assert.Equal(t, CategoryMethodRSSCategory, result.Method)
	assert.InDelta(t, 0.90, result.Confidence, 0.0001)
}

func TestClassifyArticle_TechnologyURLRule(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		URL:         "https://news.example.com/tehnologija/ai/123",
		Title:       "Случаен наслов",
		Description: "Без силни клучни зборови",
	})

	assert.Equal(t, TopicTechnology, result.Category)
	assert.Equal(t, CategoryMethodURLRule, result.Method)
	assert.InDelta(t, 0.88, result.Confidence, 0.0001)
}

func TestClassifyArticle_SKOPJEvsMAKEDONIJA(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		URL:         "https://example.com/vesti/123",
		Title:       "Скопје добива нова автобуска линија",
		Description: "Владата најави поддршка за проектот",
	})

	assert.Equal(t, TopicSkopje, result.Category)
	assert.Equal(t, CategoryMethodKeywordRule, result.Method)
	assert.Equal(t, "skopje_strong", result.Signals["overrideApplied"])
}

func TestClassifyArticle_HRONIKAOverride(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		URL:         "https://example.com/vesti/incident",
		Title:       "Убиство по натпревар во НБА лига",
		Description: "Судот отвори истрага по тепачка",
	})

	assert.Equal(t, TopicChronicle, result.Category)
	assert.Equal(t, CategoryMethodKeywordRule, result.Method)
	assert.Equal(t, "hronika_strong", result.Signals["overrideApplied"])
}

func TestClassifyArticle_REGIONvsSVET(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		URL:         "https://example.com/vesti/analiza",
		Title:       "Србија и Софија со нов договор",
		Description: "НАТО следи безбедносна катастрофа во регионот",
	})

	assert.Equal(t, TopicRegion, result.Category)
	assert.Equal(t, CategoryMethodKeywordRule, result.Method)
}

func TestClassifyArticle_KULTURAvsSCENA(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		URL:         "https://example.com/vesti/feature",
		Title:       "Инфлуенсер промовира нова книга",
		Description: "Публиката ја следи промоцијата.",
	})

	assert.Equal(t, TopicCulture, result.Category)
	assert.Equal(t, CategoryMethodKeywordRule, result.Method)
	assert.Equal(t, "kultura_over_scena", result.Signals["overrideApplied"])
}

func TestClassifyArticle_DeterministicOrdering(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	input := ClassificationInput{
		URL:         "https://example.com/vesti/ambiguous",
		Title:       "Скопје и Владата најавија проект",
		Description: "Министер во Центар зборува за јавен превоз",
	}

	first := classifier.ClassifyArticle(input)
	for i := 0; i < 100; i++ {
		next := classifier.ClassifyArticle(input)
		assert.Equal(t, first.Category, next.Category)
		assert.Equal(t, first.Method, next.Method)
		assert.InDelta(t, first.Confidence, next.Confidence, 0.000001)
	}
}
