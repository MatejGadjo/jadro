import type { ClusterResponse, HomepageClusterSection } from '@/api/types'
import type { FeedArticle, FeedCluster, TopicArticle, TopicSection } from '@/lib/feedTypes'
import { mapClusterResponseArticles, toRelativeTime } from '@/lib/clusterArticles'
import { HOMEPAGE_TOPICS, TOPIC_ITEMS } from '@/lib/topics'


function mapHomepageArticle(
  article: HomepageClusterSection['sources'][number]['articles'][number],
  source: HomepageClusterSection['sources'][number]['source'],
): FeedArticle {
  return {
    id: article.id,
    source: {
      id: source.id,
      name: source.name,
      logo_url: source.logo_url,
    },
    title: article.title,
    url: article.url,
    publishedAt: toRelativeTime(article.published_at, article.fetched_at),
    ...(article.image_url ? { imageUrl: article.image_url } : {}),
  }
}

export function mapFeaturedSectionsToStoryClusters(sections: HomepageClusterSection[]): FeedCluster[] {
  return sections
    .map<FeedCluster | null>((section) => {
      const articles = section.sources
        .map((group) => {
          const leadArticle = group.articles.find((article) => Boolean(article.image_url)) ?? group.articles[0]
          if (!leadArticle) {
            return null
          }
          return mapHomepageArticle(leadArticle, group.source)
        })
        .filter((article): article is FeedArticle => article !== null)

      if (articles.length === 0) {
        return null
      }

      const mainArticleIndex = articles.findIndex((article) => Boolean(article.imageUrl))
      const orderedArticles = [...articles]
      if (mainArticleIndex > 0) {
        const [articleWithImage] = orderedArticles.splice(mainArticleIndex, 1)
        if (articleWithImage) {
          orderedArticles.unshift(articleWithImage)
        }
      }

      const [mainArticle, ...otherArticles] = orderedArticles

      return {
        id: section.cluster.id,
        headline: section.cluster.headline,
        topic: section.cluster.topic,
        articles: [mainArticle, ...otherArticles],
        ...(section.cluster.display_title ? { displayTitle: section.cluster.display_title } : {}),
      }
    })
    .filter((cluster): cluster is FeedCluster => cluster !== null)
}

function normalizeTopicName(name: string): string {
  return name.trim().toLowerCase()
}

function findTopicItem(name: string) {
  const normalizedName = normalizeTopicName(name)
  return TOPIC_ITEMS.find((topic) => (
    normalizeTopicName(topic.label) === normalizedName
    || normalizeTopicName(topic.value) === normalizedName
    || normalizeTopicName(topic.id) === normalizedName
  ))
}

export function mapCategorySectionsToTopicSections(sections: HomepageClusterSection[]): TopicSection[] {
  const sectionsByTopicId = new Map<string, HomepageClusterSection[]>()

  sections.forEach((section) => {
    const topicName = section.category || section.cluster.topic
    const matchedTopic = findTopicItem(topicName)
    if (matchedTopic) {
      const existing = sectionsByTopicId.get(matchedTopic.id) ?? []
      existing.push(section)
      sectionsByTopicId.set(matchedTopic.id, existing)
    }
  })

  return HOMEPAGE_TOPICS.map((topic) => {
    const topicSections = sectionsByTopicId.get(topic.id) ?? []
    const articles: TopicArticle[] = topicSections.slice(0, 3).flatMap((topicSection) => {
      const sourceGroup = topicSection.sources[0]
      if (!sourceGroup) return []
      const article = sourceGroup.articles[0]
      if (!article) return []
      return [{
        id: article.id,
        clusterId: topicSection.cluster.id,
        source: {
          id: sourceGroup.source.id,
          name: sourceGroup.source.name,
          logo_url: sourceGroup.source.logo_url,
        },
        title: article.title,
        url: article.url,
        publishedAt: toRelativeTime(article.published_at, article.fetched_at),
        ...(article.image_url ? { imageUrl: article.image_url } : {}),
      } satisfies TopicArticle]
    })

    return {
      id: topic.id,
      name: topic.label,
      articles,
    } satisfies TopicSection
  })
}

export function mapCategorySectionsToTopicSectionsFromClusters(
  sections: HomepageClusterSection[],
  clusterDetailsById: Map<number, ClusterResponse>,
): TopicSection[] {
  const lightweightSections = mapCategorySectionsToTopicSections(sections)
  const sectionsByTopicId = new Map<string, HomepageClusterSection>()

  sections.forEach((section) => {
    const topicName = section.category || section.cluster.topic
    const matchedTopic = findTopicItem(topicName)
    if (matchedTopic && HOMEPAGE_TOPICS.some((topic) => topic.id === matchedTopic.id)) {
      sectionsByTopicId.set(matchedTopic.id, section)
    }
  })

  return lightweightSections.map((topic) => {
    const section = sectionsByTopicId.get(topic.id)
    if (!section) {
      return topic
    }

    const clusterDetails = clusterDetailsById.get(section.cluster.id)
    if (!clusterDetails) {
      return topic
    }

    return {
      ...topic,
      articles: mapClusterResponseArticles(clusterDetails.articles, 3) as TopicArticle[],
    }
  })
}

export function mapTopicClustersToTopicSections(
  clusterDetailsByTopicId: Map<string, ClusterResponse>,
): TopicSection[] {
  return HOMEPAGE_TOPICS.map((topic) => {
    const clusterDetails = clusterDetailsByTopicId.get(topic.id)

    return {
      id: topic.id,
      name: topic.label,
      articles: clusterDetails ? mapClusterResponseArticles(clusterDetails.articles, 3) as TopicArticle[] : [],
    }
  })
}
