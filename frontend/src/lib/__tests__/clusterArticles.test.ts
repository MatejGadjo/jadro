import { describe, expect, it } from 'vitest'
import { mapClusterResponseArticles, prioritizeClusterArticles, toRelativeTime } from '@/lib/clusterArticles'
import type { FeedArticle } from '@/lib/feedTypes'
import type { ClusterResponse } from '@/api/types'

const article = (id: number, imageUrl?: string): FeedArticle => ({
  id,
  source: { id: 1, name: 'Source' },
  title: `Article ${id}`,
  url: `https://example.com/${id}`,
  publishedAt: 'now',
  ...(imageUrl ? { imageUrl } : {}),
})

describe('prioritizeClusterArticles', () => {
  it('moves the first article with an image to the front', () => {
    const out = prioritizeClusterArticles([article(1), article(2, 'a.jpg'), article(3, 'b.jpg')])
    expect(out.map((a) => a.id)).toEqual([2, 1, 3])
  })

  it('keeps order when the lead already has an image or none do', () => {
    expect(prioritizeClusterArticles([article(1, 'a.jpg'), article(2)]).map((a) => a.id)).toEqual([1, 2])
    expect(prioritizeClusterArticles([article(1), article(2)]).map((a) => a.id)).toEqual([1, 2])
  })

  it('does not mutate its input', () => {
    const input = [article(1), article(2, 'a.jpg')]
    prioritizeClusterArticles(input)
    expect(input.map((a) => a.id)).toEqual([1, 2])
  })
})

describe('toRelativeTime', () => {
  it('falls back to fetchedAt and handles invalid dates', () => {
    expect(toRelativeTime(null, new Date().toISOString())).not.toBe('Recently')
    expect(toRelativeTime('not-a-date', 'also-not')).toBe('Recently')
  })
})

describe('mapClusterResponseArticles', () => {
  it('maps API articles, prioritises images and applies the limit', () => {
    const now = new Date().toISOString()
    const api = [1, 2, 3].map((id) => ({
      id,
      url: `https://example.com/${id}`,
      title: `T${id}`,
      snippet: '',
      published_at: now,
      fetched_at: now,
      image_url: id === 3 ? 'https://example.com/3.jpg' : undefined,
      source: { id: 9, name: 'Src', base_url: 'https://example.com', rss_url: 'https://example.com/feed' },
    })) as unknown as ClusterResponse['articles']

    const out = mapClusterResponseArticles(api, 2)
    expect(out).toHaveLength(2)
    expect(out[0].id).toBe(3)
    expect(out[0].imageUrl).toBe('https://example.com/3.jpg')
    expect(out[1]).not.toHaveProperty('imageUrl')
  })
})
