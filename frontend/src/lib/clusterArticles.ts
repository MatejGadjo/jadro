import { formatDistanceToNowStrict, parseISO } from 'date-fns'
import { mk } from 'date-fns/locale'
import type { ClusterResponse } from '@/api/types'
import type { MockArticle } from '@/lib/mockData'


export function toRelativeTime(publishedAt: string | null | undefined, fetchedAt: string): string {
  const candidate = publishedAt ?? fetchedAt
  const parsed = parseISO(candidate)
  if (Number.isNaN(parsed.getTime())) {
    return 'Recently'
  }
  return formatDistanceToNowStrict(parsed, { addSuffix: true, locale: mk })
}

export function prioritizeClusterArticles(
  articles: MockArticle[],
): MockArticle[] {
  if (articles.length === 0) {
    return articles
  }

  const orderedArticles = [...articles]
  const mainArticleIndex = orderedArticles.findIndex((article) => Boolean(article.imageUrl))
  if (mainArticleIndex > 0) {
    const [articleWithImage] = orderedArticles.splice(mainArticleIndex, 1)
    if (articleWithImage) {
      orderedArticles.unshift(articleWithImage)
    }
  }

  return orderedArticles
}

export function mapClusterResponseArticles(
  articles: ClusterResponse['articles'],
  limit?: number,
): MockArticle[] {
  const mappedArticles = articles.map((article) => ({
    id: article.id,
    source: {
      id: article.source.id,
      name: article.source.name,
      ...(article.source.logo_url ? { logo_url: article.source.logo_url } : {}),
    },
    title: article.title,
    url: article.url,
    publishedAt: toRelativeTime(article.published_at, article.fetched_at),
    ...(article.image_url ? { imageUrl: article.image_url } : {}),
  }))

  const orderedArticles = prioritizeClusterArticles(mappedArticles)
  return typeof limit === 'number' ? orderedArticles.slice(0, limit) : orderedArticles
}
