// View-model types shared across feed/topic UI. These are the shapes the
// presentational components consume after API responses are mapped (see
// `src/lib/clusterArticles.ts` and `src/lib/homepageData.ts`). They are
// intentionally decoupled from the raw API types in `src/api/types.ts`.

export interface FeedArticle {
  id: number
  source: {
    id: number
    name: string
    logo_url?: string
  }
  title: string
  url: string
  publishedAt: string
  author?: string
  imageUrl?: string
}

export interface FeedCluster {
  id: number
  headline: string
  displayTitle?: string
  articles: FeedArticle[]
  topic: string
}

export interface TopicArticle {
  id: number
  clusterId?: number
  source: {
    id: number
    name: string
    logo_url?: string
  }
  title: string
  url: string
  publishedAt: string
  author?: string
  imageUrl?: string
}

export interface TopicSection {
  id: string
  name: string
  articles: TopicArticle[]
}
