package topics

// Topic represents a canonical taxonomy category.
type Topic string

const (
	TopicMacedonia  Topic = "Македонија"
	TopicEconomy    Topic = "Економија"
	TopicTechnology Topic = "Технологија"
	TopicWorld      Topic = "Свет"
	TopicSkopje     Topic = "Скопје"
	TopicSport      Topic = "Спорт"
	TopicChronicle  Topic = "Хроника"
	TopicCulture    Topic = "Култура"
	TopicScene      Topic = "Сцена"
	TopicRegion     Topic = "Регион"
)

// TopicFallback is used whenever topic signals are missing or inconclusive.
const TopicFallback Topic = TopicRegion

var canonicalTopicOrder = []Topic{
	TopicMacedonia,
	TopicEconomy,
	TopicTechnology,
	TopicWorld,
	TopicSkopje,
	TopicSport,
	TopicChronicle,
	TopicCulture,
	TopicScene,
	TopicRegion,
}

var canonicalTopicNameSet = map[string]struct{}{
	string(TopicMacedonia):  {},
	string(TopicEconomy):    {},
	string(TopicTechnology): {},
	string(TopicWorld):      {},
	string(TopicSkopje):     {},
	string(TopicSport):      {},
	string(TopicChronicle):  {},
	string(TopicCulture):    {},
	string(TopicScene):      {},
	string(TopicRegion):     {},
}

// IsAllowedTopicName reports whether a topic name belongs to the canonical taxonomy.
func IsAllowedTopicName(name string) bool {
	_, ok := canonicalTopicNameSet[name]
	return ok
}

// DefaultFallbackTopic returns the topic used as catch-all.
func DefaultFallbackTopic() Topic {
	return TopicFallback
}

// DefaultFallbackTopicName returns the fallback topic as string.
func DefaultFallbackTopicName() string {
	return string(DefaultFallbackTopic())
}
