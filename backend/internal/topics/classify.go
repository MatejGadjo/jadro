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
	"strings"
)

const (
	rssCandidateBoostPerHit = 2.0

	forceConfidence       = 1.0
	rssCategoryConfidence = 0.90
	urlRuleConfidence     = 0.88
	urlExactConfidence    = 0.92
	keywordConfidence     = 0.78
	fallbackConfidence    = 0.50
)

// CategoryMethod is the deterministic decision path used by classification.
type CategoryMethod string

const (
	CategoryMethodForce       CategoryMethod = "force"
	CategoryMethodRSSCategory CategoryMethod = "rss_category"
	CategoryMethodURLRule     CategoryMethod = "url_rule"
	CategoryMethodKeywordRule CategoryMethod = "keyword_rule"
	CategoryMethodFallback    CategoryMethod = "fallback"
)

// ClassificationInput holds all deterministic signals used for categorization.
type ClassificationInput struct {
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

// forcedFeedRule forces a category for feeds whose URL starts with key
// (normalized host/path prefix, e.g. "kajgana.com/sport").
type forcedFeedRule struct {
	key      string
	category Topic
}

type Classifier struct {
	forcedDomains map[string]Topic
	forcedFeeds   []forcedFeedRule
	rssTagMap     map[string]Topic
	urlRules      []compiledURLRule
	keywords      map[Topic][]compiledKeyword
}

//go:embed config/category_rss_tag_map.json config/category_url_rules.json config/category_keywords.json config/source_force_category.json
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

	forceMap := make(map[string]string)
	if err := loadJSON(configFS, "config/source_force_category.json", &forceMap); err != nil {
		return nil, err
	}

	return newClassifier(forceMap, rssTagMap, urlRules, keywordMap)
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
	forceMap map[string]string,
	rssTagMap map[string]string,
	urlRuleCfg []urlRuleConfig,
	keywordMap map[string][]keywordConfig,
) (*Classifier, error) {
	classifier := &Classifier{
		forcedDomains: make(map[string]Topic),
		rssTagMap:     make(map[string]Topic),
		urlRules:      make([]compiledURLRule, 0, len(urlRuleCfg)),
		keywords:      make(map[Topic][]compiledKeyword),
	}

	for rawKey, rawCategory := range forceMap {
		category, ok := parseTopic(rawCategory)
		if !ok {
			return nil, fmt.Errorf("invalid forced category %q for source %q", rawCategory, rawKey)
		}
		if strings.Contains(strings.Trim(rawKey, "/"), "/") {
			key := normalizeSourceKey(rawKey)
			if key == "" {
				return nil, fmt.Errorf("invalid forced source key %q", rawKey)
			}
			classifier.forcedFeeds = append(classifier.forcedFeeds, forcedFeedRule{key: key, category: category})
			continue
		}
		domain := normalizeDomain(rawKey)
		if domain == "" {
			return nil, fmt.Errorf("invalid forced source domain %q", rawKey)
		}
		classifier.forcedDomains[domain] = category
	}
	sort.Slice(classifier.forcedFeeds, func(i, j int) bool {
		return classifier.forcedFeeds[i].key < classifier.forcedFeeds[j].key
	})

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

// normalizeSourceKey normalizes a URL (or bare host/path) to "host/path"
// form — lowercase, no scheme, no www, no trailing slash — so forced-feed
// prefixes like "kajgana.com/sport" can match feed URLs.
func normalizeSourceKey(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return ""
	}

	host := parsed.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimPrefix(host, "www.")

	return host + strings.TrimSuffix(parsed.Path, "/")
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

// maxScore returns the highest-scoring topic; ties resolve to the earlier
// topic in canonicalTopicOrder so results are deterministic.
func maxScore(scores map[Topic]float64) (Topic, float64) {
	top := canonicalTopicOrder[0]
	topScore := scores[top]
	for _, topic := range canonicalTopicOrder[1:] {
		if scores[topic] > topScore {
			top = topic
			topScore = scores[topic]
		}
	}
	return top, topScore
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

// matchForcedSource checks single-topic sources: feed-URL prefixes first
// (multi-feed domains like kajgana.com), then plain domains.
func (c *Classifier) matchForcedSource(feedURL, domain string) (Topic, string, bool) {
	if feedKey := normalizeSourceKey(feedURL); feedKey != "" {
		for _, rule := range c.forcedFeeds {
			if feedKey == rule.key || strings.HasPrefix(feedKey, rule.key+"/") {
				return rule.category, rule.key, true
			}
		}
	}
	if category, ok := c.forcedDomains[domain]; ok {
		return category, domain, true
	}
	return "", "", false
}

func (c *Classifier) matchURLRule(rawURL, domain string) (ClassificationResult, bool) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ClassificationResult{}, false
	}

	path := parsedURL.Path
	// Cyrillic URL segments arrive percent-encoded (e.g. /%D1%81%D0%BF%D0%BE%D1%80%D1%82/);
	// decode so Cyrillic patterns can match.
	if decoded, err := url.PathUnescape(path); err == nil {
		path = decoded
	}
	path = strings.ToLower(path)
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

// ClassifyArticle runs a deterministic, ML-free pipeline:
// 0) forced single-topic sources, 1) RSS category map, 2) URL rules,
// 3) keyword scoring (max-score winner), 4) safe fallback.
// Confidence semantics: 1.0 forced source; >=0.88 strong structural signals;
// 0.78 keyword-driven; 0.50 fallback/default signal only.
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

	if category, matchedKey, ok := c.matchForcedSource(input.FeedURL, domain); ok {
		signals["forcedSource"] = matchedKey
		return classificationResult(category, forceConfidence, CategoryMethodForce, signals)
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

	topTopic, topScore := maxScore(scores)

	signals["scores"] = scoreSignals(scores)
	signals["topScore"] = map[string]any{"category": string(topTopic), "score": topScore}

	if topScore > 0 {
		return classificationResult(topTopic, keywordConfidence, CategoryMethodKeywordRule, signals)
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
