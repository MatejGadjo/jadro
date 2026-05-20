package topics

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	rssCandidateBoostPerHit = 2.0

	rssCategoryConfidence = 0.90
	urlRuleConfidence     = 0.88
	urlExactConfidence    = 0.92
	keywordHighConfidence = 0.85
	keywordMidConfidence  = 0.75
	fallbackConfidence    = 0.50
)

// CategoryMethod is the deterministic decision path used by classification.
type CategoryMethod string

const (
	CategoryMethodFeedMap     CategoryMethod = "feed_map"
	CategoryMethodRSSCategory CategoryMethod = "rss_category"
	CategoryMethodURLRule     CategoryMethod = "url_rule"
	CategoryMethodKeywordRule CategoryMethod = "keyword_rule"
	CategoryMethodFallback    CategoryMethod = "fallback"
)

// ClassificationInput holds all deterministic signals used for categorization.
type ClassificationInput struct {
	FeedID        *int64
	FeedURL       string
	Domain        string
	URL           string
	Title         string
	Description   string
	RSSCategories []string
}

// ClassificationResult is the output persisted on each article record.
type ClassificationResult struct {
	Category   Topic
	Confidence float64
	Method     CategoryMethod
	Signals    map[string]any
}

type feedMapConfig struct {
	FeedIDs  map[string]string `json:"feed_ids"`
	FeedURLs map[string]string `json:"feed_urls"`
}

type urlRuleConfig struct {
	Pattern    string   `json:"pattern"`
	Category   string   `json:"category"`
	Strength   string   `json:"strength"`
	Domains    []string `json:"domains"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type keywordConfig struct {
	Pattern string  `json:"pattern"`
	Weight  float64 `json:"weight"`
}

type compiledURLRule struct {
	pattern          string
	re               *regexp.Regexp
	category         Topic
	strength         string
	domains          []string
	hasCustomConf    bool
	customConfidence float64
}

type compiledKeyword struct {
	pattern      string
	weight       float64
	alternatives []string
}

type Classifier struct {
	feedIDMap  map[int64]Topic
	feedURLMap map[string]Topic
	rssTagMap  map[string]Topic
	urlRules   []compiledURLRule
	keywords   map[Topic][]compiledKeyword
}

//go:embed config/category_feed_map.json config/category_rss_tag_map.json config/category_url_rules.json config/category_keywords.json
var classifierConfigFS embed.FS

var (
	defaultClassifier = mustLoadDefaultClassifier()
	punctuationRE     = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	whitespaceRE      = regexp.MustCompile(`\s+`)
	diacriticReplacer = strings.NewReplacer(
		"č", "c", "ć", "c", "š", "s", "ž", "z", "đ", "d",
		"á", "a", "à", "a", "â", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "ô", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
	)
)

func mustLoadDefaultClassifier() *Classifier {
	classifier, err := loadClassifierFromFS(classifierConfigFS)
	if err != nil {
		panic(fmt.Sprintf("topics: failed to load classifier config: %v", err))
	}
	return classifier
}

func loadClassifierFromFS(configFS fs.FS) (*Classifier, error) {
	var feedCfg feedMapConfig
	if err := loadJSON(configFS, "config/category_feed_map.json", &feedCfg); err != nil {
		return nil, err
	}

	rssTagMap := make(map[string]string)
	if err := loadJSON(configFS, "config/category_rss_tag_map.json", &rssTagMap); err != nil {
		return nil, err
	}

	var urlRules []urlRuleConfig
	if err := loadJSON(configFS, "config/category_url_rules.json", &urlRules); err != nil {
		return nil, err
	}

	keywordMap := make(map[string][]keywordConfig)
	if err := loadJSON(configFS, "config/category_keywords.json", &keywordMap); err != nil {
		return nil, err
	}

	return newClassifier(feedCfg, rssTagMap, urlRules, keywordMap)
}

func loadJSON(configFS fs.FS, path string, out any) error {
	data, err := fs.ReadFile(configFS, path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func newClassifier(
	feedCfg feedMapConfig,
	rssTagMap map[string]string,
	urlRuleCfg []urlRuleConfig,
	keywordMap map[string][]keywordConfig,
) (*Classifier, error) {
	classifier := &Classifier{
		feedIDMap:  make(map[int64]Topic),
		feedURLMap: make(map[string]Topic),
		rssTagMap:  make(map[string]Topic),
		urlRules:   make([]compiledURLRule, 0, len(urlRuleCfg)),
		keywords:   make(map[Topic][]compiledKeyword),
	}

	for rawFeedID, rawCategory := range feedCfg.FeedIDs {
		feedID, err := strconv.ParseInt(strings.TrimSpace(rawFeedID), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid feed id %q: %w", rawFeedID, err)
		}
		category, ok := parseTopic(rawCategory)
		if !ok {
			return nil, fmt.Errorf("invalid category %q for feed id %q", rawCategory, rawFeedID)
		}
		classifier.feedIDMap[feedID] = category
	}

	for rawFeedURL, rawCategory := range feedCfg.FeedURLs {
		feedURL := normalizeFeedURL(rawFeedURL)
		if feedURL == "" {
			continue
		}
		category, ok := parseTopic(rawCategory)
		if !ok {
			return nil, fmt.Errorf("invalid category %q for feed url %q", rawCategory, rawFeedURL)
		}
		classifier.feedURLMap[feedURL] = category
	}

	for rawTag, rawCategory := range rssTagMap {
		normalizedTag := normalizeToken(rawTag)
		if normalizedTag == "" {
			continue
		}
		category, ok := parseTopic(rawCategory)
		if !ok {
			return nil, fmt.Errorf("invalid category %q for rss tag %q", rawCategory, rawTag)
		}
		classifier.rssTagMap[normalizedTag] = category
	}

	for _, ruleCfg := range urlRuleCfg {
		if strings.TrimSpace(ruleCfg.Pattern) == "" {
			continue
		}

		category, ok := parseTopic(ruleCfg.Category)
		if !ok {
			return nil, fmt.Errorf("invalid url-rule category %q for pattern %q", ruleCfg.Category, ruleCfg.Pattern)
		}

		re, err := regexp.Compile(ruleCfg.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid url-rule pattern %q: %w", ruleCfg.Pattern, err)
		}

		domains := normalizeDomainList(ruleCfg.Domains)
		rule := compiledURLRule{
			pattern:  ruleCfg.Pattern,
			re:       re,
			category: category,
			strength: strings.ToLower(strings.TrimSpace(ruleCfg.Strength)),
			domains:  domains,
		}
		if ruleCfg.Confidence != nil {
			rule.hasCustomConf = true
			rule.customConfidence = clampConfidence(*ruleCfg.Confidence)
		}
		classifier.urlRules = append(classifier.urlRules, rule)
	}

	for rawCategory, entries := range keywordMap {
		category, ok := parseTopic(rawCategory)
		if !ok {
			return nil, fmt.Errorf("invalid keyword category %q", rawCategory)
		}

		compiled := make([]compiledKeyword, 0, len(entries))
		for _, entry := range entries {
			alternatives := splitAlternatives(entry.Pattern)
			if len(alternatives) == 0 {
				continue
			}
			compiled = append(compiled, compiledKeyword{
				pattern:      entry.Pattern,
				weight:       entry.Weight,
				alternatives: alternatives,
			})
		}
		if len(compiled) > 0 {
			classifier.keywords[category] = compiled
		}
	}

	return classifier, nil
}

func normalizeDomainList(rawDomains []string) []string {
	if len(rawDomains) == 0 {
		return nil
	}
	unique := make(map[string]struct{}, len(rawDomains))
	out := make([]string, 0, len(rawDomains))
	for _, rawDomain := range rawDomains {
		domain := normalizeDomain(rawDomain)
		if domain == "" {
			continue
		}
		if _, exists := unique[domain]; exists {
			continue
		}
		unique[domain] = struct{}{}
		out = append(out, domain)
	}
	sort.Strings(out)
	return out
}

func splitAlternatives(pattern string) []string {
	raw := strings.Split(pattern, "|")
	unique := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, candidate := range raw {
		norm := normalizeToken(candidate)
		if norm == "" {
			continue
		}
		if _, exists := unique[norm]; exists {
			continue
		}
		unique[norm] = struct{}{}
		out = append(out, norm)
	}
	return out
}

func parseTopic(raw string) (Topic, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	if IsAllowedTopicName(trimmed) {
		return Topic(trimmed), true
	}

	switch strings.ToUpper(trimmed) {
	case "MAKEDONIJA", "MACEDONIA":
		return TopicMacedonia, true
	case "EKONOMIJA", "ECONOMY":
		return TopicEconomy, true
	case "TEHNOLOGIJA", "TECHNOLOGY", "TECH":
		return TopicTechnology, true
	case "SVET", "WORLD", "INTERNATIONAL":
		return TopicWorld, true
	case "REGION", "BALKAN":
		return TopicRegion, true
	case "SKOPJE":
		return TopicSkopje, true
	case "SPORT":
		return TopicSport, true
	case "HRONIKA", "CHRONICLE", "CRIME":
		return TopicChronicle, true
	case "KULTURA", "CULTURE":
		return TopicCulture, true
	case "SCENA", "SCENE", "ENTERTAINMENT":
		return TopicScene, true
	default:
		return "", false
	}
}

func normalizeFeedURL(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return strings.TrimSuffix(trimmed, "/")
	}

	host := normalizeDomain(parsed.Host)
	path := strings.TrimSuffix(strings.ToLower(parsed.Path), "/")
	if path == "" {
		path = "/"
	}
	return parsed.Scheme + "://" + host + path
}

func normalizeDomain(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}

	if strings.Contains(trimmed, "://") {
		if parsed, err := url.Parse(trimmed); err == nil {
			trimmed = parsed.Host
		}
	}

	if strings.Contains(trimmed, "/") {
		if parsed, err := url.Parse("https://" + trimmed); err == nil && parsed.Host != "" {
			trimmed = parsed.Host
		}
	}

	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	}

	return strings.TrimPrefix(trimmed, "www.")
}

func normalizeToken(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return ""
	}
	normalized = punctuationRE.ReplaceAllString(normalized, " ")
	normalized = whitespaceRE.ReplaceAllString(normalized, " ")
	return strings.TrimSpace(normalized)
}

func simplifyDiacritics(raw string) string {
	return diacriticReplacer.Replace(raw)
}

func normalizeTextVariants(title, description string) []string {
	normalized := normalizeToken(strings.TrimSpace(title + " " + description))
	if normalized == "" {
		return nil
	}

	variants := []string{" " + normalized + " "}
	simplified := simplifyDiacritics(normalized)
	if simplified != normalized {
		variants = append(variants, " "+simplified+" ")
	}
	return variants
}

func containsTerm(variants []string, term string) bool {
	normalizedTerm := normalizeToken(term)
	if normalizedTerm == "" {
		return false
	}
	needle := " " + normalizedTerm + " "
	for _, variant := range variants {
		if strings.Contains(variant, needle) {
			return true
		}
	}
	return false
}

func (ck compiledKeyword) firstMatch(variants []string) string {
	for _, candidate := range ck.alternatives {
		if containsTerm(variants, candidate) {
			return candidate
		}
	}
	return ""
}

func initScoreMap() map[Topic]float64 {
	scores := make(map[Topic]float64, len(canonicalTopicOrder))
	for _, topic := range canonicalTopicOrder {
		scores[topic] = 0
	}
	return scores
}

type rankedScore struct {
	Topic Topic
	Score float64
}

func rankScores(scores map[Topic]float64) []rankedScore {
	ranked := make([]rankedScore, 0, len(canonicalTopicOrder))
	for _, topic := range canonicalTopicOrder {
		ranked = append(ranked, rankedScore{Topic: topic, Score: scores[topic]})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})
	return ranked
}

func scoreSignals(scores map[Topic]float64) map[string]float64 {
	out := make(map[string]float64, len(canonicalTopicOrder))
	for _, topic := range canonicalTopicOrder {
		out[string(topic)] = scores[topic]
	}
	return out
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func classificationResult(category Topic, confidence float64, method CategoryMethod, signals map[string]any) ClassificationResult {
	return ClassificationResult{
		Category:   category,
		Confidence: clampConfidence(confidence),
		Method:     method,
		Signals:    signals,
	}
}

func resolveDomain(input ClassificationInput) string {
	if domain := normalizeDomain(input.Domain); domain != "" {
		return domain
	}
	if parsed, err := url.Parse(input.URL); err == nil {
		if domain := normalizeDomain(parsed.Host); domain != "" {
			return domain
		}
	}
	if feedDomain := normalizeDomain(input.FeedURL); feedDomain != "" {
		return feedDomain
	}
	return ""
}

func isMacedonianDomain(domain string) bool {
	normalized := normalizeDomain(domain)
	if normalized == "" {
		return false
	}
	return normalized == "mk" || strings.HasSuffix(normalized, ".mk")
}

func hasAnyTerm(variants []string, terms []string) bool {
	for _, term := range terms {
		if containsTerm(variants, term) {
			return true
		}
	}
	return false
}

func (c *Classifier) matchURLRule(rawURL, domain string) (ClassificationResult, bool) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ClassificationResult{}, false
	}

	path := strings.ToLower(parsedURL.Path)
	if path == "" {
		path = "/"
	}

	for i, rule := range c.urlRules {
		if !rule.re.MatchString(path) {
			continue
		}

		domainMatched := len(rule.domains) == 0
		if len(rule.domains) > 0 {
			for _, allowed := range rule.domains {
				if domain == allowed || strings.HasSuffix(domain, "."+allowed) {
					domainMatched = true
					break
				}
			}
		}
		if !domainMatched {
			continue
		}

		confidence := urlRuleConfidence
		if rule.hasCustomConf {
			confidence = rule.customConfidence
		} else if len(rule.domains) > 0 && rule.strength == "exact" {
			confidence = urlExactConfidence
		}

		signals := map[string]any{
			"urlRuleMatch": map[string]any{
				"ruleIndex": i,
				"pattern":   rule.pattern,
				"path":      path,
				"strength":  rule.strength,
				"domains":   rule.domains,
			},
		}
		return classificationResult(rule.category, confidence, CategoryMethodURLRule, signals), true
	}

	return ClassificationResult{}, false
}

func singleMatchedTopic(matches map[Topic]int) (Topic, bool) {
	matchedTopic := Topic("")
	count := 0
	for _, topic := range canonicalTopicOrder {
		if matches[topic] > 0 {
			matchedTopic = topic
			count++
		}
	}
	return matchedTopic, count == 1
}

func applyOverride(scores map[Topic]float64, variants []string) (string, Topic, float64, bool) {
	if scores[TopicChronicle] >= 8 {
		return "hronika_strong", TopicChronicle, 0.80, true
	}
	if scores[TopicSport] >= 8 {
		return "sport_strong", TopicSport, 0.80, true
	}
	if scores[TopicSkopje] >= 6 && scores[TopicChronicle] < 8 && scores[TopicSport] < 8 {
		return "skopje_strong", TopicSkopje, 0.78, true
	}
	if scores[TopicRegion] >= 6 && scores[TopicWorld] < scores[TopicRegion]+2 {
		return "region_over_world", TopicRegion, 0.76, true
	}
	if hasAnyTerm(variants, []string{"театар", "музеј", "изложба", "книга", "поезија"}) {
		return "kultura_over_scena", TopicCulture, 0.74, true
	}
	if hasAnyTerm(variants, []string{"ријалити", "инфлуенсер", "ѕвезда", "скандал"}) {
		return "scena_over_kultura", TopicScene, 0.74, true
	}
	return "", "", 0, false
}

// ClassifyArticle runs a deterministic, ML-free 6-step pipeline:
// 1) feed map, 2) RSS category map, 3) URL rules, 4) keyword scoring,
// 5) deterministic overrides, 6) safe fallback.
// Confidence semantics: >=0.90 very strong structural signals; 0.75-0.85 keyword-driven;
// 0.50 fallback/default signal only.
func ClassifyArticle(input ClassificationInput) ClassificationResult {
	return defaultClassifier.ClassifyArticle(input)
}

// ClassifyArticle classifies one RSS item deterministically.
func (c *Classifier) ClassifyArticle(input ClassificationInput) ClassificationResult {
	signals := make(map[string]any)
	domain := resolveDomain(input)
	if domain != "" {
		signals["domain"] = domain
	}

	if input.FeedID != nil {
		if category, ok := c.feedIDMap[*input.FeedID]; ok {
			signals["feedMatch"] = map[string]any{"feedId": *input.FeedID}
			return classificationResult(category, 0.98, CategoryMethodFeedMap, signals)
		}
	}

	if category, ok := c.feedURLMap[normalizeFeedURL(input.FeedURL)]; ok {
		signals["feedMatch"] = map[string]any{"feedUrl": normalizeFeedURL(input.FeedURL)}
		return classificationResult(category, 0.98, CategoryMethodFeedMap, signals)
	}

	scores := initScoreMap()

	if len(input.RSSCategories) > 0 {
		matchedCategories := make(map[Topic]int)
		matchedSignals := make([]map[string]any, 0, len(input.RSSCategories))
		for _, rawCategory := range input.RSSCategories {
			normalized := normalizeToken(rawCategory)
			if normalized == "" {
				continue
			}
			category, ok := c.rssTagMap[normalized]
			if !ok {
				continue
			}
			matchedCategories[category]++
			matchedSignals = append(matchedSignals, map[string]any{
				"raw":        rawCategory,
				"normalized": normalized,
				"category":   string(category),
			})
		}
		if len(matchedSignals) > 0 {
			signals["rssCategoryMatches"] = matchedSignals
		}

		if category, ok := singleMatchedTopic(matchedCategories); ok {
			return classificationResult(category, rssCategoryConfidence, CategoryMethodRSSCategory, signals)
		}

		if len(matchedCategories) > 1 {
			candidateSignals := make(map[string]float64, len(matchedCategories))
			for _, topic := range canonicalTopicOrder {
				hits := matchedCategories[topic]
				if hits == 0 {
					continue
				}
				boost := float64(hits) * rssCandidateBoostPerHit
				scores[topic] += boost
				candidateSignals[string(topic)] = boost
			}
			signals["rssCandidateScores"] = candidateSignals
		}
	}

	if result, ok := c.matchURLRule(input.URL, domain); ok {
		for key, value := range result.Signals {
			signals[key] = value
		}
		result.Signals = signals
		return result
	}

	variants := normalizeTextVariants(input.Title, input.Description)
	keywordMatches := make(map[string][]map[string]any)
	for _, topic := range canonicalTopicOrder {
		for _, keyword := range c.keywords[topic] {
			matchedTerm := keyword.firstMatch(variants)
			if matchedTerm == "" {
				continue
			}
			scores[topic] += keyword.weight
			keywordMatches[string(topic)] = append(keywordMatches[string(topic)], map[string]any{
				"pattern": keyword.pattern,
				"weight":  keyword.weight,
				"matched": matchedTerm,
			})
		}
	}
	if len(keywordMatches) > 0 {
		signals["keywordMatches"] = keywordMatches
	}

	ranked := rankScores(scores)
	top := ranked[0]
	second := ranked[1]
	gap := top.Score - second.Score

	signals["scores"] = scoreSignals(scores)
	signals["topScore"] = map[string]any{"category": string(top.Topic), "score": top.Score}
	signals["secondScore"] = map[string]any{"category": string(second.Topic), "score": second.Score}
	signals["scoreGap"] = gap

	if top.Score >= 10 && gap >= 4 {
		return classificationResult(top.Topic, keywordHighConfidence, CategoryMethodKeywordRule, signals)
	}
	if top.Score >= 7 && gap >= 3 {
		return classificationResult(top.Topic, keywordMidConfidence, CategoryMethodKeywordRule, signals)
	}

	if overrideName, overrideCategory, overrideConfidence, ok := applyOverride(scores, variants); ok {
		signals["overrideApplied"] = overrideName
		return classificationResult(overrideCategory, overrideConfidence, CategoryMethodKeywordRule, signals)
	}

	fallbackCategory := TopicWorld
	fallbackReason := "non_local_or_unknown_domain"
	if isMacedonianDomain(domain) {
		fallbackCategory = TopicMacedonia
		fallbackReason = "mk_domain"
	}
	signals["fallbackReason"] = fallbackReason
	return classificationResult(fallbackCategory, fallbackConfidence, CategoryMethodFallback, signals)
}
