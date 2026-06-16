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

func TestClassifyArticle_ForcedSourceDomain(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	// Misleading economy keywords + RSS category — forced source must win.
	result := classifier.ClassifyArticle(ClassificationInput{
		FeedURL:       "https://sportmanija.mk/feed/",
		Domain:        "sportmanija.mk",
		URL:           "https://sportmanija.mk/vesti/123",
		Title:         "Банка и инфлација во фокусот",
		Description:   "Економија и берза",
		RSSCategories: []string{"Економија"},
	})

	assert.Equal(t, TopicSport, result.Category)
	assert.Equal(t, CategoryMethodForce, result.Method)
	assert.InDelta(t, 1.0, result.Confidence, 0.0001)
	assert.Equal(t, "sportmanija.mk", result.Signals["forcedSource"])
}

func TestClassifyArticle_ForcedSourceFeedPrefix(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	result := classifier.ClassifyArticle(ClassificationInput{
		FeedURL:     "https://kajgana.com/sport/rss",
		Domain:      "kajgana.com",
		URL:         "https://kajgana.com/nekoja-statija",
		Title:       "Општ наслов",
		Description: "Опис",
	})

	assert.Equal(t, TopicSport, result.Category)
	assert.Equal(t, CategoryMethodForce, result.Method)
	assert.InDelta(t, 1.0, result.Confidence, 0.0001)
	assert.Equal(t, "kajgana.com/sport", result.Signals["forcedSource"])
}

func TestClassifyArticle_NonForcedFeedOnForcedPrefixDomain(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	// kajgana.com/vesti is not forced — must fall through to the classifier.
	result := classifier.ClassifyArticle(ClassificationInput{
		FeedURL:       "https://kajgana.com/vesti/rss",
		Domain:        "kajgana.com",
		URL:           "https://kajgana.com/nekoja-statija",
		Title:         "Општ наслов",
		Description:   "Опис",
		RSSCategories: []string{"Економија"},
	})

	assert.Equal(t, TopicEconomy, result.Category)
	assert.Equal(t, CategoryMethodRSSCategory, result.Method)
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

func TestClassifyArticle_CyrillicURLRule(t *testing.T) {
	classifier := mustClassifierFromDefaultConfig(t)

	// Percent-encoded /спорт/ path segment.
	result := classifier.ClassifyArticle(ClassificationInput{
		URL:         "https://news.example.com/%D1%81%D0%BF%D0%BE%D1%80%D1%82/fudbal/123",
		Title:       "Случаен наслов",
		Description: "Без силни клучни зборови",
	})

	assert.Equal(t, TopicSport, result.Category)
	assert.Equal(t, CategoryMethodURLRule, result.Method)
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
